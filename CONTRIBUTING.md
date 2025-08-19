# How to contribute

We'd love to accept your patches and contributions to this project.

## Before you begin

### Sign our Contributor License Agreement

Contributions to this project must be accompanied by a
[Contributor License Agreement](https://cla.developers.google.com/about) (CLA).
You (or your employer) retain the copyright to your contribution; this simply
gives us permission to use and redistribute your contributions as part of the
project.

If you or your current employer have already signed the Google CLA (even if it
was for a different project), you probably don't need to do it again.

Visit <https://cla.developers.google.com/> to see your current agreements or to
sign a new one.

### Review our community guidelines

This project follows
[Google's Open Source Community Guidelines](https://opensource.google/conduct/).

## Releasing (Google team members only)

This section is for Google team members who are responsible for releasing new versions of the test server and SDKs.

### Releasing the `test-server` binary

This process creates a new GitHub release and attaches the compiled binaries.

#### Prerequisites

Make sure you have `goreleaser` installed:

```sh
go install github.com/goreleaser/goreleaser/v2@latest
```

#### Steps

1.  Ensure your local `main` branch is up-to-date and clean:
    ```sh
    git checkout main && git pull origin main && git clean -xdf
    ```
2.  Create and push a new version tag. For example, for version `v0.2.2`:
    ```sh
    git tag -a v0.2.2 -m "Release v0.2.2"
    git push origin v0.2.2
    ```
3.  Run GoReleaser:
    ```sh
    ~/go/bin/goreleaser release
    ```

    Note: This may fail with `error=missing GITHUB_TOKEN, GITLAB_TOKEN and GITEA_TOKEN`. To create a token, follow
    https://github.com/settings/tokens and set an environment variable `export GITHUB_TOKEN=<token>` before
    retrying the command.
5.  Verify that a new release with the updated binaries is available on the project's GitHub Releases page.

### Updating the Go release binary pin in the SDKs

After a new `test-server` binary is released, you need to update the checksums pinned in the SDKs.

1.  Run the `update-sdk-checksums` script with the new version tag. For example:
    ```sh
    go run scripts/update-sdk-checksums/main.go v0.2.2
    ```
    This updates the pinned checksums (currently only in the TypeScript SDK).
2.  Commit and push the changes. These changes will be included in the next SDK release. Example PR:
    https://github.com/google/test-server/pull/22

### Publishing the TypeScript SDK to npm

1.  Ensure your local `main` branch is up-to-date and clean:
    ```sh
    git checkout main && git pull origin main && git clean -xdf
    ```
2.  Navigate to the TypeScript SDK directory:
    ```sh
    cd sdks/typescript
    ```
3.  Update the `version` in `package.json` (e.g., using `npm version patch`).
4.  Commit and push the changes. Example PR: https://github.com/google/test-server/pull/23
5.  Install dependencies and build the SDK:
    ```sh
    npm ci && npm run build
    ```
6.  Publish the new version to npm following internal guidance at go/wombat-dressing-room. (When prompted,
    create a package specific publish token for  `test-server-sdk`.)
  
