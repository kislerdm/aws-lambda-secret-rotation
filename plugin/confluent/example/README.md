# Confluent API Key Rotation with the AWS Lambda

## Prerequisites

- AWS Account with an [access key](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html)
- [Confluent Cloud](https://www.confluent.io/) account with a
  Cloud [API key](https://docs.confluent.io/cloud/current/api.html)
- [terraform](https://www.terraform.io/) ~> 1.3.3
- [go](https://go.dev) ~> 1.19
- [gnuMake](https://www.gnu.org/software/make/)

## How to run

1. Export the AWS access key-secret pair to the shall environment:

```commandline
export AWS_ACCESS_KEY_ID=##KeyID##
export AWS_SECRET_ACCESS_KEY=##KeySecret##
```

_**Example**_

If awscli is used and an AWS profile is configured, run the following command:

```commandline
 export AWS_ACCESS_KEY_ID=$(aws configure get aws_access_key_id --profile ##ProfileName##)
 export AWS_SECRET_ACCESS_KEY=$(aws configure get aws_secret_access_key --profile ##ProfileName##)
```

**_Note_** that [awscli](https://aws.amazon.com/cli/) is required.

2. Export Confluent key-secret pair to the shell environment:

```commandline
TF_VAR_confluent_key_id=##ConfluentAPIKey##
TF_VAR_confluent_secret=##ConfluentAPISecret##
```

3. Export required Confluent cluster's attributes

```commandline
export TF_VAR_environment_id=##ConfluentEnvironment##
export TF_VAR_cluster_id=##KafkaClusterID##
export TF_VAR_kafka_boostrap_server=##KafkaBootstrapServerHost##
```

4. Run terraform init:

```commandline
terraform init
```

5. Run terraform validate:

```commandline
terraform validate
```

6. Run terraform plan:

```commandline
terraform plan
```

7. Run terraform apply:

```commandline
terraform apply -auto-approve
```

**_Note_** that the terraform state will be stored locally, hence secure its access to prevent leak of secrets.

Clean the resources upon completion of the tests:

```commandline
terraform destroy -auto-approve
rm -rf terraform.tfstate* .terraform .terraform.lock.hcl
```

## References

- [AWS Access Key](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html)
- [Confluent API](https://docs.confluent.io/cloud/current/api.html)
- [Confluent Environments](https://docs.confluent.io/cloud/current/access-management/hierarchy/cloud-environments.html)
- [Kafka bootstrapServer](https://kafka.apache.org/documentation/#producerconfigs_bootstrap.servers)

## Terraform Module

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~> 4.51.0 |
| <a name="requirement_confluent"></a> [confluent](#requirement\_confluent) | ~> 1.25.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.51.0 |
| <a name="provider_confluent"></a> [confluent](#provider\_confluent) | 1.25.0 |
| <a name="provider_local"></a> [local](#provider\_local) | 2.3.0 |
| <a name="provider_null"></a> [null](#provider\_null) | 3.2.1 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [aws_cloudwatch_log_group.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/cloudwatch_log_group) | resource |
| [aws_iam_policy.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_policy) | resource |
| [aws_iam_role.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role) | resource |
| [aws_iam_role_policy_attachment.lambda_neon](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy_attachment) | resource |
| [aws_lambda_function.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_function) | resource |
| [aws_lambda_permission.secretsmanager](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/lambda_permission) | resource |
| [aws_secretsmanager_secret.admin](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret) | resource |
| [aws_secretsmanager_secret_rotation.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_rotation) | resource |
| [aws_secretsmanager_secret_version.admin](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_version) | resource |
| [aws_secretsmanager_secret_version.this](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_version) | resource |
| [confluent_api_key.this](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/api_key) | resource |
| [confluent_service_account.this](https://registry.terraform.io/providers/confluentinc/confluent/latest/docs/resources/service_account) | resource |
| [null_resource.this](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [local_file.this](https://registry.terraform.io/providers/hashicorp/local/latest/docs/data-sources/file) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_id"></a> [cluster\_id](#input\_cluster\_id) | Confluent Kafka Cluster ID. | `string` | n/a | yes |
| <a name="input_confluent_key_id"></a> [confluent\_key\_id](#input\_confluent\_key\_id) | Confluent API KEY. | `string` | n/a | yes |
| <a name="input_confluent_secret"></a> [confluent\_secret](#input\_confluent\_secret) | Confluent API Secret. | `string` | n/a | yes |
| <a name="input_environment_id"></a> [environment\_id](#input\_environment\_id) | Confluent Environment ID. | `string` | n/a | yes |
| <a name="input_kafka_boostrap_server"></a> [kafka\_boostrap\_server](#input\_kafka\_boostrap\_server) | Kafka bootstrap server. | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->