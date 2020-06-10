# Data Types

Comprehensive documentation for working with data types in ZQL is still a work
in progress. In the meantime, here's a few tips to get started with.

* Values read in by `zq` are stored internally and treated in expressions using one of the data types described in the [Typedefs](../../../zng/docs/spec.md#211-typedefs) section of the ZNG spec.
* See the [Zeek Type Mappings](../../../zng/docs/zeek-compat.md#zeek-type-mappings) table for details on which ZNG data types correspond to the [data types](https://docs.zeek.org/en/current/script-reference/types.html) that appear in Zeek logs.
* ZQL provides a [type casting](https://en.wikipedia.org/wiki/Type_conversion) syntax using `:` followed by a ZNG data type.

#### Example:

The value in the JSON input below would ordinarily be treated as a string, but we can cast it to an `ip` type. This allows a downstream `filter` to correctly find the value in a CIDR match.

```zq-command
echo '{"src": "192.168.1.5"}' | zq -t 'put src=src:ip | filter 192.168.1.0/24' -
```

#### Output:
```zq-output
#0:record[src:ip]
0:[192.168.1.5;]
```
