# `zed` Python Package

The `zed` Python package provides a client for the REST API served by
[`zed lake serve`](../../cmd/zed/lake#serve).

## Installation

Install the latest version like this:
```sh
pip3 install "git+https://github.com/brimdata/zed#subdirectory=python/zed"
```

Install the version compatible with a local `zed` like this:
```sh

pip install "git+https://github.com/brimdata/zed@$(zed -version | cut -d ' ' -f 2)#subdirectory=python/zed"
```

## Example

Run a Zed lake service from your shell.
```sh
mkdir scratch
zed lake serve -R scratch
```
> Or you can launch the Brim app and it will run a Zed lake service
> on the default port at localhost:9867.

Then, from Python, create a pool, load some data, and query it.
```python
import zed

# Connect to the REST API at the default base URL (http://127.0.0.1:9867).
# To use a different base URL, supply it as an argument.
client = zed.Client()

c.create_pool('TestPool')

# Load some ZSON records from a string.  A file-like object also works.
# Data format is detected automatically and can be JSON, NDJSON, Zeek TSV,
# ZJSON, ZNG, or ZSON.
c.load('TestPool', '{s:"hello"} {s:"world"}')

# Begin executing a Zed query for all records in TestPool.
# This returns an iterator, not a container.
records = client.query('from TestPool'):

# Stream records from the server.
for record in records:
    print(record)
```
