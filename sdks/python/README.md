# Create virtual enviroment
```
python3 -m venv ~/env
source ~/env/bin/activate
```

# Install all the dependencies
```
pip3 install -r requirements.txt
```

# Build python wheel

```
rm -rf build/ dist/ *.egg-info/ src/test_server_sdk/bin/ && find . -depth -name "__pycache__" -type d -exec rm -rf {} \;
python3 -m build
```
