### Operator

&emsp; **sort** &mdash; sort values

### Synopsis

```
sort [-r] [-nulls first|last] [<expr> [, <expr> ...]]
```
### Description

The `sort` operator sorts its input by reading all values until the end of input,
sorting the values according to the provided sort expression(s), and emitting
the values in the sorted order.

By default, the sort order is ascending, from lowest value to highest.  If the `-r`
flag is provided, the sort order is descending.

Zed follows the SQL convention that, by default, `null` values appear last
in either case of ascending or descending sort.  This can be overridden
by specifying `-nulls first`.

If not all data fits in memory, values are spilled to temporary storage
and sorted with an external merge sort.

The sort expressions act as primary key, secondary key, and so forth.

If no sort expression is provided, a sort key is guessed based on heuristics applied
to the values present.
The heuristic examines the first input record and finds the first field in
left-to-right order that is an integer, or if no integer field is found,
the first field that is floating point. If no such numeric field is found, `sort` finds
the first field in left-to-right order that is _not_ of the `time` data type.
Note that there are some cases (such as the output of a grouped aggregation performed on heterogeneous data) where the first input record to `sort`
may vary even when the same query is executed repeatedly against the same data.
If you require a query to show deterministic output on repeated execution,
an explicit field list must be provided.

Note that a total order is defined over the space of all Zed values even
between values of different types so sort order is always well-defined even
when comparing heterogeneously typed values.

> TBD: document the definition of the total order

### Examples

_A simple sort with a null_
```mdtest-command
echo '2 null 1 3' | zq -z 'sort this' -
```
=>
```mdtest-output
1
2
3
null
```
_With no sort expression, sort will sort by `this` for non-records_
```mdtest-command
echo '2 null 1 3' | zq -z sort -
```
=>
```mdtest-output
1
2
3
null
```
_The "nulls last" default may be overridden_
```mdtest-command
echo '2 null 1 3' | zq -z 'sort -nulls first' -
```
=>
```mdtest-output
null
1
2
3
```
_With no sort expression, sort will find a numeric key_
```mdtest-command
echo '{s:"bar",k:2}{s:"bar",k:3}{s:"foo",k:1}' | zq -z sort -
```
=>
```mdtest-output
{s:"foo",k:1}
{s:"bar",k:2}
{s:"bar",k:3}
```
_It's best practice to provide the sort key_
```mdtest-command
echo '{s:"bar",k:2}{s:"bar",k:3}{s:"foo",k:1}' | zq -z 'sort k' -
```
=>
```mdtest-output
{s:"foo",k:1}
{s:"bar",k:2}
{s:"bar",k:3}
```
_Sort with a secondary key_
```mdtest-command
echo '{s:"bar",k:2}{s:"bar",k:3}{s:"foo",k:2}' | zq -z 'sort k,s' -
```
=>
```mdtest-output
{s:"bar",k:2}
{s:"foo",k:2}
{s:"bar",k:3}
```
_Sort with an expression_
```mdtest-command
echo '{s:"sum 2",x:2,y:0}{s:"sum 3",x:1,y:2}{s:"sum 0",x:-1,y:-1}' | zq -z 'sort x+y' -
```
=>
```mdtest-output
{s:"sum 0",x:-1,y:-1}
{s:"sum 2",x:2,y:0}
{s:"sum 3",x:1,y:2}
```
