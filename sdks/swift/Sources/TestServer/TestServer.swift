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

struct TestServerOptions {
    let configPath: String
    let recordingDir: String
    let mode: String // "record" or "replay"
    let binaryPath: String
    let testServerSecrets: String?
}

class TestServer {
    private var process: Process?
    private let options: TestServerOptions
    
    init(options: TestServerOptions) {
        self.options = options
    }
    
    func start() async throws {
        let binaryURL: URL
        let fileManager = FileManager.default
        
        if fileManager.fileExists(atPath: options.binaryPath) {
            binaryURL = URL(fileURLWithPath: options.binaryPath)
        } else {
            let targetDir = URL(fileURLWithPath: options.binaryPath).deletingLastPathComponent()
            print("[TestServerSdk] Installing binary to \(targetDir.path)...")
            binaryURL = try await BinaryInstaller.ensureBinary(at: targetDir)
        }

        let arguments = [
            options.mode,
            "--config", options.configPath,
            "--recording-dir", options.recordingDir
        ]
        
        let process = Process()
        process.executableURL = binaryURL
        process.arguments = arguments
        
        if let secrets = options.testServerSecrets {
             var env = ProcessInfo.processInfo.environment
             env["TEST_SERVER_SECRETS"] = secrets
             process.environment = env
        }
        
        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = pipe
        
        pipe.fileHandleForReading.readabilityHandler = { handle in
            if let data = try? handle.read(upToCount: handle.availableData.count),
               let str = String(data: data, encoding: .utf8), !str.isEmpty {
                print("[TestServer] \(str)", terminator: "")
            }
        }

        try process.run()
        self.process = process
        
        try await awaitHealthyTestServer()
    }
    
    func stop() {
        process?.terminate()
        process = nil
    }
    
    private func awaitHealthyTestServer() async throws {
        let healthURLString = try extractHealthURL(from: options.configPath)
        guard let url = URL(string: healthURLString) else {
            throw NSError(domain: "TestServer", code: -1, userInfo: [NSLocalizedDescriptionKey: "Invalid health URL"])
        }
        
        print("[TestServer] Waiting for healthy server at \(url)...")
        try await checkHealth(url: url)
    }
    
    private func extractHealthURL(from configPath: String) throws -> String {
        let content = try String(contentsOfFile: configPath, encoding: .utf8)
        let fullRange = NSRange(content.startIndex..., in: content)

        // Find the first 'source_port', looks for "source_port: 1234"
        let portPattern = #"source_port:\s*(\d+)"#
        let portRegex = try NSRegularExpression(pattern: portPattern)
        let portMatch = portRegex.firstMatch(in: content, range: fullRange)
        
        guard let portRange = portMatch?.range(at: 1),
              let portRangeInString = Range(portRange, in: content) else {
            print("[TestServer] Warning: Could not parse source_port from config. Defaulting to 9000.")
            return "http://localhost:9000/health"
        }
        let port = String(content[portRangeInString])
        
        var healthPath = "/health" // Default
        let healthPattern = #"health:\s*([\w/]+)"#
        
        if let healthRegex = try? NSRegularExpression(pattern: healthPattern),
           let healthMatch = healthRegex.firstMatch(in: content, range: fullRange),
           let healthRangeInString = Range(healthMatch.range(at: 1), in: content) {
            healthPath = String(content[healthRangeInString])
        }
        
        return "http://localhost:\(port)\(healthPath)"
    }


    
    private func checkHealth(url: URL) async throws {
        let session = URLSession.shared
        let maxRetries = 20
        let delay = 0.5

        for _ in 0..<maxRetries {
            if let process = process, !process.isRunning {
                throw NSError(domain: "TestServer", code: -1, 
                            userInfo: [NSLocalizedDescriptionKey: "Server process died unexpectedly during startup."])
            }

            do {
                let (_, response) = try await session.data(from: url)
                if let httpResponse = response as? HTTPURLResponse, httpResponse.statusCode == 200 {
                    return
                }
            } catch { /* retry */ }
            
            try await Task.sleep(nanoseconds: UInt64(delay * 1_000_000_000))
        }
        throw NSError(domain: "TestServer", code: -1, userInfo: [NSLocalizedDescriptionKey: "Health check failed"])
    }

}
