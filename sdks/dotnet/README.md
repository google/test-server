This folder contains the .NET SDK for `test-server`. It provides a small runtime wrapper to start/stop the `test-server` binary and a helper installer that downloads and verifies the native binary. During test runtime, the SDK first checks if the `test-server` binary is already downloaded and verified, otherwise it downloads and verifies it.

## Example of setting TestServerOptions

```csharp
using TestServerSdk;

var binaryPathDir = "dir/you/want/the/binary/to/be/downloaded";
var options = new TestServerOptions
{
  ConfigPath = Path.GetFullPath("../test-server.yml"),
  RecordingDir = Path.GetFullPath("../Recordings"),
  Mode = "replay",
  BinaryPath = Path.GetFullPath(Path.Combine(binaryPathDir, "test-server"))
};
```
