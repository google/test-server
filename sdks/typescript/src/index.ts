/**
 * Copyright 2025 Google LLC
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { spawn, ChildProcess } from 'child_process';
import * as path from 'path';
import * as fs from 'fs';

const PROJECT_NAME = 'test-server';

const getBinaryPath = (): string => {
    const platform = process.platform;
    const binaryName = platform === 'win32' ? `${PROJECT_NAME}.exe` : PROJECT_NAME;
    // Assuming this script (when compiled) is in sdks/typescript/dist/index.js
    // So __dirname is sdks/typescript/dist
    const binaryPath = path.resolve(__dirname, '..', 'bin', binaryName); 
    
    if (!fs.existsSync(binaryPath)) {
        throw new Error(
            `test-server binary not found at ${binaryPath}. ` +
            `This usually means the postinstall script failed to download or extract the binary. ` +
            `Please try reinstalling the test-server-sdk package. ` +
            `If the issue persists, check the output of the postinstall script for errors.`
        );
    }
    return binaryPath;
};

export interface TestServerOptions {
    /** Path to the test-server configuration file. */
    configPath: string;
    /** Directory to store/load recordings. */
    recordingDir: string;
    /**
     * Mode to run test-server in.
     * - 'record': Forces record mode.
     * - 'replay': Forces replay mode.
     * - 'cli-driven': Mode is determined by CLI arguments. Defaults to 'replay' unless --record is passed.
     */
    mode: 'record' | 'replay' | 'cli-driven';
    /** Optional environment variables for the test-server process. */
    env?: NodeJS.ProcessEnv;
    /** Optional callback for stdout data. */
    onStdOut?: (data: string) => void;
    /** Optional callback for stderr data. */
    onStdErr?: (data:string) => void;
    /** Optional callback for when the process exits. */
    onExit?: (code: number | null, signal: NodeJS.Signals | null) => void;
    /** Optional callback for when an error occurs in spawning the process. */
    onError?: (err: Error) => void;
}

/**
 * Starts the test-server process.
 * @param options Configuration options for starting the server.
 * @returns The spawned ChildProcess instance.
 */
export function startTestServer(options: TestServerOptions): ChildProcess {
    const { configPath, recordingDir, mode: optionsMode, env, onStdOut, onStdErr, onExit, onError } = options;
    const binaryPath = getBinaryPath();

    let effectiveMode: 'record' | 'replay';

    if (optionsMode === 'record') {
        effectiveMode = 'record';
    } else if (optionsMode === 'replay') {
        effectiveMode = 'replay';
    } else { // optionsMode === 'cli-driven'
        console.log('Process args: ');
        console.log(process.argv);
        effectiveMode = process.argv.includes('--record') ? 'record' : 'replay';
    }

    const args = [
        effectiveMode,
        '--config',
        configPath,
        '--recording-dir',
        recordingDir,
    ];

    console.log(`[test-server-sdk] Starting test-server in '${effectiveMode}' mode. Command: ${binaryPath} ${args.join(' ')}`);

    const serverProcess = spawn(binaryPath, args, {
        env: { ...process.env, ...env },
        windowsHide: true, // Hide console window on Windows
    });

    serverProcess.stdout?.on('data', (data: Buffer) => {
        const output = data.toString();
        if (onStdOut) {
            onStdOut(output);
        } else {
            // Default behavior: log to console, clearly marking it.
            output.trimEnd().split('\n').forEach(line => console.log(`[test-server STDOUT] ${line}`));
        }
    });

    serverProcess.stderr?.on('data', (data: Buffer) => {
        const output = data.toString();
        if (onStdErr) {
            onStdErr(output);
        } else {
            // Default behavior: log to console, clearly marking it.
            output.trimEnd().split('\n').forEach(line => console.error(`[test-server STDERR] ${line}`));
        }
    });

    serverProcess.on('exit', (code, signal) => {
        console.log(`[test-server-sdk] test-server process (PID: ${serverProcess.pid}) exited with code ${code} and signal ${signal}`);
        if (onExit) {
            onExit(code, signal);
        }
    });

    serverProcess.on('error', (err) => {
        console.error('[test-server-sdk] Failed to start or manage test-server process:', err);
        if (onError) {
            onError(err);
        } else {
            // If no custom error handler, rethrow to make it obvious something went wrong.
            throw err;
        }
    });
    
    console.log(`[test-server-sdk] test-server process (PID: ${serverProcess.pid}) started.`);
    return serverProcess;
}

/**
 * Stops the test-server process.
 * @param serverProcess The ChildProcess instance to stop.
 * @returns A Promise that resolves when the process has exited.
 */
export function stopTestServer(serverProcess: ChildProcess): Promise<void> {
    return new Promise((resolve, reject) => {
        if (!serverProcess || serverProcess.killed || serverProcess.exitCode !== null) {
            console.log('[test-server-sdk] test-server process already stopped or not running.');
            resolve();
            return;
        }

        const pid = serverProcess.pid;
        console.log(`[test-server-sdk] Attempting to stop test-server process (PID: ${pid})...`);

        // Add listeners specifically for this stop operation
        const exitListener = (code: number | null, signal: NodeJS.Signals | null) => {
            clearTimeout(killTimeout);
            console.log(`[test-server-sdk] test-server process (PID: ${pid}) confirmed exit (code: ${code}, signal: ${signal}).`);
            resolve();
        };
        const errorListener = (err: Error) => {
            clearTimeout(killTimeout);
            console.error(`[test-server-sdk] Error during test-server process (PID: ${pid}) termination:`, err);
            reject(err);
        };

        serverProcess.once('exit', exitListener);
        serverProcess.once('error', errorListener);
        
        const killedBySigterm = serverProcess.kill('SIGTERM');

        if (!killedBySigterm) {
            // This can happen if the process already exited between the check and kill attempt.
            console.warn(`[test-server-sdk] SIGTERM signal to PID ${pid} failed. Process might have already exited.`);
            // Clean up listeners and resolve as the 'exit' event might not fire if already gone.
            serverProcess.removeListener('exit', exitListener);
            serverProcess.removeListener('error', errorListener);
            resolve(); 
            return;
        }
        console.log(`[test-server-sdk] SIGTERM sent to test-server process (PID: ${pid}). Waiting for graceful exit...`);

        const killTimeout = setTimeout(() => {
            if (!serverProcess.killed && serverProcess.exitCode === null) {
                console.warn(`[test-server-sdk] test-server process (PID: ${pid}) did not terminate with SIGTERM after 5s. Sending SIGKILL.`);
                serverProcess.kill('SIGKILL');
            }
        }, 5000); // 5 seconds timeout
    });
}
