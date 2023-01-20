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

provider "neon" {}

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

resource "aws_secretsmanager_secret" "this" {
  name                    = "neon/mybranch/mydb/myrole"
  description             = "Neon SaaS access details for mydb, myrole @ mybranch"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "neon"
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
  secret_id     = aws_secretsmanager_secret.this.id
  secret_string = jsonencode({
    project_id = neon_project.this.id
    branch_id  = neon_branch.this.id
    dbname     = neon_database.this.name
    host       = neon_branch.this.host
    user       = neon_role.this.name
    password   = neon_role.this.password
  })
}

resource "aws_secretsmanager_secret" "admin" {
  name                    = "neon/admin"
  description             = "Neon admin API key"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "neon"
  }
}

data "external" "env" {
  program = ["${path.module}/../.env"]
}

resource "aws_secretsmanager_secret_version" "admin" {
  secret_id     = aws_secretsmanager_secret.admin.id
  secret_string = jsonencode({
    token = data.external.env.result["NEON_API_KEY"]
  })
}

data "aws_iam_policy_document" "lambda_neon" {
  statement {
    effect  = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]
    resources = ["arn:aws:logs:*:*:*"]
  }

  statement {
    effect  = "Allow"
    actions = [
      "secretsmanager:GetResourcePolicy",
      "secretsmanager:GetSecretValue",
      "secretsmanager:PutSecretValue",
      "secretsmanager:UpdateSecretVersionStage",
    ]
    resources = [aws_secretsmanager_secret.this.arn]
  }

  statement {
    effect  = "Allow"
    actions = [
      "secretsmanager:GetSecretValue",
    ]
    resources = [aws_secretsmanager_secret.admin.arn]
  }

  statement {
    effect  = "Allow"
    actions = [
      "secretsmanager:GetSecretValue",
    ]
    resources = [aws_secretsmanager_secret.admin.arn]
  }

  # if resources run in VPC
  #  statement {
  #    effect  = "Allow"
  #    actions = [
  #      "ec2:CreateNetworkInterface",
  #      "ec2:DeleteNetworkInterface",
  #      "ec2:DescribeNetworkInterfaces",
  #      "ec2:DetachNetworkInterface"
  #    ]
  #    resources = ["*"]
  #  }

  # if custom KMS key is used to encrypt the secret
  # https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets-required-permissions-function.html#rotating-secrets-required-permissions-function-cust-key-example
  #  statement {
  #    effect  = "Allow"
  #    actions = [
  #      "kms:Decrypt",
  #      "kms:GenerateDataKey"
  #    ]
  #    resources = ["KMSKeyARN"]
  #  }
}

resource "aws_iam_policy" "lambda_neon" {
  name   = "LambdaSecretRotation@neon-user"
  policy = data.aws_iam_policy_document.lambda_neon.json
}

locals {
  lambda_name = "neon-user"
  region      = "eu-west-1"
  account_id  = "564549078962"
}

resource "aws_iam_role" "this" {
  name = "secret-rotation@neon-user"

  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Action    = "sts:AssumeRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      },
    ]
  })
}

resource "aws_iam_role_policy_attachment" "lambda_neon" {
  policy_arn = aws_iam_policy.lambda_neon.arn
  role       = aws_iam_role.this.name
}

resource "aws_cloudwatch_log_group" "this" {
  name              = "/aws/lambda/${local.lambda_name}"
  retention_in_days = 1
}

resource "aws_lambda_function" "this" {
  function_name = local.lambda_name
  role          = aws_iam_role.this.arn

  filename         = "${path.module}/../bin/${local.lambda_name}.zip"
  source_code_hash = filebase64sha256("${path.module}/../bin/${local.lambda_name}.zip")
  runtime          = "go1.x"
  handler          = local.lambda_name
  memory_size      = 256

  environment {
    variables = {
      NEON_TOKEN_SECRET_ARN = aws_secretsmanager_secret.admin.arn
    }
  }
}

resource "aws_lambda_permission" "secretsmanager" {
  statement_id  = "AllowExecutionFromSecretsmanager"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.this.function_name
  principal     = "secretsmanager.amazonaws.com"
  source_arn    = aws_secretsmanager_secret.this.arn
}
