# Test-server

A lightweight record-replay reverse proxy for software testing.

## Usage

Create a configuration file for your test server, example:

```yml
endpoints:
  - target_host: generativelanguage.googleapis.com 
    target_type: https
    target_port: 443
    source_type: http
    source_port: 1443
    redact_request_headers:
      - X-Goog-Api-Key
      - Authorization
  - target_host: us-central1-aiplatform.googleapis.com
    target_type: https
    target_port: 443
    source_type: http
    source_port: 1444
    redact_request_headers:
      - X-Goog-Api-Key
      - Authorization

```

The configuration above specifies that test-server will be providing a testing endpoint for https://generativelanguage.googleapis.com:443 on http://localhost:1443  And a testing endpoint for https://us-central1-aiplatform.googleapis.com:443 on http://localhost:1444

The configuration also specifies that the `X-Goog-Api-Key` and `Authorization` http headers will be redacted from the recordings for both endpoints.


### Running in record mode

To start test-server in record mode invoke:

```sh
test-server record --config <CONFIG_FILE> --recording-dir <RECORDING_DIR>
```

This runs test-server as a reverse proxy, with all interactions being saved to files under <RECORDING_DIR>.


### Running in replay mode

To start test-server in replay mode invoke:

```sh
test-server replay --config <CONFIG_FILE> --recording-dir <RECORDING_DIR>
```

This will have test-server listen on the local endpoints and respond to requests with the recorded responses.
Requests that were not recorded will be answered with an internal server error.


## Implementation

This library is implemented as a Go Binary that can be run as a standalone executable.


### SDKs

This repository also defines a TypeScript SDK which wraps around the Go Binary and provides a usability layer for TS projects by introducing convenience methods to start and stop the test-server.
