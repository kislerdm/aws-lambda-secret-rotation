# Neon User Credentials Rotation with the AWS Lambda

## Prerequisites

- AWS Account with an [access key](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html)
- [Neon](https://neon.tech/) account with an [API key](https://neon.tech/docs/manage/api-keys)
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
TF_VAR_neon_key_id=##NeonAPIKey##
```

3. Run terraform init:

```commandline
terraform init
```

4. Run terraform validate:

```commandline
terraform validate
```

5. Run terraform plan:

```commandline
terraform plan
```

6. Run terraform apply:

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
- [Neon API](https://neon.tech/docs/reference/api-reference)

## Terraform Module

<!-- BEGIN_TF_DOCS -->
## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_aws"></a> [aws](#requirement\_aws) | ~> 4.0 |
| <a name="requirement_neon"></a> [neon](#requirement\_neon) | 0.1.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_aws"></a> [aws](#provider\_aws) | 4.52.0 |
| <a name="provider_local"></a> [local](#provider\_local) | 2.3.0 |
| <a name="provider_neon"></a> [neon](#provider\_neon) | 0.1.0 |
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
| [neon_branch.this](https://registry.terraform.io/providers/kislerdm/neon/0.1.0/docs/resources/branch) | resource |
| [neon_database.this](https://registry.terraform.io/providers/kislerdm/neon/0.1.0/docs/resources/database) | resource |
| [neon_project.this](https://registry.terraform.io/providers/kislerdm/neon/0.1.0/docs/resources/project) | resource |
| [neon_role.this](https://registry.terraform.io/providers/kislerdm/neon/0.1.0/docs/resources/role) | resource |
| [null_resource.this](https://registry.terraform.io/providers/hashicorp/null/latest/docs/resources/resource) | resource |
| [local_file.this](https://registry.terraform.io/providers/hashicorp/local/latest/docs/data-sources/file) | data source |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_neon_api_key"></a> [neon\_api\_key](#input\_neon\_api\_key) | Neon API KEY. | `string` | n/a | yes |

## Outputs

No outputs.
<!-- END_TF_DOCS -->