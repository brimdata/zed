### Operator

&emsp; **summarize** &mdash; perform aggregations

### Synopsis

```
[summarize] [<field>:=]<agg> [where <expr>][, [<field>:=]<agg> [where <expr>] ...] [by [<field>][:=<expr>] ...]
```
### Description

The `summarize` operator consumes all of its input, applies an [aggregate function](../aggregates/README.md)
to each input value optionally organized with the group-by keys specified after
the `by` keyword, and at the end of input produces one or more aggregations
for each unique set of group-by key values.

The `summarize` keyword is optional since it is an
[implied operator](../overview.md#26-implied-operators).

Each aggregate function may be optionally followed by a `where` clause, which
applies a Boolean expression that indicates, for each input value,
whether to deliver it to that aggregate. (`where` clauses are analogous
to the [`where` operator](where.md).)

The output field names for each aggregate and each key are optional.  If omitted,
a field name is inferred from each right-hand side, e.g, the output field for the
[`count` aggregate function](../aggregates/count.md) is simply `count`.

A key may be either an expression or a field.  If the key field is omitted,
it is inferred from the expression, e.g., the field name for `by lower(s)`
is `lower`.

When the result of `summarize` is a single value (e.g., a single aggregate
function without group-by keys) and there is no field name specified, then
the output is that single value rather than a single-field record
containing that value.

If the cardinality of group-by keys causes the memory footprint to exceed
a limit, then each aggregate's partial results are spilled to temporary storage
and the results merged into final results using an external merge sort.
The same mechanism that spills to storage can also spill across the network
to a cluster of workers in an adaptive shuffle, though this is not yet implemented.

### Examples

Average the input sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'summarize avg(this)' -
```
=>
```mdtest-output
2.5
```

To format the output of a single-valued aggregation into a record, simply specify
an explicit field for the output:
```mdtest-command
echo '1 2 3 4' | zq -z 'summarize mean:=avg(this)' -
```
=>
```mdtest-output
{mean:2.5}
```

When multiple aggregate functions are specified, even without explicit field names,
a record result is generated with field names implied by the functions:
```mdtest-command
echo '1 2 3 4' | zq -z 'summarize avg(this),sum(this),count()' -
```
=>
```mdtest-output
{avg:2.5,sum:10,count:4(uint64)}
```

Sum the input sequence, leaving out the `summarize` keyword:
```mdtest-command
echo '1 2 3 4' | zq -z 'sum(this)' -
```
=>
```mdtest-output
10
```

Create integer sets by key and sort the output to get a deterministic order:
```mdtest-command
echo '{k:"foo",v:1}{k:"bar",v:2}{k:"foo",v:3}{k:"baz",v:4}' | zq -z 'set:=union(v) by key:=k' - | sort
```
=>
```mdtest-output
{key:"bar",set:|[2]|}
{key:"baz",set:|[4]|}
{key:"foo",set:|[1,3]|}
```

Use a `where` clause:
```mdtest-command
echo '{k:"foo",v:1}{k:"bar",v:2}{k:"foo",v:3}{k:"baz",v:4}' | zq -z 'set:=union(v) where v > 1 by key:=k' - | sort
```
=>
```mdtest-output
{key:"bar",set:|[2]|}
{key:"baz",set:|[4]|}
{key:"foo",set:|[3]|}
```
