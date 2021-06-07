# Expressions

Comprehensive documentation for Zed expressions is still a work in progress. In
the meantime, here's an example expression with simple math to get started:

```zq-command zed-sample-data/zeek-default
zq -f table 'duration > 100 | put total_bytes:=orig_bytes+resp_bytes | cut orig_bytes,resp_bytes,total_bytes' conn.log.gz
```

#### Output:
```zq-output head:5
ORIG_BYTES RESP_BYTES TOTAL_BYTES
32         0          32
32         0          32
406        1720       2126
32         31         63
...
```
