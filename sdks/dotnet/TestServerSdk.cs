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
using System.Diagnostics;
using System.IO;
using System.Threading;
using System.Threading.Tasks;
using System.Text.Json;
using YamlDotNet.RepresentationModel;
using System.Net.Http;

namespace TestServerSdk
{
  public class TestServerOptions
  {
    public required string ConfigPath { get; set; }
    public required string RecordingDir { get; set; }
    public required string Mode { get; set; } // "record" or "replay"
    public required string BinaryPath { get; set; }

    public Action<string>? OnStdOut { get; set; }
    public Action<string>? OnStdErr { get; set; }
    public Action<int?, string>? OnExit { get; set; }
    public Action<Exception>? OnError { get; set; }
  }

  public class TestServerProcess
  {
    private Process? _process;
    private readonly TestServerOptions _options;
    private readonly string _binaryPath;

    public TestServerProcess(TestServerOptions options)
    {
      _options = options;
      _binaryPath = GetBinaryPath();
    }

    private string GetBinaryPath()
    {
      var binaryName = Environment.OSVersion.Platform == PlatformID.Win32NT ? "test-server.exe" : "test-server";

      var p = Path.GetFullPath(_options.BinaryPath);
      if (File.Exists(p)) return p;

      // If the binary does not exist at the provided path, attempt to install it into that folder
      try
      {
        var targetDir = Path.GetDirectoryName(p) ?? Path.GetFullPath(Directory.GetCurrentDirectory());
        Console.WriteLine($"[TestServerSdk] test-server not found at {p}. Installing into {targetDir}...");
        BinaryInstaller.EnsureBinaryAsync(targetDir, "v0.2.7").GetAwaiter().GetResult();
        if (File.Exists(p)) return p;
        throw new FileNotFoundException($"[TestServerSdk] After installation, test-server binary still not found at: {p}");
      }
      catch (Exception ex)
      {
        throw new FileNotFoundException($"[TestServerSdk] TestServerOptions.BinaryPath was set but file not found and installer failed: {p}", ex);
      }
    }

    public async Task<Process> StartAsync()
    {
      var args = $"{_options.Mode} --config {_options.ConfigPath} --recording-dir {_options.RecordingDir}";
      var psi = new ProcessStartInfo
      {
        FileName = _binaryPath,
        Arguments = args,
        RedirectStandardOutput = true,
        RedirectStandardError = true,
        UseShellExecute = false,
        CreateNoWindow = true
      };
      _process = new Process { StartInfo = psi, EnableRaisingEvents = true };
      _process.OutputDataReceived += (s, e) => { if (e.Data != null) _options.OnStdOut?.Invoke(e.Data); };
      _process.ErrorDataReceived += (s, e) => { if (e.Data != null) _options.OnStdErr?.Invoke(e.Data); };
      _process.Exited += (s, e) => _options.OnExit?.Invoke(_process?.ExitCode, _process?.ExitCode.ToString() ?? string.Empty);
      try
      {
        _process.Start();
        _process.BeginOutputReadLine();
        _process.BeginErrorReadLine();
        await AwaitHealthyTestServer();
        return _process;
      }
      catch (Exception ex)
      {
        _options.OnError?.Invoke(ex);
        throw;
      }
    }

    public async Task StopAsync()
    {
      if (_process == null || _process.HasExited)
        return;
      _process.Kill();
      await Task.Run(() => _process.WaitForExit(5000));
    }

    private async Task AwaitHealthyTestServer()
    {
      var yaml = File.ReadAllText(_options.ConfigPath);
      var input = new StringReader(yaml);
      var yamlStream = new YamlStream();
      yamlStream.Load(input);
      var root = (YamlMappingNode)yamlStream.Documents[0].RootNode;
      if (!root.Children.ContainsKey(new YamlScalarNode("endpoints"))) return;
      var endpoints = (YamlSequenceNode)root.Children[new YamlScalarNode("endpoints")];
      foreach (YamlMappingNode endpoint in endpoints)
      {
        if (!endpoint.Children.ContainsKey(new YamlScalarNode("health"))) continue;
        var sourceType = endpoint.Children[new YamlScalarNode("source_type")].ToString();
        var sourcePort = endpoint.Children[new YamlScalarNode("source_port")].ToString();
        var healthPath = endpoint.Children[new YamlScalarNode("health")].ToString();
        var url = $"{sourceType}://localhost:{sourcePort}{healthPath}";
        await HealthCheck(url);
      }
    }

    private async Task HealthCheck(string url)
    {
      using var client = new HttpClient();
      const int maxRetries = 10;
      int delay = 100;
      for (int i = 0; i < maxRetries; i++)
      {
        try
        {
          var response = await client.GetAsync(url);
          if (response.IsSuccessStatusCode) return;
        }
        catch { }
        await Task.Delay(delay);
        delay *= 2;
      }
      throw new Exception($"Health check failed for {url}");
    }
  }
}
