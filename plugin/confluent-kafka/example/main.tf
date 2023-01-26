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
  description = "Confluent API KEY"
}

variable "confluent_secret" {
  type        = string
  sensitive   = true
  description = "Confluent API Secret"
}

provider "confluent" {
  cloud_api_key    = var.confluent_key_id
  cloud_api_secret = var.confluent_secret
}

locals {
  cluster_id          = "lkc-3xpgj"
  environment_id      = "env-yny5j"
  bootstrap_server    = "pkc-4r297.europe-west1.gcp.confluent.cloud:9092"
  kafka_rest_endpoint = "https://pkc-4r297.europe-west1.gcp.confluent.cloud:443"

  sa = { "foo-bar" : "" }
}

resource "confluent_service_account" "this" {
  for_each     = local.sa
  display_name = each.key
  description  = "SA for ${each.key}"
}

resource "confluent_api_key" "this" {
  for_each     = local.sa
  display_name = "${local.cluster_id}:${each.key}"
  description  = "API key for ${each.key}"

  owner {
    id          = confluent_service_account.this[each.key].id
    kind        = confluent_service_account.this[each.key].kind
    api_version = "iam/v2"
  }

  managed_resource {
    id          = local.cluster_id
    kind        = "Cluster"
    api_version = "cmk/v2"
    environment {
      id = local.environment_id
    }
  }

  lifecycle {
    prevent_destroy = true
  }
}

resource "aws_secretsmanager_secret" "admin" {
  name                    = "confluent/admin"
  description             = "Confluent admin API key"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "confluent"
    type     = "admin"
  }
}

resource "aws_secretsmanager_secret_version" "admin" {
  secret_id     = aws_secretsmanager_secret.admin.id
  secret_string = jsonencode({
    cloud_api_key    = var.confluent_key_id
    cloud_api_secret = var.confluent_secret
  })
}

resource "aws_secretsmanager_secret" "this" {
  for_each                = local.sa
  name                    = "confluent/${each.key}"
  description             = "Confluent credentials for SA ${each.key}"
  recovery_window_in_days = 0

  tags = {
    project  = "demo"
    platform = "confluent"
    type     = "user"
  }
}

resource "aws_secretsmanager_secret_version" "this" {
  for_each      = local.sa
  secret_id     = aws_secretsmanager_secret.this[each.key].id
  secret_string = jsonencode({
    user             = confluent_api_key.this[each.key].id
    password         = confluent_api_key.this[each.key].secret
    cluster_id       = local.cluster_id
    bootstrap_server = local.bootstrap_server
    user_id          = confluent_service_account.this[each.key].id
  })
}
