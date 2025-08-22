# Usage samples for `@google/genai/node`

To run the samples first build the SDKs, from the sdks/typescript/:

```sh
# Build the SDK
npm install
npm run build
```

Then `cd sdks/typescript/samples`, install the newly built test-server typescript sdk:

```sh
# install the test-server typescript sdk
cd samples
rm -Rf node_modules # To clean up the old dependencies
npm install
```

Now from sdks/typescript/samples, you can run 
```sh
rm -Rf dist # To clean up the old build

npm run test
#or
npm run test:record
```