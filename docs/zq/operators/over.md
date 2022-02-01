### Operator

&emsp; **over** &mdash; traverse nested values as a lateral query

### Synopsis

```
over <expr> [, <expr>...]
over <expr> [, <expr>...] [with <var>=<expr> [, ... <var>[=<expr>]] => ( <lateral> )
```
The `over` operator traverses complex values to create a new sequence
of derived values (e.g., the elements of an array) and either
(in the first form) sends the new values directly to its output or
(in the second form) sends the values to a scoped computation as indicated
by `<lateral>`, which may represent any Zed subquery operating on the
derived sequence of values as `this`.

Each expression `<expr>` is evaluated in left-to-right order and derived sequences are
generated from each such result depending on its types:
* an array value generates each of its element,
* a map value generates a sequence of records of the form `{key:<key>,value:<value>}` for each
entry in the map, and
* all other values generate a single value equal to itself.

Records can be converted to maps with the [_flatten_ function](../functions/flatten.md)
resulting in a map that can be traversed,
e.g., if `this` is a record, it can be traversed with `over flatten(this)`.

The nested subquery depicted as `<lateral>` is called a "lateral query" as the
outer query operates on the top-level sequence of values while the lateral
query operates on subsequences of values derived from each input value.
This pattern rhymes with the SQL pattern of a "lateral join", which runs a
SQL subquery for each row of the outer query's table.

In a Zed lateral query, each input value induces a derived subsequence and
for each such input, the lateral query runs to completion and yields its results.
In this way, operators like `sort` and `summarize`, which operate on their
entire input, run to completion for each subsequence and yield to the output the
lateral result set for each outer input as a sequence of values.

Within the lateral query, `this` refers to the values of the subsequence thereby
preventing lateral expressions from accessing the outer `this`.
To accommodate such references, the _over_ operator includes a _with_ clause
that binds arbitrary expressions evaluated in the outer scope
to variables that may be referenced by name in the lateral scope.

> Note that any such variable definitions override implied field references
> of `this`.  If a both a field named "x" and a variable named "x" need be
> referenced in the lateral scope, the field reference should be qualified as `this.x`
> while the variable is referenced simply as `x`.

Lateral queries may be nested to arbitrary depth and accesses to variables
in parent lateral query bodies follows lexical scoping.


### Examples

_Over evaluates each expression and emits it_
```mdtest-command
echo null | zq -z 'over 1,2,"foo"' -
```
=>
```mdtest-output
1
2
"foo"
```
_The over clause is evaluated once per each input value_
```mdtest-command
echo "null null" | zq -z 'over 1,2' -
```
=>
```mdtest-output
1
2
1
2
```
_Array elements are enumerated_
```mdtest-command
echo null | zq -z 'over [1,2],[3,4,5]' -
```
=>
```mdtest-output
1
2
3
4
5
```
_Over traversing an array_
```mdtest-command
echo '{a:[1,2,3]}' | zq -z 'over a' -
```
=>
```mdtest-output
1
2
3
```
_Filter the traversed values_

```mdtest-command
echo '{a:[6,5,4]} {a:[3,2,1]}' | zq -z 'over a | this % 2 == 0' -
```
=>
```mdtest-output
6
4
2
```
_Aggregate the traversed values_

```mdtest-command
echo '{a:[1,2]} {a:[3,4,5]}' | zq -z 'over a | sum(this)' -
```
=>
```mdtest-output
{sum:15}
```
_Aggregate the traversed values in a lateral query_
```mdtest-command
echo '{a:[1,2]} {a:[3,4,5]}' | zq -z 'over a => ( sum(this) )' -
```
=>
```mdtest-output
{sum:3}
{sum:12}
```
_Access the outer values in a lateral query_
```mdtest-command
echo '{a:[1,2],s:"foo"} {a:[3,4,5],s:"bar"}' | zq -z 'over a with s => (sum(this) | yield {s,sum})' -
```
=>
```mdtest-output
{s:"foo",sum:3}
{s:"bar",sum:12}
```
_Traverse a record by flattening it_
```mdtest-command
echo '{s:"foo",r:{a:1,b:2}} {s:"bar",r:{a:3,b:4}} ' | zq -z 'over flatten(r) with s => (yield {s,key:key[0],value})' -
```
=>
```mdtest-output
{s:"foo",key:"a",value:1}
{s:"foo",key:"b",value:2}
{s:"bar",key:"a",value:3}
{s:"bar",key:"b",value:4}
```
