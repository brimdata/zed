# Python Library

You can also use `zed` from Python.  After you install the Zed Python:
```
pip3 install "git+https://github.com/brimdata/zed#subdirectory=python/zed"
```
You can hit the Zed service from a Python program:
```python
import zed

# Connect to the default lake at http://localhost:9867.  To use a
# different lake, supply its URL via the ZED_LAKE environment variable
# or as an argument here.
client = zed.Client()

# Begin executing a Zed query for all records in the pool named "Demo".
# This returns an iterator, not a container.
records = client.query('from Demo')

# Stream records from the server.
for record in records:
    print(record)
```
See the [python/zed](python/zed) directory for more details.


The `zed` Python package provides a client for the REST API served by
[`zed serve`](../../cmd/zed/serve).

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
zed serve -R scratch
```
> Or you can launch the Brim app and it will run a Zed lake service
> on the default port at http://localhost:9867.

Then, from Python, create a pool, load some data, and query it.
```python
import zed

# Connect to the default lake at http://localhost:9867.  To use a
# different lake, supply its URL via the ZED_LAKE environment variable
# or as an argument here.
client = zed.Client()

client.create_pool('TestPool')

# Load some ZSON records from a string.  A file-like object also works.
# Data format is detected automatically and can be JSON, NDJSON, Zeek TSV,
# ZJSON, ZNG, or ZSON.
client.load('TestPool', '{s:"hello"} {s:"world"}')

# Begin executing a Zed query for all records in TestPool.
# This returns an iterator, not a container.
records = client.query('from TestPool')

# Stream records from the server.
for record in records:
    print(record)
```
