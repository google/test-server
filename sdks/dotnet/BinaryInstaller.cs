/*
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

using System;
using System.IO;
using System.Net.Http;
using System.Security.Cryptography;
using System.Text.Json;
using System.Threading.Tasks;
using System.Runtime.InteropServices;
using System.Diagnostics;
using SharpCompress.Readers;
using SharpCompress.Common;
using System.Reflection;

namespace TestServerSdk
{
  public static class BinaryInstaller
  {
    private const string GithubOwner = "google";
    private const string GithubRepo = "test-server";
    private const string ProjectName = "test-server";

    /// <summary>
    /// Ensures the test-server binary for the given version is present in the specified output directory.
    /// It will download the release asset from GitHub, verify its SHA256 checksum, extract it, and set executable permissions.
    /// The checksums are read from a 'checksums.json' file expected to be embeded into the TestServerSdk.dll.
    /// </summary>
    public static async Task EnsureBinaryAsync(string outDir, string version = "v0.2.7")
    {
      var assembly = Assembly.GetExecutingAssembly();
      var resourceName = "TestServerSdk.checksums.json";
      string checksumsJson;

      Console.WriteLine($"[SDK] Attempting to read embedded resource: '{resourceName}'");

      using (var stream = assembly.GetManifestResourceStream(resourceName))
      {
        if (stream == null)
        {
          var availableResources = string.Join(", ", assembly.GetManifestResourceNames());
          throw new FileNotFoundException(
              $"Could not find the embedded resource '{resourceName}'. " +
              $"This is a packaging error. Available resources: [{availableResources}]");
        }
        using (var reader = new StreamReader(stream))
        {
          checksumsJson = reader.ReadToEnd();
        }
      }
      Console.WriteLine($"[SDK] Found and read embedded checksums file successfully.");

      using var doc = JsonDocument.Parse(checksumsJson);
      var versionNode = doc.RootElement.TryGetProperty(version, out var vNode)
        ? vNode
        : throw new InvalidOperationException($"Checksums.json does not contain an entry for version {version}.");

      var (goOs, archPart, archiveExt, platform) = GetPlatformDetails();
      var archiveName = $"{ProjectName}_{goOs}_{archPart}{archiveExt}";

      var expectedChecksumNode = versionNode.TryGetProperty(archiveName, out var cNode)
        ? cNode
        : throw new InvalidOperationException($"Checksums.json for {version} does not contain an entry for {archiveName}.");

      var expectedChecksum = expectedChecksumNode.GetString();
      if (string.IsNullOrEmpty(expectedChecksum) || expectedChecksum.StartsWith("PLEASE_RUN_UPDATE_SCRIPT"))
        throw new InvalidOperationException($"Checksum for {archiveName} in {version} looks invalid or is a placeholder.");

      var binDir = Path.GetFullPath(outDir);
      Directory.CreateDirectory(binDir);
      var binaryName = platform == "win32" ? ProjectName + ".exe" : ProjectName;
      var finalBinaryPath = Path.Combine(binDir, binaryName);

      if (File.Exists(finalBinaryPath))
      {
        Console.WriteLine($"[SDK] {ProjectName} binary already exists at {finalBinaryPath}. Skipping download.");
        EnsureExecutable(finalBinaryPath);
        return;
      }

      var downloadUrl = $"https://github.com/{GithubOwner}/{GithubRepo}/releases/download/{version}/{archiveName}";
      var archivePath = Path.Combine(binDir, archiveName);

      try
      {
        await DownloadFileAsync(downloadUrl, archivePath);
        var actualChecksum = await ComputeSha256Async(archivePath);
        if (!string.Equals(actualChecksum, expectedChecksum, StringComparison.OrdinalIgnoreCase))
        {
          throw new InvalidOperationException($"Checksum mismatch for {archiveName}. Expected: {expectedChecksum}, Actual: {actualChecksum}");
        }

        ExtractArchive(archivePath, archiveExt, binDir);
        EnsureExecutable(finalBinaryPath);

        Console.WriteLine($"[SDK] {ProjectName} ready at {finalBinaryPath}");
      }
      finally
      {
        // Ensure the downloaded archive is cleaned up even if something goes wrong.
        if (File.Exists(archivePath))
        {
          try { File.Delete(archivePath); } catch { /* Best effort */ }
        }
      }
    }

    private static (string goOs, string archPart, string archiveExt, string platform) GetPlatformDetails()
    {
      string platform;
      if (RuntimeInformation.IsOSPlatform(OSPlatform.OSX)) platform = "darwin";
      else if (RuntimeInformation.IsOSPlatform(OSPlatform.Linux)) platform = "linux";
      else if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows)) platform = "win32";
      else throw new PlatformNotSupportedException("Unsupported OS platform");

      var arch = RuntimeInformation.ProcessArchitecture;
      string archPart;
      if (arch == Architecture.X64) archPart = "x86_64";
      else if (arch == Architecture.Arm64) archPart = "arm64";
      else throw new PlatformNotSupportedException("Unsupported architecture");

      string goOs = platform == "darwin" ? "Darwin" : platform == "linux" ? "Linux" : "Windows";
      string archiveExt = platform == "win32" ? ".zip" : ".tar.gz";
      return (goOs, archPart, archiveExt, platform);
    }

    private static async Task DownloadFileAsync(string url, string destinationPath)
    {
      Console.WriteLine($"[TestServerSDK] Downloading {url} -> {destinationPath}...");
      using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(2) };
      using var resp = await client.GetAsync(url, HttpCompletionOption.ResponseHeadersRead);
      resp.EnsureSuccessStatusCode();
      using var stream = await resp.Content.ReadAsStreamAsync();
      using var fs = new FileStream(destinationPath, FileMode.Create, FileAccess.Write, FileShare.None);
      await stream.CopyToAsync(fs);
      Console.WriteLine("[TestServerSDK] Download complete.");
    }

    private static async Task<string> ComputeSha256Async(string filePath)
    {
      using var stream = File.OpenRead(filePath);
      using var sha = SHA256.Create();
      var hash = await sha.ComputeHashAsync(stream);
      return BitConverter.ToString(hash).Replace("-", string.Empty).ToLowerInvariant();
    }

    private static void ExtractArchive(string archivePath, string archiveExt, string destDir)
    {
      Console.WriteLine($"[TestServerSDK] Extracting {archivePath} to {destDir}...");
      if (archiveExt == ".zip")
      {
        System.IO.Compression.ZipFile.ExtractToDirectory(archivePath, destDir, true);
      }
      else
      {
        using var fileStream = File.OpenRead(archivePath);
        using var reader = ReaderFactory.Open(fileStream);
        while (reader.MoveToNextEntry())
        {
          if (reader.Entry.IsDirectory) continue;
          reader.WriteEntryToDirectory(destDir, new ExtractionOptions { ExtractFullPath = true, Overwrite = true });
        }
      }
      Console.WriteLine("[TestServerSDK] Extraction complete.");
    }

    private static void EnsureExecutable(string binaryPath)
    {
      if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows)) return;
      try
      {
        var psi = new ProcessStartInfo
        {
          FileName = "chmod",
          Arguments = $"+x {QuotePath(binaryPath)}",
          RedirectStandardOutput = true,
          RedirectStandardError = true,
          UseShellExecute = false
        };
        using var p = Process.Start(psi) ?? throw new InvalidOperationException("Failed to start 'chmod' process.");
        p.WaitForExit();
        if (p.ExitCode != 0)
        {
          Console.WriteLine($"[TestServerSDK WARNING] chmod failed: {p.StandardError.ReadToEnd()}");
        }
        else
        {
          Console.WriteLine($"[TestServerSDK] Set executable permissions on {binaryPath}");
        }
      }
      catch (Exception ex)
      {
        Console.WriteLine($"[TestServerSDK WARNING] Could not set executable permission: {ex.Message}");
      }
    }

    private static string QuotePath(string p) => p.Contains(' ') ? '"' + p + '"' : p;
  }
}
