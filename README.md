# AWS Lambda to rotate Secret in AWS Secretsmanager

AWS Lambda function
to [rotate](https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets.html) secrets, e.g. database access credentials, stored in AWS Secretsmanager.

## Architecture - C4 Containers Diagram

![architecture](architecture.svg)

## Development

### Requirements

- go ~> 1.19
- Docker
- gnuMake
