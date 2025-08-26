To run the sample, nevigate to sdks/python/samples

```sh
# ensure a force update
pip install --force-reinstall ../dist/test_server_sdk-0.1.0-py3-none-any.whl

# check what is in the package
pip show -f test_server_sdk

# run test with replay mode
pytest -sv

# run test with record mode
pytest -sv --record
```