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
