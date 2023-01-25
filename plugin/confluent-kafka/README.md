# Plugin for AWS Lambda to rotate Confluent Cloud Kafka credentials

[![logo](https://login-static.confluent.io/web/3.799.1/favicon.ico)](https://www.confluent.io/)

[![Go Report Card](https://goreportcard.com/badge/github.com/kislerdm/aws-lambda-secret-rotation/plugin/confluent-kafka)](https://goreportcard.com/report/github.com/kislerdm/aws-lambda-secret-rotation/plugin/confluent-kafka)

## Requirements

Secrets (see the [types definition](models.go)):

- _Secret Admin_ shall be compliant with the type `SecretAdmin`
- _Secret User_ shall be compliant with the type `SecretUser`

## AWS Lambda Configuration

The environment variable `ADMIN_SECRET_ARN` must contain the _Secret Admin_'
s [ARN](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html).

Optionally, the environment variable `DEBUG` can be set to "yes", or "true" to activate debug level logs.
