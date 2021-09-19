# `zed` Python Package

The `zed` Python package provides a client for the REST API served by
[`zed lake serve`](../../cmd/zed/lake#serve).

## Installation

Install the latest version like this:
```sh
pip install "git+https://github.com/brimdata/zed#subdirectory=python/zed"
```

Install the version compatible with a local `zed` like this:
```sh

pip install "git+https://github.com/brimdata/zed@$(zed -version | cut -d ' ' -f 2)#subdirectory=python/zed"
```

## Example

Run a Zed lake service from your shell.
```sh
mkdir scratch
cd scratch
zed lake serve
```
> Or you can launch the Brim app and it will run a Zed lake service
> on the default port at localhost:9867.

In another shell, create a pool and load some data.
```sh
zapi create TestPool
zapi use TestPool@main
echo '{s:"hello"} {s:"world"}' | zapi load -
```

Then query the pool from Python.
```python
import zed

# Connect to the REST API at the default base URL (http://127.0.0.1:9867).
# To use a different base URL, supply it as an argument.
client = zed.Client()

# Begin executing a Zed query for all records in the pool named
# "TestPool".  This returns an iterator, not a container.
records = zed.query('from TestPool'):

# Stream records from the server.
for record in records:
    print(record)
```
