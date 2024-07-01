### Operator

&emsp; **load** &mdash; add and commit data to a pool

### Synopsis

```
load <pool>[@<branch>] [author <author>] [message <message>] [meta <meta>]
```

:::tip Note
The `load` operator is exclusively for working with pools in a
[Zed lake](../../commands/zed.md) and is not available for use in
[`zq`](../../commands/zq.md).
:::

### Description

The `load` operator populates the specified `<pool>` with the values it
receives as input. Much like how [`zed load`](../../commands/zed.md#load)
is used at the command line to populate a pool with data from files, streams,
and URIs, the `load` operator is used to save query results from your Zed
pipeline to a pool in the same Zed lake. `<pool>` is a string indicating the
[name or ID](../../commands/zed.md#data-pools) of the destination pool.
If the optional `@<branch>` string is included then the data will be committed
to an existing branch of that name, otherwise the `main` branch is assumed.
The `author`, `message`, and `meta` strings may also be provided to further
describe the committed data, similar to the same `zed load` options.

### Input Data

Examples below assume the existence of the Zed lake created and populated
by the following commands:

```mdtest-command
export ZED_LAKE=example
zed -q init
zed -q create -orderby flip:asc coinflips
zed branch -q -use coinflips onlytails
echo '{flip:1,result:"heads"} {flip:2,result:"tails"}' | zed load -q -use coinflips -
zed -q create -orderby flip:asc bigflips
zed query -f text 'from :branches | yield pool.name + "@" + branch.name | sort'
```

The lake then contains the two pools:

```mdtest-output
bigflips@main
coinflips@main
coinflips@onlytails
```

### Examples

_Modify some values, load them into the `main` branch of our empty `bigflips` pool, and see what was loaded_
```mdtest-command
zed -lake example query 'from coinflips | result:=upper(result) | load bigflips' > /dev/null
zed -lake example query -z 'from bigflips'
```
=>
```mdtest-output
{flip:1,result:"HEADS"}
{flip:2,result:"TAILS"}
```

_Add a filtered subset of records to our `onlytails` branch, while also adding metadata_
```mdtest-command
zed -lake example query 'from coinflips | result=="tails" | load coinflips@onlytails author "Steve" message "A subset" meta "\"Additional metadata\""' > /dev/null
zed -lake example query -z 'from coinflips@onlytails'
```
=>
```mdtest-output
{flip:2,result:"tails"}
```
