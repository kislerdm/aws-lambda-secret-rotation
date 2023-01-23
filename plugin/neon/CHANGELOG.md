# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [plugin/neon/v0.1.1] - 2023-01-23

### Fix

- Attached pre-built binaries to be deployed as AWS Lambda 

### Changed

- Fixed changelog's header tagging

## [plugin/neon/v0.1.0] - 2023-01-22

### Added

- Implements `ServiceClient` interface to communicate with Neon SaaS to
  rotate [user](https://neon.tech/docs/manage/users/)'s access password:

```go
package main

import (
	serviceClient "github.com/kislerdm/aws-lambda-secret-rotation/plugin/neon"
	sdk "github.com/kislerdm/neon-sdk-go"
)

func main() {
	neonSDK, err := sdk.NewClient(sdk.WithAPIKey("foobarbazqux"))
	if err != nil {
		panic(err)
	}

	_ = serviceClient.NewServiceClient(neonSDK)
}
```
