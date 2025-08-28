import pytest
import os
from pathlib import Path
from test_server_sdk.test_server_wrapper import TestServer


# 2. Get paths from environment variables or use defaults
config_path_from_env = os.getenv("TEST_CONFIG_PATH")
recordings_dir_from_env = os.getenv("TEST_RECORDINGS_DIR")

# 3. Set the final paths, converting the string from the env var to a Path object
SAMPLE_PACKAGE_ROOT = Path(__file__).resolve().parent
CONFIG_FILE_PATH = Path(config_path_from_env) if config_path_from_env else SAMPLE_PACKAGE_ROOT / "test-data" / "config" / "test-server-config.yml"
RECORDINGS_DIR = Path(recordings_dir_from_env) if recordings_dir_from_env else SAMPLE_PACKAGE_ROOT / "test-data" / "recordings"

def pytest_addoption(parser):
    """Adds the --record command-line option to pytest."""
    parser.addoption(
        "--record", action="store_true", default=False, help="Run test-server in record mode."
    )

@pytest.fixture(scope="session")
def test_server_mode(request):
    """
    Returns 'record' or 'replay' based on the --record command-line flag.
    This fixture can be used by any test.
    """
    return "record" if request.config.getoption("--record") else "replay"

@pytest.fixture(scope="class")
def managed_server(test_server_mode):
    """
    A fixture that starts the test-server before any tests in a class run,
    and stops it after they have all finished.
    """
    print(f"\n[PyTest] Using test-server mode: '{test_server_mode}'")
    
    # The TestServer context manager handles start and stop automatically
    with TestServer(
        config_path=str(CONFIG_FILE_PATH),
        recording_dir=str(RECORDINGS_DIR),
        mode=test_server_mode
    ) as server:
        print(f"[PyTest] Test-server started with PID: {server.process.pid}")
        # The 'yield' passes control to the tests.
        yield server
        # Code after yield runs after the last test in the class finishes
        print(f"\n[PyTest] Test-server with PID: {server.process.pid} stopped.")
