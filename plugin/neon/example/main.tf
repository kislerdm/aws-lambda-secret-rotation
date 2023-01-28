terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }

    neon = {
      source  = "kislerdm/neon"
      version = "0.1.0"
    }
  }
}

locals {
  plugin      = "neon"
  lambda_name = "${local.plugin}-key-rotation"
}

variable "neon_api_key" {
  type        = string
  sensitive   = true
  description = "Neon API KEY."
}

provider "neon" {
  api_key = var.neon_api_key
}

provider "aws" {
  region = "eu-west-1"
}

resource "neon_project" "this" {
  name = "myproject"
}

resource "neon_branch" "this" {
  project_id = neon_project.this.id
  name       = "mybranch"
}

resource "neon_role" "this" {
  project_id = neon_project.this.id
  branch_id  = neon_branch.this.id
  name       = "myrole"
}

resource "neon_database" "this" {
  project_id = neon_project.this.id
  branch_id  = neon_branch.this.id
  name       = "mydb"
  owner_name = neon_role.this.name
}

resource "aws_secretsmanager_secret" "admin" {
  name                    = "${local.plugin}/SecretAdmin"
  description             = "Neon admin API key"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "neon"
    type     = "admin"
  }
}

resource "aws_secretsmanager_secret_version" "admin" {
  secret_id = aws_secretsmanager_secret.admin.id
  secret_string = jsonencode({
    token = var.neon_api_key
  })
}

resource "aws_secretsmanager_secret" "this" {
  name                    = "${local.plugin}/SecretUser"
  description             = "Neon SaaS user's access details"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "neon"
    type     = "user"
  }
}

resource "aws_secretsmanager_secret_rotation" "this" {
  secret_id           = aws_secretsmanager_secret.this.id
  rotation_lambda_arn = aws_lambda_function.this.arn

  rotation_rules {
    automatically_after_days = 1
  }
}

resource "aws_secretsmanager_secret_version" "this" {
  secret_id = aws_secretsmanager_secret.this.id
  secret_string = jsonencode({
    project_id = neon_project.this.id
    branch_id  = neon_branch.this.id
    dbname     = neon_database.this.name
    host       = neon_branch.this.host
    user       = neon_role.this.name
    password   = neon_role.this.password
  })
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
        Resource = [aws_secretsmanager_secret.this.arn]
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
  statement_id  = "${local.lambda_name}-myrole"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.this.function_name
  principal     = "secretsmanager.amazonaws.com"
  source_arn    = aws_secretsmanager_secret.this.arn
}
