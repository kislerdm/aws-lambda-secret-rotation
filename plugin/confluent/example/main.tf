terraform {
  required_providers {
    confluent = {
      source  = "confluentinc/confluent"
      version = "~> 1.25.0"
    }
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.51.0"
    }
  }
}

variable "confluent_key_id" {
  type        = string
  sensitive   = true
  description = "Confluent API KEY."
}

variable "confluent_secret" {
  type        = string
  sensitive   = true
  description = "Confluent API Secret."
}

variable "cluster_id" {
  type        = string
  description = "Confluent Kafka Cluster ID."
}

variable "environment_id" {
  type        = string
  description = "Confluent Environment ID."
}

variable "kafka_boostrap_server" {
  type        = string
  description = "Kafka bootstrap server."
}

provider "confluent" {
  cloud_api_key    = var.confluent_key_id
  cloud_api_secret = var.confluent_secret
}

locals {
  plugin      = "confluent"
  lambda_name = "${local.plugin}-key-rotation"

  sa = { "foo-bar" : "" }
}

resource "confluent_service_account" "this" {
  for_each     = local.sa
  display_name = each.key
  description  = "SA for ${each.key}"
}

resource "confluent_api_key" "this" {
  for_each     = local.sa
  display_name = "${var.cluster_id}:${each.key}"
  description  = "API key for ${each.key}"

  owner {
    id          = confluent_service_account.this[each.key].id
    kind        = confluent_service_account.this[each.key].kind
    api_version = "iam/v2"
  }

  managed_resource {
    id          = var.cluster_id
    kind        = "Cluster"
    api_version = "cmk/v2"
    environment {
      id = var.environment_id
    }
  }

  #  lifecycle {
  #    prevent_destroy = true
  #  }
}

resource "aws_secretsmanager_secret" "admin" {
  name                    = "${local.plugin}/SecretAdmin"
  description             = "Confluent admin API key"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "confluent"
    type     = "admin"
  }
}

resource "aws_secretsmanager_secret_version" "admin" {
  secret_id = aws_secretsmanager_secret.admin.id
  secret_string = jsonencode({
    cloud_api_key    = var.confluent_key_id
    cloud_api_secret = var.confluent_secret
  })
}

resource "aws_secretsmanager_secret" "this" {
  for_each                = local.sa
  name                    = "${local.plugin}/SecretUser/${each.key}"
  description             = "Confluent credentials for SA ${each.key}"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "confluent"
    type     = "user"
  }
}

resource "aws_secretsmanager_secret_version" "this" {
  for_each  = local.sa
  secret_id = aws_secretsmanager_secret.this[each.key].id
  secret_string = jsonencode({
    user             = confluent_api_key.this[each.key].id
    password         = confluent_api_key.this[each.key].secret
    bootstrap_server = var.kafka_boostrap_server
  })
}

resource "aws_secretsmanager_secret_rotation" "this" {
  for_each            = local.sa
  secret_id           = aws_secretsmanager_secret.this[each.key].id
  rotation_lambda_arn = aws_lambda_function.this.arn
  rotation_rules {
    automatically_after_days = 1
  }
}

#### Lambda

resource "aws_iam_policy" "this" {
  name = "LambdaSecretRotation@${local.plugin}"
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = concat([
      {
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
        ]
        Resource = ["arn:aws:logs:*:*:*"]
      },
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetResourcePolicy",
          "secretsmanager:GetSecretValue",
          "secretsmanager:PutSecretValue",
          "secretsmanager:UpdateSecretVersionStage",
          "secretsmanager:DescribeSecret",
        ]
        Resource = [for i in aws_secretsmanager_secret.this : i.arn]
      },
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = [aws_secretsmanager_secret.admin.arn]
      },
      ]
    )
  })
}

resource "aws_iam_role" "this" {
  name = "secret-rotation@${local.plugin}"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_neon" {
  policy_arn = aws_iam_policy.this.arn
  role       = aws_iam_role.this.name
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/${local.plugin}"
  retention_in_days = 1
}

resource "null_resource" "this" {
  triggers = {
    md5 = join(",", [
      for file in concat(
        [for f in fileset("${path.module}/../../../", "{*.go,go.mod,go.sum}") : "${path.module}/../../../${f}"],
        [for f in fileset("${path.module}/../../../plugin/${local.plugin}", "{*.go,go.mod,go.sum}") : "${path.module}/../../../plugin/${local.plugin}/${f}"],
        [for f in fileset("${path.module}/../../../plugin/${local.plugin}/cmd/lambda/", "*.go") : "${path.module}/../../../plugin/${local.plugin}/cmd/lambda/${f}"],
      ) : filemd5(file)
    ])
  }

  provisioner "local-exec" {
    command = "cd ${path.module}/../../.. && make build PLUGIN=${local.plugin} TAG=local"
  }
}

data "local_file" "this" {
  filename   = "${path.module}/../../../bin/${local.plugin}/aws-lambda-secret-rotation_${local.plugin}_local.zip"
  depends_on = [null_resource.this]
}

resource "aws_lambda_function" "this" {
  function_name = local.lambda_name
  role          = aws_iam_role.this.arn

  filename         = data.local_file.this.filename
  source_code_hash = base64sha256(data.local_file.this.content_base64)
  runtime          = "go1.x"
  handler          = "lambda"
  memory_size      = 256
  timeout          = 30

  environment {
    variables = {
      ADMIN_SECRET_ARN = aws_secretsmanager_secret.admin.arn
      DEBUG            = "true"
    }
  }

  depends_on = [null_resource.this]
}

resource "aws_lambda_permission" "secretsmanager" {
  for_each      = local.sa
  statement_id  = "${local.lambda_name}-${each.key}"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.this.function_name
  principal     = "secretsmanager.amazonaws.com"
  source_arn    = aws_secretsmanager_secret.this[each.key].arn
}
