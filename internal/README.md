# Development Guide: Testing Local test-server Changes

## Overview
This guide outlines the standard procedure for compiling and testing local modifications to the Go test-server. The test-server is a Go application distributed via language-specific SDK wrappers for ease of adoption.

The recommended workflow for end-to-end validation is to test through the SDK sample. This ensures that any changes to the core Go binary are compatible with the SDK integration layer.

## Step-by-Step e2e Testing Workflow with TypeScript SDK

The process involves compiling your local Go source code, preparing the TypeScript environment, and then linking your new binary to the sample project for testing. 

> NOTE: The steps mentioned below assumes that you have never run the Typescript Sample before or haven't run the Typescript Sample for a while. If your Typescript Sample can compile and run, and your change only related to the go-server, you only need step 1 and step 4 to use your local build.

**Step 1: Compile the Go Binary**

First, compile your local changes into an executable test-server binary.

Navigate to the project's root directory (`test-server`) and build the Go application.

```sh
go build
```

After a successful build, a new executable named `test-server` will be present in the root directory.

**Step 2: Build the TypeScript SDK Wrapper**

The TypeScript SDK acts as a wrapper around the Go binary. It must be built before we can use it in the TypeScript samples.

Navigate to the TypeScript SDK directory: `test-server/sdks/typescript`.

Install dependencies and build the SDK package.

```sh
npm install
npm run build
```

**Step 3: Prepare the Sample**

The sample project consumes the TypeScript SDK. We will set it up to run our tests.

Navigate to the sample project directory: `test-server/sdks/typescript/samples`.

Install its dependencies. This command installs the test-server-sdk we just built in step 2.

```sh
# ensure you have have a clean installation.
rm -Rf node_modules 
npm install
```

**Step 4: Link Your Local test-server Binary**

This is the most critical step. You must replace the pre-packaged binary in the sample project's dependencies with the custom binary you built in Step 1.

After the previous steps, the directory structure looks like this. The goal is to replace the test-server binary highlighted below, the graph below is a tree view of your directory structure from `test-server/sdks/typescript/samples`.

```diff
├── dist
│   └── specs
│   ├── sample.spec.js
│   └── sample.spec.js.map
├── jasmine.json
├── node_modules
│   ├── ... (other packages hidden)
│   ├── test-server-sdk
│   │   ├── bin
│   │   │   ├── CHANGELOG.md
│   │   │   ├── LICENSE
│   │   │   ├── README.md
+│  │   │   └── test-server
│   │   ├── checksums.json
│   │   ├── LICENSE
│   │   ├── package.json
│   │   └── postinstall.js
│   └── ... (other packages hidden)
├── package.json
├── package-lock.json
├── src
│   └── specs
│   └── sample.spec.ts
├── test-data
│   ├── config
│   │   └── test-server-config.yml
│   └── recordings
└── tsconfig.json

```

From the `test-server/sdks/typescript/samples` directory, run the following command to move your freshly compiled binary into place, overwriting the old one.

```sh

# Move the test-server binary from the project root to the samples' node_modules directory.
mv ../../../test-server ./node_modules/test-server-sdk/bin/test-server
```

**Step 5: Run the Integration Tests**

You are now ready to run the sample project's test suite against your local test-server build.

From the `test-server/sdks/typescript/samples` directory:

To run tests in playback mode:

```sh
#To run tests in replay mode:
npm run test
#To run tests in record mode (and regenerate test recordings):
npm run test:record
```

Any output or errors from the test run will now reflect the behavior of your local test-server changes.

You should also see the actual output like this:

```
[test-server-sdk] Starting test-server in 'replay' mode. Command: /usr/local/google/home/test-server/sdks/typescript/sample/node_modules/test-server-sdk/bin/test-server replay --config /usr/local/google/home/wanlindu/test-server/sdks/typescript/sample/test-data/config/test-server-config.yml --recording-dir /usr/local/google/home/wanlindu/test-server/sdks/typescript/sample/test-data/recordings
```
You can cross check if the binary used is what you provided.