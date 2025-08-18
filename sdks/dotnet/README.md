This folder contains the .NET SDK for `test-server`. It provides a small runtime wrapper to start/stop the `test-server` binary and a helper installer that downloads and verifies the native binary.

Quick start
1. Build the library and the installer tool:

```bash
cd sdks/dotnet
dotnet build
```

2. Install the native `test-server` binary (download + verify):

```bash
dotnet run --project tools/installer -- v0.2.5
```

What the installer does
- Downloads the release archive for the specified `v*` version from GitHub releases.
- Verifies the SHA-256 checksum against the embedded `checksums.json` (or falls back to `sdks/typescript/checksums.json` in the repo).
- Extracts the binary into `sdks/dotnet/bin/` and sets executable permissions on Unix.

Using the SDK in your project
- Reference the produced NuGet package (`TestServerSdk.0.1.0.nupkg`) or add a project reference to this library.
- The SDK now requires callers to provide an explicit `BinaryPath` when constructing `TestServerOptions`. Example of `TestServerOptions`:

```csharp
using TestServerSdk;

var binaryPathDir = "dir/you/want/the/binary/to/be/downloaded";

var options = new TestServerOptions
{
    BinaryPath = Path.FullPath(Path.Combine(binaryPathDir, "test-server"))
};

var server = new TestServerProcess(options);
```

Packaging
- Building with `GeneratePackageOnBuild` enabled will produce `bin/Debug/TestServerSdk.0.1.0.nupkg`.

Notes
- If you plan to publish the .NET SDK independently, consider embedding or publishing checksums with the package so the installer remains self-contained.
- The SDK uses `YamlDotNet` for parsing test-server YAML configs and `SharpCompress` for archive extraction.
