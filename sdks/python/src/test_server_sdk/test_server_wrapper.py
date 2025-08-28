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

import subprocess
import os
import sys
import threading
import time
import yaml
from pathlib import Path
from typing import Optional
import requests

PROJECT_NAME = "test-server"

class TestServer:
    """A context manager for running a Go test-server binary."""

    def __init__(self, config_path: str, recording_dir: str, mode: str = "replay"):
        self.config_path = Path(config_path).resolve()
        self.recording_dir = Path(recording_dir).resolve()
        self.mode = mode
        self.teardown_timeout: Optional[int] = 5
        self.process: Optional[subprocess.Popen] = None
        self._stdout_thread: Optional[threading.Thread] = None
        self._stderr_thread: Optional[threading.Thread] = None

    def _get_binary_path(self) -> Path:
        """
        Finds the platform-specific binary. If not found, attempts to run the
        installer script.
        """
        binary_name = f"{PROJECT_NAME}.exe" if sys.platform == "win32" else PROJECT_NAME
        binary_path = Path(__file__).parent / "bin" / binary_name

        # If the binary doesn't exist, try to install it
        if not binary_path.exists():
            raise FileNotFoundError(
                    f"test-server binary not found at {binary_path}."
            )   
        return binary_path

    def _read_stream(self, stream, stream_name: str):
        """Reads and prints lines from a subprocess stream."""
        while self.process and self.process.poll() is None:
            line = stream.readline()
            if not line:
                break
            print(f"[test-server {stream_name}] {line.strip()}")

    def _health_check(self, url: str, retries: int = 5, delay_sec: float = 0.1):
        """Performs a health check with exponential backoff."""
        for i in range(retries):
            try:
                response = requests.get(url, timeout=1)
                response.raise_for_status()
                print(f"Health check for {url} passed.")
                return
            except requests.exceptions.RequestException as e:
                print(f"Health check attempt {i + 1} failed for {url}. Error: {e}")
                backoff_delay = delay_sec * (2 ** i)
                time.sleep(backoff_delay)
        
        raise TimeoutError(f"Health check failed for {url} after {retries} retries.")

    def start(self):
        """Starts the test-server process and waits for it to be healthy."""
        binary_path = self._get_binary_path()
        args = [
            str(binary_path),
            self.mode,
            "--config",
            str(self.config_path),
            "--recording-dir",
            str(self.recording_dir),
        ]

        print(f"Starting test-server in '{self.mode}' mode...")
        print(f"   Command: {' '.join(args)}")

        self.process = subprocess.Popen(
            args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding='utf-8',
        )

        self._stdout_thread = threading.Thread(
            target=self._read_stream, args=(self.process.stdout, "STDOUT"), daemon=True
        )
        self._stderr_thread = threading.Thread(
            target=self._read_stream, args=(self.process.stderr, "STDERR"), daemon=True
        )
        self._stdout_thread.start()
        self._stderr_thread.start()

        # Parse the config file to find the health check URL(s)
        print(f"Reading config file for health check endpoints: {self.config_path}")
        try:
            with open(self.config_path, 'r') as f:
                config = yaml.safe_load(f)

            if 'endpoints' in config and config['endpoints']:
                for endpoint in config['endpoints']:
                    # Check if the endpoint has a health check path defined
                    if 'health' in endpoint and 'source_port' in endpoint:
                        port = endpoint['source_port']
                        path = endpoint['health']
                        
                        # Construct the full URL and run the check
                        health_url = f"http://localhost:{port}{path}"
                        print(f"Performing health check on: {health_url}")
                        self._health_check(health_url)
        except FileNotFoundError:
            print(f"Config file not found at {self.config_path}. Skipping health check.")
            self.stop()
            raise FileNotFoundError
        except Exception as e:
            print(f"Error parsing config file or running health check: {e}")
            self.stop()
            raise e

    def stop(self):
        """Stops the test-server process gracefully."""
        if not self.process or self.process.poll() is not None:
            print("Server not running or already stopped.")
            return

        print(f"Stopping test-server process (PID: {self.process.pid})...")
        self.process.terminate()  # Sends SIGTERM (graceful shutdown)
        try:
            self.process.wait(timeout=self.teardown_timeout)
            print("Server terminated gracefully.")
        except subprocess.TimeoutExpired:
            print("Server did not respond to SIGTERM. Sending SIGKILL...")
            self.process.kill()  # Sends SIGKILL (forceful shutdown)
            self.process.wait()
            print("Server killed.")
        
        # Clean up the reader threads
        if self._stdout_thread:
            self._stdout_thread.join()
        if self._stderr_thread:
            self._stderr_thread.join()
        
    def __enter__(self):
        """Called when entering the 'with' block."""
        self.start()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Called when exiting the 'with' block. Ensures cleanup."""
        self.stop()
