### Operator

&emsp; **from** &mdash; source data from pools, files, or URIs

### Synopsis

```
from <pool>[@<commitish>]
from <pattern>
file <path> [format <format>]
get <uri> [format <format>]
from (
   pool <pool>[@<commitish>] [ => <leg> ]
   pool <pattern>
   file <path> [format <format>] [ => <leg> ]
   get <uri> [format <format>] [ => <leg> ]
   ...
)
```
### Description

The `from` operator identifies one or more data sources and transmits
their data to its output.  A data source can be
* the name of a data pool in a Zed lake, with optional [commitish](../../commands/zed.md#142-commitish);
* the names of multiple data pools, expressed as a [regular expression](../overview.md#711-regular-expressions) or [glob](../overview.md#712-globs) pattern;
* a path to a file; or
* an HTTP, HTTPS, or S3 URI.
Paths and URIs may be followed by an optional format specifier.

Sourcing data from pools is only possible when querying a lake, such as
via the [`zed` command](../../commands/zed.md) or
[Zed lake API](../../lake/api.md). Sourcing data from files is only possible
with the [`zq` command](../../commands/zq.md).

When a single pool name is specified without `@`-referencing a commit or ID, or
when using a pool pattern, the tip of the `main` branch of each pool is
accessed.

In the first four forms, a single source is connected to a single output.
In the fifth form, multiple sources are accessed in parallel and may be
[joined](join.md), [combined](combine.md), or [merged](merge.md).

A data path can be split with the [`fork` operator](fork.md) as in
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
using the [`switch` operator](switch.md):
```
from ... | switch color (
  case "red" => op1 | op2 | ...
  case "blue" => op1 | op2 | ...
  default => op1 | op2 | ...
) | ...
```

### Input Data

Examples below below assume the existence of the Zed lake created and populated
by the following commands:

```mdtest-command
export ZED_LAKE=example
zed -q init
zed -q create coinflips
echo '{flip:1,result:"heads"} {flip:2,result:"tails"}' | zed -q -use coinflips load -
zed -q -use coinflips branch trial 
echo '{flip:3,result:"heads"}' | zed -q -use coinflips@trial load -
zed -q create numbers
echo '{number:1,word:"one"} {number:2,word:"two"} {number:3,word:"three"}' | zed -q -use numbers load -
zed ls | awk '{ print $1 }' | sort
```

The lake then contains the two pools:

```mdtest-output
coinflips
numbers
```

The following file `hello.zson` is also used.

```mdtest-input hello.zson
{greeting:"hello world!"}
```

### Examples

_Source structured data from a local file_

```mdtest-command
zq -z 'file hello.zson | yield greeting'
```
=>
```mdtest-output
"hello world!"
```

_Source data from a local file, but in line format_
```mdtest-command
zq -z 'file hello.zson format line | head 1'
```
=>
```mdtest-output
"{greeting:\"hello world!\"}"
```

_Source structured data from a URI_
```mdtest-command
zq -z 'get https://raw.githubusercontent.com/brimdata/zui-insiders/main/package.json | yield productName'
```
=>
```mdtest-output
"Zui - Insiders"
```

_Source data from a pool, defaulting to the `main` branch_
```mdtest-command
zed -lake example query -z 'from coinflips'
```
=>
```mdtest-output
{flip:2,result:"tails"}
{flip:1,result:"heads"}
```

_Source data from a specific branch of a pool_
```mdtest-command
zed -lake example query -z 'from coinflips@trial'
```
=>
```mdtest-output
{flip:3,result:"heads"}
{flip:2,result:"tails"}
{flip:1,result:"heads"}
```

_Count the number of records in the `main` branch of all pools_
```mdtest-command
zed -lake example query -f text 'from * | count()'
```
=>
```mdtest-output
5
```
_Join the data from multiple pools_
```mdtest-command
zed -lake example query -z '
  from (
    pool coinflips => sort flip
    pool numbers => sort number
  ) | join on flip=number word'
```
=>
```mdtest-output
{flip:1,result:"heads",word:"one"}
{flip:2,result:"tails",word:"two"}
```
