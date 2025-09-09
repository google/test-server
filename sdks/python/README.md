# Build the Python test server sdk

## Create virtual enviroment
```
python3 -m venv ~/env
source ~/env/bin/activate
```

## Install all the dependencies
```
pip3 install -r requirements.txt
```

## Build python wheel

```sh
# Ensure a clean build
rm -rf build/ dist/ src/*.egg-info/ *.egg-info && find . -depth -name "__pycache__" -type d -exec rm -rf {} \;
# Build the python wheel
python3 -m build
```

# User the Python test server sdk

## Installation of the Python Test Server sdk

```sh
# Ensure a clean installation
pip3 uninstall test-server-sdk -y
# Install from the dist, use --force-reinstall to alwasy install fresh
pip3 install --force-reinstall dist/test_server_sdk-0.1.0-py3-none-any.whl
# This is the command to download and verify the underlying golang executabel.
download_golang_executable

# Check on the files
pip show -f test_server_sdk
```
You should see something very similar to this output, note tehat the `../../../bin/download_golang_executable` entry exist.
```
Name: test-server-sdk
Version: 0.1.0
Summary: A python wrapper for test-server.
Home-page: https://github.com/google/test-server/sdks/python
Author:
Author-email: Google LLC <googleapis-packages@google.com>
License-Expression: Apache-2.0
Location: /usr/local/google/home/wanlindu/env/lib/python3.13/site-packages
Requires: PyYAML, requests
Required-by:
Files:
  ../../../bin/download_golang_executable
  test_server_sdk-0.1.0.dist-info/INSTALLER
  test_server_sdk-0.1.0.dist-info/METADATA
  test_server_sdk-0.1.0.dist-info/RECORD
  test_server_sdk-0.1.0.dist-info/REQUESTED
  test_server_sdk-0.1.0.dist-info/WHEEL
  test_server_sdk-0.1.0.dist-info/direct_url.json
  test_server_sdk-0.1.0.dist-info/entry_points.txt
  test_server_sdk-0.1.0.dist-info/licenses/LICENSE
  test_server_sdk-0.1.0.dist-info/top_level.txt
  test_server_sdk/__init__.py
  test_server_sdk/__pycache__/__init__.cpython-313.pyc
  test_server_sdk/__pycache__/install.cpython-313.pyc
  test_server_sdk/__pycache__/test_server_wrapper.cpython-313.pyc
  test_server_sdk/checksums.json
  test_server_sdk/install.py
  test_server_sdk/test_server_wrapper.py
  ```

## Python Configuring the Python Test Server SDK

The Python `TestServer` is a convenient wrapper around the core Go test-server executable. You can configure it using parameters that directly correspond to the Go server's command-line flags.

You have the flexibility to provide these settings by passing them directly to the `TestServer` class, using environment variables, or creating custom `pytest` fixtures.

### Configuration Options

| Go Flag / ENV | Initialization Parameter | Description | Default Value | Sample Implementation (refer to the `python/sample/conftest.py` file) |
| :--- | :--- | :--- | :--- | :--- |
| `record` / `replay` | **`mode`** | Sets the server to either `'record'` or `'replay'`. | `'replay'` | Set via the `--record` pytest flag. |
| `--config` | **`config_path`** | The file path to the server's configuration file. | -- | Set via environment variable. |
| `--recording-dir` | **`recording_dir`** | The directory for saving or retrieving recordings. | -- | Set via environment variable. |
| -- | **`teardown_timeout`**| An optional grace period (in seconds) to wait before forcefully shutting down the server. | `5` | Left out to use default value  |

