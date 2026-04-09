/*
 * Copyright 2026 Google LLC
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

package com.google.testserver

import java.io.File
import java.io.FileInputStream
import java.io.IOException
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.util.concurrent.TimeUnit
import org.yaml.snakeyaml.Yaml

class TestServer(private val options: TestServerOptions) {

  private var process: Process? = null

  fun start(): Process {
    val binaryFile =
        options.binaryPath?.let { File(it) } ?: BinaryInstaller.ensureBinary(options.outDir)

    val args = mutableListOf<String>()
    args.add(binaryFile.absolutePath)
    args.add(options.mode)
    args.add("--config")
    args.add(options.configPath)
    args.add("--recording-dir")
    args.add(options.recordingDir)

    println("[TestServer] Starting test-server: ${args.joinToString(" ")}")

    val processBuilder = ProcessBuilder(args)
    processBuilder.redirectErrorStream(
        true) // Merge stdout and stderr for simplicity, or we can handle separately

    val env = processBuilder.environment()
    options.env?.forEach { (k, v) -> env[k] = v }
    options.testServerSecrets?.let { env["TEST_SERVER_SECRETS"] = it }

    val p = processBuilder.start()
    process = p

    // Log output in a separate thread
    Thread {
          p.inputStream.bufferedReader().use { reader ->
            var line = reader.readLine()
            while (line != null) {
              println("[test-server] $line")
              line = reader.readLine()
            }
          }
        }
        .start()

    // Wait for it to be healthy
    awaitHealthy()

    return p
  }

  fun stop() {
    process?.let { p ->
      if (p.isAlive) {
        println("[TestServer] Stopping test-server process (PID: ${p.pid()})...")
        p.destroy()
        if (!p.waitFor(5, TimeUnit.SECONDS)) {
          println("[TestServer] Process did not exit in time. Forcibly destroying...")
          p.destroyForcibly()
        }
        println("[TestServer] Stopped.")
      }
    }
  }

  private fun awaitHealthy() {
    val yaml = Yaml()
    val configStream = FileInputStream(options.configPath)
    val config = yaml.load<Map<String, Any>>(configStream)

    val endpoints = config["endpoints"] as? List<Map<String, Any>> ?: return

    for (endpoint in endpoints) {
      val healthPath = endpoint["health"] as? String ?: continue
      val sourceType = endpoint["source_type"] as? String ?: "http"
      val sourcePort = endpoint["source_port"]?.toString() ?: continue

      val url = "$sourceType://localhost:$sourcePort$healthPath"
      healthCheck(url)
    }
  }

  private fun healthCheck(url: String) {
    val maxRetries = 10
    var delay = 100L
    val client = HttpClient.newHttpClient()

    for (i in 0 until maxRetries) {
      try {
        val request = HttpRequest.newBuilder().uri(URI.create(url)).build()
        val response = client.send(request, HttpResponse.BodyHandlers.ofString())
        if (response.statusCode() == 200) {
          println("[TestServer] Health check passed for $url")
          return
        }
      } catch (e: Exception) {
        // Ignore and retry
        println("[TestServer] Health check attempt ${i + 1} failed for $url: ${e.message}")
      }
      Thread.sleep(delay)
      delay *= 2
    }
    throw IOException("[TestServer] Health check failed for $url after $maxRetries retries.")
  }
}

data class TestServerOptions(
    val configPath: String,
    val recordingDir: String,
    val mode: String, // "record" or "replay"
    val outDir: File = File("build/test-server"), // Where to download/resolve binary
    val binaryPath: String? = null, // Optional, if provided use this instead of downloading
    val testServerSecrets: String? = null,
    val env: Map<String, String>? = null,
)
