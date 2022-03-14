### Operator

&emsp; **from** &mdash; source data from pools, URIs, or connectors

### Synopsis

```
from <pool>[@<tag>] [range <start>] [to <end>] [ => <leg> ]
get <uri>
file <path>
from (
   pool <pool>[@<tag>] [range <start>] [to <end>] [ => <leg> ]
   get <uri> [ => <leg> ]
   file <path> [ => <leg> ]
   ...
)
```
### Description

The `from` operator identifies data from a source `<src>` and logically
transmits the data to its output.  A `<src>` is:
* a URI representing and HTTP endpoint, S3 endoint, or file; or,
* the name of a data pool in a Zed lake.

In the first form, a single source is connected to a single output.
In the second form, multiple sources are accessed in parallel and may be
[joined](join.md), [combined](combine.md), or [merged](merge.md).

In the examples above, the data source is implied.  For example, the
`zed query` command takes a list of files and the concatenated files
are the implied input.
Likewise, in the [Brim app](https://github.com/brimdata/brim),
the UI allows for the selection of a data source and key range.

Data sources can also be explicitly specified using the `from` keyword.
Depending on the operating context, `from` may take a file system path,
an HTTP URL, an S3 URL, or in the
context of a Zed lake, the name of a data pool.

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
