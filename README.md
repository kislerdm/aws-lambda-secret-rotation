# AWS Lambda to rotate Neon Database Access Details stored in AWS Secretsmanager

AWS Lambda function
to [rotate](https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets.html) [Neon](https://neon.tech/)
role's access credentials stored in AWS Secretsmanager.

## Architecture - C4 Containers Diagram

![architecture](architecture.svg)

## Development

### Requirements

- go ~> 1.19
- Docker
- gnuMake
