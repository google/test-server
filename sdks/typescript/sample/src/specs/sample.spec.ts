import { startTestServer, stopTestServer, TestServerOptions } from 'test-server-sdk';
import { ChildProcess } from 'child_process';
import * as path from 'path';
import * as http from 'http';

// Define options and paths directly in the spec file
// When compiled, this file will be in dist/specs/, so __dirname is .../sample/dist/specs
// We want to get to .../sample/
const samplePackageRoot = path.resolve(__dirname, '..', '..'); 
const configFilePath = path.join(samplePackageRoot, 'test-data', 'config', 'test-server-config.yml');
const recordingsDir = path.join(samplePackageRoot, 'test-data', 'recordings');

console.log(`[SampleSpec] Using test-server mode: 'cli-driven' (defaults to replay, or record if --record is passed)`);

const sampleTestServerOptions: TestServerOptions = {
    configPath: configFilePath,
    recordingDir: recordingsDir,
    mode: 'cli-driven',
    onStdOut: (data) => console.log(`[test-server STDOUT] ${data.trimEnd()}`),
    onStdErr: (data) => console.error(`[test-server STDERR] ${data.trimEnd()}`),
    onError: (err) => console.error('[test-server ERROR] Failed to start or manage test-server process:', err),
};

describe('Sample Test Suite (with test-server)', () => {
    let serverProcess: ChildProcess | null = null;

    beforeAll(async () => {
        try {
            serverProcess = await startTestServer(sampleTestServerOptions);
            console.log(`[SampleSpec] test-server started with PID: ${serverProcess.pid}. Waiting for it to be ready...`);
            // TODO(amirh): Replace this with some sort of a readiness check.
            await new Promise(resolve => setTimeout(resolve, 500));
            console.log('[SampleSpec] test-server should be ready.');
        } catch (error) {
            console.error('[SampleSpec] Failed to start test-server for suite:', error);
            throw error; // Fail fast if server doesn't start
        }
    }, 15000); // Increased timeout for beforeAll

    afterAll(async () => {
        if (serverProcess) {
            console.log('[SampleSpec] Tearing down test-server for this suite...');
            await stopTestServer(serverProcess);
            serverProcess = null;
            console.log('[SampleSpec] test-server stopped for suite.');
        }
    }, 15000); // Increased timeout for afterAll

    it('should receive a 200 response from proxied www.github.com', async () => {
        console.log('[SampleSpec] Making request to test-server proxy for www.github.com...');
        
        await new Promise<void>((resolve, reject) => {
            const req = http.get('http://localhost:18080/', (res) => {
                let data = '';
                expect(res.statusCode).toBe(200);

                res.on('data', (chunk) => {
                    data += chunk;
                });

                res.on('end', () => {
                    expect(JSON.stringify(res.headers)).toContain('github');
                    console.log('[SampleSpec] Received response. Because the body of the response can be empty, test headers instead.');
                    resolve();
                });
            });

            req.on('error', (e) => {
                console.error(`[SampleSpec] HTTP request error: ${e.message}`);
                reject(e);
            });

            req.end();
        });
    }, 10000);
});

describe('Another Sample Test Suite (without test-server)', () => {
    it('should run a basic test independently of test-server', () => {
        console.log('[SampleSpec] Running a test in a suite that does not manage test-server.');
        expect(true).toBe(true);
    });
});
