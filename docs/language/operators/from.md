### Operator

&emsp; **from** &mdash; source data from pools, files, or URIs

### Synopsis

```
from <pool>[@<tag>] [ => <leg> ]
file <path> [format <format>]
get <uri> [format <format>]
from (
   pool <pool>[@<tag>] [ => <leg> ]
   file <path> [format <format>] [ => <leg> ]
   get <uri> [format <format>] [ => <leg> ]
   ...
)
```
### Description

The `from` operator identifies one or more data sources and transmits
their data to its output.  A data source can be
* the name of a data pool in a Zed lake;
* a path to a file; or
* an HTTP, HTTPS, or S3 URI.
Paths and URIs may be followed by an optional format specifier.

In the first three forms, a single source is connected to a single output.
In the fourth form, multiple sources are accessed in parallel and may be
[joined](join.md), [combined](combine.md), or [merged](merge.md).

A data path can be split with the `fork` operator as in
```
from PoolOne | fork (
  => op1 | op2 | ...
  => op1 | op2 | ...
) | merge ts | ...
```

Or multiple pools can be accessed and, for example, joined:
```
from (
  pool PoolOne => op1 | op2 | ...
  pool PoolTwo => op1 | op2 | ...
) | join on key=key | ...
```

Similarly, data can be routed to different paths with replication
using `switch`:
```
from ... | switch color (
  case "red" => op1 | op2 | ...
  case "blue" => op1 | op2 | ...
  default => op1 | op2 | ...
) | ...
```

The output of a fork consists of multiple legs that must be merged.
If the downstream operator expects a single input, then the output legs are
merged with an automatically inserted [combine operator](combine.md).

### Examples

_Copy input to two paths and merge_
```mdtest-command
echo '1 2' | zq -z 'fork (=>pass =>pass) | sort this' -
```
=>
```mdtest-output
1
1
2
2
```
