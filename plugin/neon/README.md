# Plugin for AWS Lambda to reset Neon Role's Password

[![logo](https://neon.tech/static/logo-white-9b5ae00331360361ba068980af7383ba.svg)](https://neon.tech)

[![Go Report Card](https://goreportcard.com/badge/github.com/kislerdm/aws-lambda-secret-rotation/plugin/neon)](https://goreportcard.com/report/github.com/kislerdm/aws-lambda-secret-rotation/plugin/neon)

> Neon is a fully managed serverless PostgreSQL with a generous free tier. Neon separates storage and compute and offers
> modern developer features such as serverless, branching, bottomless storage, and more. Neon is open source and written
> in Rust.

Find more about Neon [here](https://neon.tech/docs/introduction/about/).

## Requirements

Secrets (see the [types definition](models.go)):

- _Secret Admin_ shall be compliant with the type `SecretAdmin`
- _Secret User_ shall be compliant with the type `SecretUser`

## AWS Lambda Configuration

The environment variable `NEON_TOKEN_SECRET_ARN` must contain the _Secret Admin_'
s [ARN](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html).

Optionally, the environment variable `DEBUG` can be set to "yes", or "true" to activate debug level logs.
