import pytest
import requests
import json
from pathlib import Path


@pytest.mark.usefixtures("managed_server")
class TestSampleWithServer:
    """A test suite that requires the test-server to be running."""

    def test_should_receive_200_from_proxied_github_single_name(self):
        """Tests that a request to the proxy returns a successful response."""
        print("[PyTest] Making request to test-server proxy for www.github.com...")

        custom_headers = {
            'Test-Name': 'python-sample-test-single-name',
        }   
        
        # Use the 'requests' library for a simpler HTTP call
        response = requests.get("http://localhost:17080/", headers=custom_headers, timeout=10)

        # Pytest uses simple 'assert' statements for checks
        assert response.status_code == 200
        assert "github" in json.dumps(dict(response.headers))
        
        print("[PyTest] Received 200 OK, content check passed.")
    
    def test_should_receive_200_from_proxied_github_recursive_path(self):
        """Tests that a request to the proxy returns a successful response."""
        print("[PyTest] Making request to test-server proxy for www.github.com...")

        custom_headers = {
            'Test-Name': 'models/folder1/folder2/python-sample-test_recursive_path',
        }   
        
        # Use the 'requests' library for a simpler HTTP call
        response = requests.get("http://localhost:17080/", headers=custom_headers, timeout=10)

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
