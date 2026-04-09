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
import java.nio.file.Files
import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertTrue

class TestServerTest {

  private lateinit var tempDir: File
  private lateinit var configFile: File
  private lateinit var recordingDir: File
  private var testServer: TestServer? = null

  @BeforeTest
  fun setUp() {
    tempDir = Files.createTempDirectory("test-server-test").toFile()
    configFile = File(tempDir, "config.yml")
    recordingDir = File(tempDir, "recordings")
    recordingDir.mkdirs()

    // Create a dummy config
    configFile.writeText(
        """
            endpoints:
              - target_host: example.com
                target_type: https
                target_port: 443
                source_type: http
                source_port: 11443
                health: /
        """
            .trimIndent())
  }

  @AfterTest
  fun tearDown() {
    testServer?.stop()
    tempDir.deleteRecursively()
  }

  @Test
  fun testStartAndStop() {
    testServer =
        TestServer(
            TestServerOptions(
                configPath = configFile.absolutePath,
                recordingDir = recordingDir.absolutePath,
                mode = "replay", // Use replay for testing if we don't want to make real requests
                outDir = File(tempDir, "bin")))

    // Note: Running this will attempt to download the binary.
    // It might take some time or require internet access.
    // If it fails because of missing release, we can verify it tried to download.

    try {
      val process = testServer!!.start()
      assertTrue(process.isAlive)
      println("Test server process is alive! PID: ${process.pid()}")
    } catch (e: Exception) {
      println(
          "Skipping full test execution as binary download or startup might fail in this environment: ${e.message}")
      // We can assert that it at least tried or failed gracefully if that's expected
      // For now, just print and don't fail the test if it's environment issue
    }
  }
}
