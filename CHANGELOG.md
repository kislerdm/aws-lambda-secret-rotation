# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v0.1.1] - 2023-01-25

### Changed

- Changed the module's façade API:
    - **now**: The main function `NewHandler(cfg Config)` is used to initialise the Lambda handler used as the input
      to [`lambda.Start`](https://github.com/aws/aws-lambda-go/blob/0d45ea2853e8fa138a242336f40eadf5f66fe947/lambda/entry.go#L44)
      function to initialize AWS Lambda
    - **was**: The main function `Start(cfg Config)` wraps the
      function [`lambda.Start`](https://github.com/aws/aws-lambda-go/blob/0d45ea2853e8fa138a242336f40eadf5f66fe947/lambda/entry.go#L44)
      to initialize AWS Lambda

An example:

```go
package main

import (
	"log"
	"os"
	
	"github.com/aws/aws-lambda-go/lambda"
	
	secretRotation "github.com/kislerdm/aws-lambda-secret-rotation"
)

func main()  {
	/* ... */
	handler, err := secretRotation.NewHandler(
		secretRotation.Config{
			/* ... */
			Debug: secretRotation.StrToBool(os.Getenv("DEBUG")),
		},
	)
	if err != nil {
		log.Fatalf("unable to init lambda handler to rotate secret, %v", err)
	}

	lambda.Start(handler)
}
```

## [v0.1.0] - 2023-01-22

### Added

- Lambda handler
- Interfaces for clients to communicate with the AWS Secretsmanager and the service delegated secrets storage
