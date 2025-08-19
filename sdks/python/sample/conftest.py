import pytest

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
