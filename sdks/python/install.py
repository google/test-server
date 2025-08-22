# Copyright 2025 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import hashlib
import os
import platform
import stat
import sys
import tarfile
import zipfile
import json
from pathlib import Path
import requests

# --- Configuration ---
TEST_SERVER_VERSION = "v0.2.7"
GITHUB_OWNER = "google"
GITHUB_REPO = "test-server"
PROJECT_NAME = "test-server"
BIN_DIR = Path(__file__).parent / "bin"

CHECKSUMS_PATH = Path(__file__).parent / "checksums.json"
try:
    with open(CHECKSUMS_PATH, "r") as f:
        ALL_EXPECTED_CHECKSUMS = json.load(f)
except (FileNotFoundError, json.JSONDecodeError) as e:
    print(f"Error loading checksums.json: {e}")
    sys.exit(1)


def get_platform_details():
    """Determines the OS and architecture to download the correct binary."""
    os_platform = sys.platform
    arch = platform.machine()
    
    if os_platform.startswith("darwin"):
        go_os = "Darwin"
        archive_extension = ".tar.gz"
    elif os_platform.startswith("linux"):
        go_os = "Linux"
        archive_extension = ".tar.gz"
    elif os_platform.startswith("win32"):
        go_os = "Windows"
        archive_extension = ".zip"
    else:
        raise OSError(f"Unsupported platform: {os_platform}")

    if arch in ["x86_64", "AMD64"]:
        go_arch = "x86_64"
    elif arch in ["arm64", "aarch64"]:
        go_arch = "arm64"
    else:
        raise OSError(f"Unsupported architecture: {arch}")
        
    binary_name = f"{PROJECT_NAME}.exe" if go_os == "Windows" else PROJECT_NAME
    return go_os, go_arch, archive_extension, binary_name


def calculate_file_sha256(file_path):
    """Calculates and returns the SHA256 checksum of a file."""
    sha256 = hashlib.sha256()
    with open(file_path, "rb") as f:
        for chunk in iter(lambda: f.read(4096), b""):
            sha256.update(chunk)
    return sha256.hexdigest()


def download_and_verify(download_url, archive_path, version, archive_name):
    """Downloads the binary archive and verifies its checksum."""
    print(f"Downloading {archive_name} from {download_url}...")
    try:
        with requests.get(download_url, stream=True, timeout=60) as r:
            r.raise_for_status()
            with open(archive_path, "wb") as f:
                for chunk in r.iter_content(chunk_size=8192):
                    f.write(chunk)
        print("Download complete.")

        print("Verifying checksum...")
        expected_checksum = ALL_EXPECTED_CHECKSUMS.get(version, {}).get(archive_name)
        if not expected_checksum:
            raise ValueError(f"Checksum for {archive_name} (version {version}) not found.")
        
        actual_checksum = calculate_file_sha256(archive_path)
        if actual_checksum != expected_checksum:
            raise ValueError(f"Checksum mismatch! Expected {expected_checksum}, got {actual_checksum}")
        print("Checksum verified successfully.")

    except Exception as e:
        # Clean up partial download on failure
        if archive_path.exists():
            archive_path.unlink()
        print(f"Failed during download or verification: {e}")
        raise


def extract_archive(archive_path, archive_extension):
    """Extracts the binary from the downloaded archive."""
    print(f"Extracting binary from {archive_path} to {BIN_DIR}...")
    try:
        if archive_extension == ".zip":
            with zipfile.ZipFile(archive_path, "r") as zip_ref:
                zip_ref.extractall(BIN_DIR)
        elif archive_extension == ".tar.gz":
            with tarfile.open(archive_path, "r:gz") as tar_ref:
                tar_ref.extractall(BIN_DIR)
        print("Extraction complete.")
    finally:
        # Clean up the archive file
        if archive_path.exists():
            archive_path.unlink()
            print(f"Cleaned up {archive_path}.")


def ensure_binary_is_executable(binary_path, go_os):
    """Sets executable permissions on the binary for non-Windows systems."""
    if go_os != "Windows":
        st = os.stat(binary_path)
        os.chmod(binary_path, st.st_mode | stat.S_IEXEC)
        print(f"Set executable permission for {binary_path}")


def main():
    """Main function to orchestrate the installation."""
    go_os, go_arch, archive_extension, binary_name = get_platform_details()
    binary_path = BIN_DIR / binary_name

    if binary_path.exists():
        print(f"{PROJECT_NAME} binary already exists at {binary_path}. Removing it for a fresh install.")
        binary_path.unlink()  # This deletes the file

    BIN_DIR.mkdir(parents=True, exist_ok=True)
    
    version = TEST_SERVER_VERSION
    archive_name = f"{PROJECT_NAME}_{go_os}_{go_arch}{archive_extension}"
    download_url = f"https://github.com/{GITHUB_OWNER}/{GITHUB_REPO}/releases/download/{version}/{archive_name}"
    archive_path = BIN_DIR / archive_name

    try:
        download_and_verify(download_url, archive_path, version, archive_name)
        extract_archive(archive_path, archive_extension)
        ensure_binary_is_executable(binary_path, go_os)
        print(f"{PROJECT_NAME} binary is ready at {binary_path}")
    except Exception as e:
        sys.exit(1) # Exit with an error code

if __name__ == "__main__":
    main()
