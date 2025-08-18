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
using System.IO.Compression;
using SharpCompress.Archives;
using SharpCompress.Archives.Tar;
using SharpCompress.Common;
using SharpCompress.Readers;

namespace TestServerSdk
{
  public static class BinaryInstaller
  {
    private const string GithubOwner = "google";
    private const string GithubRepo = "test-server";
    private const string ProjectName = "test-server";

    /// <summary>
    /// Ensures the test-server binary for the given version is present under <repo>/sdks/dotnet/bin.
    /// It will download the release asset from GitHub, verify SHA256 using the checksums.json found in the repo, extract it and set executable bits.
    /// </summary>
    public static async Task EnsureBinaryAsync(string outDir, string version = "v0.2.6")
    {
      var embeddedCandidate = Path.Combine(AppContext.BaseDirectory, "checksums.json");
      var repoChecksumsPath = File.Exists(embeddedCandidate)
        ? embeddedCandidate
        : FindChecksumsJson() ?? throw new FileNotFoundException("Could not locate sdks/typescript/checksums.json in repository parents or embedded in output. Please run this command from within the repo or provide checksums.json manually.");

      using var doc = JsonDocument.Parse(File.ReadAllText(repoChecksumsPath));
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
      var finalBinaryPath = Path.Combine(binDir, platform == "win32" ? ProjectName + ".exe" : ProjectName);
      if (File.Exists(finalBinaryPath))
      {
        Console.WriteLine($"{ProjectName} binary already exists at {finalBinaryPath}. Skipping download.");
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
          File.Delete(archivePath);
          throw new InvalidOperationException($"Checksum mismatch for {archiveName}. Expected: {expectedChecksum}, Actual: {actualChecksum}");
        }

        ExtractArchive(archivePath, archiveExt, binDir);
        EnsureExecutable(finalBinaryPath);

        Console.WriteLine($"{ProjectName} ready at {finalBinaryPath}");
      }
      catch
      {
        if (File.Exists(archivePath))
        {
          try { File.Delete(archivePath); } catch { }
        }
        throw;
      }
    }

    private static string RepoRootPathFrom(string checksumsPath)
    {
      // checksumsPath is expected to be <repo>/sdks/dotnet/checksums.json
      return Path.GetFullPath(Path.Combine(Path.GetDirectoryName(checksumsPath) ?? string.Empty, "..", ".."));
    }

    private static string? FindChecksumsJson()
    {
      // Start from AppContext.BaseDirectory and walk up to find sdks/dotnet/checksums.json
      var dir = new DirectoryInfo(AppContext.BaseDirectory);
      for (int i = 0; i < 8 && dir != null; i++)
      {
        var candidate = Path.Combine(dir.FullName, "sdks", "dotnet", "checksums.json");
        if (File.Exists(candidate)) return candidate;
        dir = dir.Parent;
      }
      // Also try the current working directory
      dir = new DirectoryInfo(Directory.GetCurrentDirectory());
      for (int i = 0; i < 4 && dir != null; i++)
      {
        var candidate = Path.Combine(dir.FullName, "sdks", "dotnet", "checksums.json");
        if (File.Exists(candidate)) return candidate;
        dir = dir.Parent;
      }
      return null;
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
      Console.WriteLine($"Downloading {url} -> {destinationPath}...");
      using var client = new HttpClient { Timeout = TimeSpan.FromMinutes(2) };
      using var resp = await client.GetAsync(url, HttpCompletionOption.ResponseHeadersRead);
      resp.EnsureSuccessStatusCode();
      using var stream = await resp.Content.ReadAsStreamAsync();
      using var fs = new FileStream(destinationPath, FileMode.Create, FileAccess.Write, FileShare.None);
      await stream.CopyToAsync(fs);
      Console.WriteLine("Download complete.");
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
      Console.WriteLine($"Extracting {archivePath} to {destDir}...");
      if (archiveExt == ".zip")
      {
        ZipFile.ExtractToDirectory(archivePath, destDir);
      }
      else
      {
        using var fileStream = File.OpenRead(archivePath);
        using var reader = ReaderFactory.Open(fileStream);
        while (reader.MoveToNextEntry())
        {
          var entry = reader.Entry;
          if (entry.IsDirectory) continue;
          var outPath = Path.Combine(destDir, entry.Key);
          var outDir = Path.GetDirectoryName(outPath);
          if (!string.IsNullOrEmpty(outDir)) Directory.CreateDirectory(outDir);
          reader.WriteEntryToDirectory(destDir, new ExtractionOptions { ExtractFullPath = true, Overwrite = true });
        }
      }
      File.Delete(archivePath);
      Console.WriteLine("Extraction complete.");
    }

    private static void EnsureExecutable(string binaryPath)
    {
      if (RuntimeInformation.IsOSPlatform(OSPlatform.Windows)) return;
      try
      {
        var psi = new ProcessStartInfo
        {
          FileName = "chmod",
          Arguments = $"755 {QuotePath(binaryPath)}",
          RedirectStandardOutput = true,
          RedirectStandardError = true,
          UseShellExecute = false
        };
        using var p = Process.Start(psi) ?? throw new InvalidOperationException("Failed to start 'chmod' process to set executable permissions");
        p.WaitForExit();
        if (p.ExitCode != 0)
        {
          var err = p.StandardError.ReadToEnd();
          Console.WriteLine($"chmod failed: {err}");
        }
        else
        {
          Console.WriteLine($"Set executable permissions on {binaryPath}");
        }
      }
      catch (Exception ex)
      {
        Console.WriteLine($"Could not set executable permission: {ex.Message}");
      }
    }

    private static string QuotePath(string p) => p.Contains(' ') ? '"' + p + '"' : p;
  }
}
