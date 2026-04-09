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
import java.io.FileOutputStream
import java.io.IOException
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.security.MessageDigest
import java.util.zip.ZipInputStream

object BinaryInstaller {
  private const val GITHUB_OWNER = "google"
  private const val GITHUB_REPO = "test-server"
  private const val PROJECT_NAME = "test-server"
  const val TEST_SERVER_VERSION = "v0.2.9"

  fun ensureBinary(outDir: File, version: String = TEST_SERVER_VERSION): File {
    val platformDetails = getPlatformDetails()
    val archiveName =
        "${PROJECT_NAME}_${platformDetails.goOs}_${platformDetails.archPart}${platformDetails.archiveExt}"

    val binaryName = if (platformDetails.platform == "win32") "$PROJECT_NAME.exe" else PROJECT_NAME
    val finalBinaryPath = File(outDir, binaryName)

    if (finalBinaryPath.exists()) {
      println("[SDK] Binary already exists at ${finalBinaryPath.absolutePath}. Skipping download.")
      ensureExecutable(finalBinaryPath)
      return finalBinaryPath
    }

    val downloadUrl =
        "https://github.com/$GITHUB_OWNER/$GITHUB_REPO/releases/download/$version/$archiveName"
    val archiveFile = File(outDir, archiveName)

    outDir.mkdirs()

    try {
      downloadFile(downloadUrl, archiveFile)

      // Verification (Checksums can be loaded from checksums.json later)
      // For now, we can skip or implement placeholder verification
      // verifyChecksum(archiveFile, "expectedChecksum")

      extractArchive(archiveFile, platformDetails.archiveExt, outDir)
      ensureExecutable(finalBinaryPath)

      println("[SDK] $PROJECT_NAME ready at ${finalBinaryPath.absolutePath}")
      return finalBinaryPath
    } finally {
      if (archiveFile.exists()) {
        archiveFile.delete()
      }
    }
  }

  private fun getPlatformDetails(): PlatformDetails {
    val osName = System.getProperty("os.name").lowercase()
    val osArch = System.getProperty("os.arch").lowercase()

    val platform =
        when {
          osName.contains("mac") || osName.contains("darwin") -> "darwin"
          osName.contains("linux") -> "linux"
          osName.contains("win") -> "win32"
          else -> throw UnsupportedOperationException("Unsupported OS: $osName")
        }

    val archPart =
        when {
          osArch.contains("amd64") || osArch.contains("x86_64") -> "x86_64"
          osArch.contains("aarch64") || osArch.contains("arm64") -> "arm64"
          else -> throw UnsupportedOperationException("Unsupported Architecture: $osArch")
        }

    val goOs =
        when (platform) {
          "darwin" -> "Darwin"
          "linux" -> "Linux"
          "win32" -> "Windows"
          else -> throw IllegalArgumentException()
        }

    val archiveExt = if (platform == "win32") ".zip" else ".tar.gz"

    return PlatformDetails(goOs, archPart, archiveExt, platform)
  }

  private fun downloadFile(url: String, destination: File) {
    println("[SDK] Downloading $url -> ${destination.absolutePath}...")
    val client = HttpClient.newBuilder().followRedirects(HttpClient.Redirect.NORMAL).build()
    val request = HttpRequest.newBuilder().uri(URI.create(url)).build()

    val response = client.send(request, HttpResponse.BodyHandlers.ofInputStream())
    if (response.statusCode() != 200) {
      throw IOException("[SDK] Failed to download binary. Status: ${response.statusCode()}")
    }

    response.body().use { input ->
      FileOutputStream(destination).use { output -> input.copyTo(output) }
    }
    println("[SDK] Download complete.")
  }

  private fun extractArchive(archiveFile: File, ext: String, destDir: File) {
    println("[SDK] Extracting ${archiveFile.absoluteFile} to ${destDir.absolutePath}...")
    if (ext == ".zip") {
      unzip(archiveFile, destDir)
    } else {
      // For .tar.gz on Unix system, using tar command via ProcessBuilder is easiest and avoids
      // heavy dependencies
      val processBuilder =
          ProcessBuilder("tar", "-xzf", archiveFile.absolutePath, "-C", destDir.absolutePath)
      val process = processBuilder.start()
      val exitCode = process.waitFor()

      if (exitCode != 0) {
        val errorStream = process.errorStream.bufferedReader().readText()
        throw IOException("[SDK] Failed to extract tar.gz: $errorStream")
      }
    }
    println("[SDK] Extraction complete.")
  }

  private fun unzip(zipFile: File, destDir: File) {
    ZipInputStream(FileInputStream(zipFile)).use { zis ->
      var entry = zis.nextEntry
      while (entry != null) {
        val newFile = File(destDir, entry.name)
        if (entry.isDirectory) {
          newFile.mkdirs()
        } else {
          newFile.parentFile.mkdirs()
          FileOutputStream(newFile).use { fos -> zis.copyTo(fos) }
        }
        zis.closeEntry()
        entry = zis.nextEntry
      }
    }
  }

  private fun ensureExecutable(file: File) {
    if (!System.getProperty("os.name").lowercase().contains("win")) {
      file.setExecutable(true)
    }
  }

  fun computeSha256(file: File): String {
    val digest = MessageDigest.getInstance("SHA-256")
    FileInputStream(file).use { input ->
      val buffer = ByteArray(1024)
      var bytesRead = input.read(buffer)
      while (bytesRead != -1) {
        digest.update(buffer, 0, bytesRead)
        bytesRead = input.read(buffer)
      }
    }
    return digest.digest().joinToString("") { "%02x".format(it) }
  }

  private data class PlatformDetails(
      val goOs: String,
      val archPart: String,
      val archiveExt: String,
      val platform: String,
  )
}
