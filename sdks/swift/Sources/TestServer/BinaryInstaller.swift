// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import Foundation

enum BinaryInstallerError: Error {
    case platformNotSupported
    case downloadFailed(Error)
    case extractionFailed
    case fileSystemError(Error)
}

/// Ensures the `test-server` binary is available on the local machine.
struct BinaryInstaller {
    private static let githubOwner = "google"
    private static let githubRepo = "test-server"
    private static let projectName = "test-server"
    static let testServerVersion = "v0.2.9"

    static func ensureBinary(at outputDirectory: URL, version: String = testServerVersion) async throws -> URL {
        let (os, arch, ext) = try getPlatformDetails()
        let archiveName = "\(projectName)_\(os)_\(arch)\(ext)"

        try FileManager.default.createDirectory(at: outputDirectory, withIntermediateDirectories: true)

        let binaryName = projectName
        let finalBinaryURL = outputDirectory.appendingPathComponent(binaryName)

        // Check if binary already exists
        if FileManager.default.fileExists(atPath: finalBinaryURL.path) {
            print("[SDK] \(projectName) binary already exists at \(finalBinaryURL.path).")
            return finalBinaryURL
        }

        let downloadURLString = "https://github.com/\(githubOwner)/\(githubRepo)/releases/download/\(version)/\(archiveName)"
        guard let downloadURL = URL(string: downloadURLString) else {
            throw BinaryInstallerError.platformNotSupported
        }

        print("[SDK] Downloading \(downloadURLString)...")
        
        let (tempLocalURL, response) = try await URLSession.shared.download(from: downloadURL)
        
        guard let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 else {
            throw BinaryInstallerError.downloadFailed(NSError(domain: "Download", code: -1, userInfo: nil))
        }

        // Move to a temporary location with correct extension for extraction
        let tempArchiveURL = outputDirectory.appendingPathComponent(archiveName)
        try? FileManager.default.removeItem(at: tempArchiveURL)
        try FileManager.default.moveItem(at: tempLocalURL, to: tempArchiveURL)

        defer {
            try? FileManager.default.removeItem(at: tempArchiveURL)
        }

        print("[SDK] Extracting to \(outputDirectory.path)...")
        try extract(archive: tempArchiveURL, to: outputDirectory, extension: ext)

        try setExecutablePermissions(at: finalBinaryURL)

        print("[SDK] Ready at \(finalBinaryURL.path)")
        return finalBinaryURL
    }

    private static func getPlatformDetails() throws -> (os: String, arch: String, ext: String) {
        #if os(macOS)
        let os = "Darwin"
        let ext = ".tar.gz"
        #elseif os(Linux)
        let os = "Linux"
        let ext = ".tar.gz"
        #else
        throw BinaryInstallerError.platformNotSupported
        #endif

        #if arch(x86_64)
        let arch = "x86_64"
        #elseif arch(arm64)
        let arch = "arm64"
        #else
        throw BinaryInstallerError.platformNotSupported
        #endif

        return (os, arch, ext)
    }

    private static func extract(archive: URL, to directory: URL, extension ext: String) throws {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/usr/bin/tar")
        process.arguments = ["-xzf", archive.path, "-C", directory.path]
        
        try process.run()
        process.waitUntilExit()
        
        if process.terminationStatus != 0 {
            throw BinaryInstallerError.extractionFailed
        }
    }

    private static func setExecutablePermissions(at url: URL) throws {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: "/bin/chmod")
        process.arguments = ["+x", url.path]
        
        try process.run()
        process.waitUntilExit()
    }
}
