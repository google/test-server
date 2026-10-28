#!/usr/bin/env bash
# OSS Exit Gate Release Script for D12: google/test-server (test-server/npm — b/576682049, b/576678670)
set -euo pipefail

EXIT_GATE_AR_URL="https://us-npm.pkg.dev/oss-exit-gate-prod/test-server--npm/"
EXIT_GATE_MANIFEST_GCS="gs://oss-exit-gate-prod-projects-bucket/test-server/npm/manifests"

npm ci
npm run build

ACCESS_TOKEN="$(gcloud auth print-access-token)"
NPMRC_TMP="$(mktemp)"
cat << EOF > "${NPMRC_TMP}"
registry=${EXIT_GATE_AR_URL}
//us-npm.pkg.dev/oss-exit-gate-prod/test-server--npm/:always-auth=true
//us-npm.pkg.dev/oss-exit-gate-prod/test-server--npm/:_authToken=${ACCESS_TOKEN}
EOF
trap 'rm -f "${NPMRC_TMP}"' EXIT

VERSION="$(jq -r .version package.json)"
echo "1. Staging test-server-sdk@${VERSION} to ${EXIT_GATE_AR_URL}..."
npm publish --userconfig="${NPMRC_TMP}" --registry="${EXIT_GATE_AR_URL}"

echo "2. Uploading OSS Exit Gate manifest to ${EXIT_GATE_MANIFEST_GCS}..."
MANIFEST_FILE="$(mktemp)"
cat << EOF > "${MANIFEST_FILE}"
{
  "publish_all": false,
  "publishing_groups": [
    {
      "packages": [
        {
          "name": "test-server-sdk",
          "version": "${VERSION}"
        }
      ]
    }
  ]
}
EOF

gcloud storage cp "${MANIFEST_FILE}" \
  "${EXIT_GATE_MANIFEST_GCS}/manifest-test-server-sdk-${VERSION}-$(date +%s).json"
rm -f "${MANIFEST_FILE}"
echo "Done! OSS Exit Gate release triggered for test-server-sdk@${VERSION}."
