import pytest
import requests
import json
from pathlib import Path


from src.test_server_wrapper import TestServer

SAMPLE_PACKAGE_ROOT = Path(__file__).resolve().parent
CONFIG_FILE_PATH = SAMPLE_PACKAGE_ROOT / "test-data" / "config" / "test-server-config.yml"
RECORDINGS_DIR = SAMPLE_PACKAGE_ROOT / "test-data" / "recordings"


class TestSampleWithServer:
    """A test suite that requires the test-server to be running."""

    @pytest.fixture(scope="class", autouse=True)
    def managed_server(self, test_server_mode):
        """
        A fixture that starts the test-server before any tests in this class run,
        and stops it after they have all finished.
        
        It uses the 'test_server_mode' fixture from conftest.py to determine
        whether to run in 'record' or 'replay' mode.
        """
        print(f"\n[PyTest] Using test-server mode: '{test_server_mode}'")
        
        # The TestServer context manager handles start and stop automatically
        with TestServer(
            config_path=str(CONFIG_FILE_PATH),
            recording_dir=str(RECORDINGS_DIR),
            mode=test_server_mode
        ) as server:
            print(f"[PyTest] Test-server started with PID: {server.process.pid}")
            # The 'yield' passes control to the tests. The server will be
            # stopped automatically when the tests in the class are done.
            yield server

    def test_should_receive_200_from_proxied_github(self):
        """Tests that a request to the proxy returns a successful response."""
        print("[PyTest] Making request to test-server proxy for www.github.com...")
        
        # Use the 'requests' library for a simpler HTTP call
        response = requests.get("http://localhost:17080/", timeout=10)

        # Pytest uses simple 'assert' statements for checks
        assert response.status_code == 200
        assert "github" in json.dumps(dict(response.headers))
        
        print("[PyTest] Received 200 OK, content check passed.")


class TestAnotherSampleWithoutServer:
    """A basic test suite that runs independently of the test-server."""

    def test_should_run_basic_check(self):
        """A simple, independent test case."""
        print("\n[PyTest] Running a test that does not manage the test-server.")
        assert True
