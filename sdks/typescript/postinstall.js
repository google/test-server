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

const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');
const crypto = require('crypto');
const axios = require('axios');
const extract = require('extract-zip');
const tar = require('tar');
const allExpectedChecksums = require('./checksums.json');
const TEST_SERVER_VERSION = 'v0.2.1';

const GITHUB_OWNER = 'google';
const GITHUB_REPO = 'test-server';
const PROJECT_NAME = 'test-server';
const BIN_DIR = path.join(__dirname, 'bin');
const getBinaryPath = () => path.join(BIN_DIR, os.platform() === 'win32' ? `${PROJECT_NAME}.exe` : PROJECT_NAME);

function getPlatformDetails() {
    const platform = os.platform();
    const arch = os.arch();
    let goOs, goArchFilenamePart, archiveExtension = '.tar.gz';

    if (platform === 'darwin') goOs = 'Darwin';
    else if (platform === 'linux') goOs = 'Linux';
    else if (platform === 'win32') {
        goOs = 'Windows';
        archiveExtension = '.zip';
    } else throw new Error(`Unsupported platform: ${platform}`);

    if (arch === 'x64') goArchFilenamePart = 'x86_64';
    else if (arch === 'arm64') goArchFilenamePart = 'arm64';
    else throw new Error(`Unsupported architecture: ${arch}`);

    return { goOs, goArchFilenamePart, archiveExtension, platform };
}

function calculateFileSha256(filePath) {
    return new Promise((resolve, reject) => {
        const hash = crypto.createHash('sha256');
        const stream = fs.createReadStream(filePath);
        stream.on('data', (data) => hash.update(data));
        stream.on('end', () => resolve(hash.digest('hex')));
        stream.on('error', (err) => reject(new Error(`Failed to calculate SHA256 for ${filePath}: ${err.message}`)));
    });
}

async function downloadBinaryArchive(downloadUrl, archivePath, version, archiveName) {
    console.log(`Downloading ${archiveName} (version: ${version}) to ${archivePath}...`);
    try {
        const writer = fs.createWriteStream(archivePath);
        const response = await axios({
            url: downloadUrl,
            method: 'GET',
            responseType: 'stream',
            timeout: 60000 // 1 minute timeout
        });
        response.data.pipe(writer);
        await new Promise((resolve, reject) => {
            writer.on('finish', resolve);
            writer.on('error', (err) => reject(new Error(`Failed to write archive to disk: ${err.message}`)));
            response.data.on('error', (err) => reject(new Error(`Failed during download stream: ${err.message}`)));
        });
        console.log('Download complete.');

        console.log(`Verifying checksum for ${archivePath}...`);
        const versionChecksums = allExpectedChecksums[version];
        if (!versionChecksums) {
            throw new Error(`Checksums not found for version ${version} in checksums.json. Please run the update script.`);
        }
        const expectedChecksum = versionChecksums[archiveName];
        if (!expectedChecksum) {
            throw new Error(
                `Checksum for ${archiveName} (version ${version}) not found in checksums.json. ` +
                `Please ensure it's defined or run the update script. ` +
                `Known archives for ${version}: ${Object.keys(versionChecksums).join(', ')}`
            );
        }
        if (expectedChecksum.startsWith("PLEASE_RUN_UPDATE_SCRIPT")) {
             throw new Error(
                `Placeholder checksum found for ${archiveName} (version ${version}). ` +
                `Please run the update script to populate actual checksums.`
            );
        }

        const actualChecksum = await calculateFileSha256(archivePath);

        if (actualChecksum !== expectedChecksum) {
            fs.unlinkSync(archivePath); // Delete the invalid file
            throw new Error(
                `Checksum mismatch for ${archiveName} (version ${version}).\n` +
                `Expected: ${expectedChecksum}\n` +
                `Actual:   ${actualChecksum}\n` +
                `The downloaded file has been deleted.`
            );
        }
        console.log('Checksum verified successfully.');

    } catch (error) {
        console.error(`Failed during binary download or checksum verification for ${archiveName} from ${downloadUrl}: ${error.message}`);
        if (error.response) {
            console.error('Download Response Status:', error.response.status);
        }
        if (fs.existsSync(archivePath)) fs.unlinkSync(archivePath); // Clean up partial download
        throw error;
    }
}

async function extractBinaryFromArchive(archivePath, archiveExtension, finalBinaryPath) {
    console.log(`Extracting binary from ${archivePath} to ${BIN_DIR}...`);
    try {
        if (archiveExtension === '.zip') {
            await extract(archivePath, { dir: BIN_DIR });
        } else if (archiveExtension === '.tar.gz') {
            await tar.x({
                file: archivePath,
                cwd: BIN_DIR,
            });
        }
        console.log('Extraction complete.');

        if (!fs.existsSync(finalBinaryPath)) {
            console.error(`Binary not found at ${finalBinaryPath} after extraction. Contents of ${BIN_DIR}:`);
            try {
                fs.readdirSync(BIN_DIR).forEach(file => console.log(`- ${file}`));
            } catch (e) {
                console.error(`Could not read contents of ${BIN_DIR}.`);
            }
            throw new Error(`Binary ${path.basename(finalBinaryPath)} not found in archive or not extracted correctly.`);
        }
    } catch (error) {
        console.error(`Failed to extract binary: ${error.message}`);
        throw error;
    } finally {
        if (fs.existsSync(archivePath)) {
            fs.unlinkSync(archivePath);
            console.log(`Cleaned up ${archivePath}.`);
        }
    }
}

function ensureBinaryIsExecutable(binaryPath, platform) {
    if (platform !== 'win32') {
        try {
            fs.chmodSync(binaryPath, 0o755);
            console.log(`Set executable permission for ${binaryPath}`);
        } catch (error) {
            console.error(`Failed to set executable permission for ${binaryPath}: ${error.message}`);
            throw error;
        }
    }
}

async function main() {
    const binaryPath = getBinaryPath();
    if (fs.existsSync(binaryPath)) {
        console.log(`${PROJECT_NAME} binary already exists at ${binaryPath}. Skipping download.`);
        ensureBinaryIsExecutable(binaryPath, os.platform());
        return;
    }

    if (!fs.existsSync(BIN_DIR)) {
        fs.mkdirSync(BIN_DIR, { recursive: true });
    }

    const { goOs, goArchFilenamePart, archiveExtension, platform } = getPlatformDetails();
    
    const version = TEST_SERVER_VERSION;
    const archiveName = `${PROJECT_NAME}_${goOs}_${goArchFilenamePart}${archiveExtension}`;
    const downloadUrl = `https://github.com/${GITHUB_OWNER}/${GITHUB_REPO}/releases/download/${version}/${archiveName}`;
    const archivePath = path.join(BIN_DIR, archiveName);

    await downloadBinaryArchive(downloadUrl, archivePath, version, archiveName);
    await extractBinaryFromArchive(archivePath, archiveExtension, binaryPath);
    ensureBinaryIsExecutable(binaryPath, platform);

    console.log(`${PROJECT_NAME} binary is ready at ${binaryPath}`);
}

main().catch(err => {
    console.error("Error during test-server postinstall script:", err.message);
    // Log full error if available and has more details
    if (err.stack && err.message !== err.stack) {
        console.error(err.stack);
    }
    process.exit(1);
});
