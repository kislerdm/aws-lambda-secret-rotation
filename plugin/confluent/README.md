# Plugin for AWS Lambda to rotate Confluent Cloud API Key

[![logo](https://login-static.confluent.io/web/3.799.1/favicon.ico)](https://www.confluent.io/)

## Requirements

Secrets (see the [types definition](models.go)):

- _Secret Admin_ shall be compliant with the type `SecretAdmin`
- _Secret User_ shall be compliant with the type `SecretUser`

### `SecretUser`

A `map` with at least two attributes is expected as the secret to rotate:

- _API Key_: is expected to be denoted as "user" by default; can be overwritten via env. variable `ATTRIBUTE_KEY`;
- _API Secret_: is expected to be denoted as "password" by default; can be overwritten via env.
  variable `ATTRIBUTE_SECRET`.

Find details about the Confluent Cloud API
keys [here](https://docs.confluent.io/cloud/current/access-management/authenticate/api-keys/api-keys.html#use-api-keys-to-control-access-in-ccloud)
.

## AWS Lambda Configuration

The environment variable `ADMIN_SECRET_ARN` must contain the _Secret Admin_'
s [ARN](https://docs.aws.amazon.com/general/latest/gr/aws-arns-and-namespaces.html).

Optionally, the environment variable `DEBUG` can be set to "yes", or "true" to activate debug level logs.
