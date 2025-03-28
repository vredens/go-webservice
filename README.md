# Web package

Helpers for WebServices using Go's net/http and github.com/labstack/echo packages.

## Features

*Server*

- Gzip enabled by default
- Simplified middleware builders for metrics and access logs
- Error logs for operational errors or request handling errors. Also supports setting a custom error log handler.

*Client*

- Automatically set the user agent to identify callers. Uses env vars or user information to construct the user agent string
	- `go-webservice/v0/{USER_AGENT}`
	- `go-webservice/v0/{SYSTEM}/{COMPONENT}`
	- `go-webservice/v0/{USER_INFO}`
- Default headers.
- Default timeouts on requests with support for setting the default timeout or on a per request basis.
- Request builder.
- Supports context.
- Simplified response handling with body closed before returning data to the caller.

## Examples

Check out the code [examples](./example_test.go).
