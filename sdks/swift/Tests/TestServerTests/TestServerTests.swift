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

import XCTest
@testable import TestServer

/// Validates the core functionality of the `TestServer` class
final class TestServerTests: XCTestCase {
    
    func testServerLifecycle() async throws {
        let tempDir = FileManager.default.temporaryDirectory.appendingPathComponent("TestServerTests")
        try? FileManager.default.createDirectory(at: tempDir, withIntermediateDirectories: true)
        
        let binDir = tempDir.appendingPathComponent("bin")


        let recordingsDir = tempDir.appendingPathComponent("recordings")
        try FileManager.default.createDirectory(at: recordingsDir, withIntermediateDirectories: true)

        let configURL = tempDir.appendingPathComponent("test-server.yml")

        let placeholderConfig = """
        endpoints:
          - source_type: http
            source_port: 1453
            health: /healthz
        """
        try placeholderConfig.write(to: configURL, atomically: true, encoding: .utf8)
        
        let options = TestServerOptions(
            configPath: configURL.path,
            recordingDir: recordingsDir.path,
            mode: "replay",
            binaryPath: binDir.appendingPathComponent("test-server").path,
            testServerSecrets: nil
        )
        
        let server = TestServer(options: options)
        
        try await server.start()
        print("✅ Server started and healthy!")
        
        server.stop()
        print("🛑 Server stopped.")
    }
}
