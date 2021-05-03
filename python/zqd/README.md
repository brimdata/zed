# `zqd` Python Package

The `zqd` Python package provides a client for the REST API served by
[`zed serve` (aka `zqd`)](../../cmd/zed#zqd).

## Installation

Install the latest version like this:
```sh
pip install 'git+https://github.com/brimdata/zed#subdirectory=python/zqd'
```

Install the version compatible with a local `zqd` like this:
```sh

pip install "git+https://github.com/brimdata/zed@$(zqd -version | cut -d ' ' -f 2)#subdirectory=python/zqd"
```

## Example

```python
import zqd

# Connect to the REST API at the default base URL (http://127.0.0.1:9867).
# To use a different base URL, supply it as an argument.
client = zqd.Client()

# Begin executing a Zed query for all records, "*", in the space named
# "your_space".  This returns an iterator, not a container.
records = client.search('your_space', '*'):

# Stream records from the server.
for record in records:
    print(record)
```
