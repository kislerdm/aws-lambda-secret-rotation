# AWS Lambda to rotate Secret in AWS Secretsmanager

AWS Lambda function
to [rotate](https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotating-secrets.html) secret's version, e.g.
database access credentials, stored in [AWS Secretsmanager](https://aws.amazon.com/secrets-manager/).

## How it works

<figure style="alignment: center;">
<img style="alignment: center;" src="architecture.svg" alt="architecture-c4-containers">
<figcaption style="alignment: center;">[C4 Container] Architecture Diagram.</figcaption>
</figure>

The diagram illustrates the logic realised in two interfaces:

- `SecretsmanagerClient` to communicate with the secrets vault, i.e. AWS Secretsmanager;
- `ServiceClient` to communicate with the system delegated credentials storage to the vault system. The client uses the
  secret "_Secret Admin_" to pass authentication and authorization in order to reset the credentials "_Secret User_".

Upon invocation, the AWS Lambda logic executes the
following [steps](https://docs.aws.amazon.com/secretsmanager/latest/userguide/rotate-secrets_turn-on-for-other.html#rotate-secrets_turn-on-for-other_step5):

1. _Create Secret_: new version of the "Secret User" secret is generated and stored in the staging label _AWSPENDING_;
2. _Set Secret_: newly generated secret's version is set in the "System deleted Access Management";
3. _Test Secret_: newly generated secret's version is tested against the "System deleted Access Management";
4. _Finish Secret_: newly generated secret's version is moved from the stage _AWSPENDING_ to _AWSCURRENT_.

The Lambda process undergoes authentication and authorization using the "_Secret Admin_" access credentials to perform
the steps.

## Plugin structure

## Contribution

The codebase is distributed under the [MIT license](LICENSE). Please feel free to open an issue ticket, or PR to
contribute.

## Development

### Requirements

- [go](https://go.dev) ~> 1.19
- [gnuMake](https://www.gnu.org/software/make/)

### Commands

Run to see available commands:

```commandline
make help
```

Run to test the `lambda` module:

```commandline
make tests
```

Run to test a plugin module:

```commandline
make test-plugin PLUGIN=##name-of-the-plugin##
```

For example, to run unit tests for the Neon plugin:

```commandline
make test-plugin PLUGIN=neon
```

Run to build lambda binary for selected plugin:

```commandline
make build PLUGIN=##name-of-the-plugin##
```

For example, to run unit tests for the Neon plugin:

```commandline
make build PLUGIN=neon
```
