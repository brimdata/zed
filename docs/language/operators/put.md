### Operator

&emsp; **put** &mdash; add or modify fields of records

### Synopsis
```
put <field>:=<expr> [, <field>:=<expr> ...]
```
### Description

The `put` operator modifies its input with
one or more [field assignments](../overview.md#25-field-assignments).
Each expression is evaluated based on the input record
and the result is either assigned to a new field of the input record if it does not
exist, or the existing field is modified in its original location with the result.

New fields are append in left-to-right order to the right of existing record fields
while modified fields are mutated in place.

If multiple fields are written in a single `put`, all the new field values are
computed first and then they are all written simultaneously.  As a result,
a computed value cannot be referenced in another expression.  If you need
to re-use a computed result, this can be done by chaining multiple `put` operators.

The "put" keyword is optional since it is an
[implied operators](../overview.md#26-implied-operators).

Each `<field>` expression must be a field reference expressed as a dotted path or one more
constant index operations on `this`, e.g., `a.b`, `this["a"]["b"]`,
etc.

Each right-hand side `<expr>` can be any Zed expression.

For any input value that is not a record, an error is emitted.

Note that when the field references are all top level,
`put` is a special case of a `yield` with a
[record literal](../overview.md#6112-record-expressions)
using a spread operator of the form:
```
yield {...this, <field>:<expr> [, <field>:<expr>...]}
```

### Examples

_A simple put_
```mdtest-command
echo '{a:1,b:2}' | zq -z 'put c:=3' -
```
=>
```mdtest-output
{a:1,b:2,c:3}
```
_The `put` keyword may be omitted_
```mdtest-command
echo '{a:1,b:2}' | zq -z 'c:=3' -
```
=>
```mdtest-output
{a:1,b:2,c:3}
```
_A `put` operation can also be done with a record literal_
```mdtest-command
echo '{a:1,b:2}' | zq -z 'yield {...this, c:3}' -
```
=>
```mdtest-output
{a:1,b:2,c:3}
```
_Missing fields show up as missing errors_
```mdtest-command
echo '{a:1,b:2,c:3}' | zq -z 'put d:=e' -
```
=>
```mdtest-output
{a:1,b:2,c:3,d:error("missing")}
```
_Non-record input values generate errors_
```mdtest-command
echo '{a:1} 1' | zq -z 'b:=2' -
```
=>
```mdtest-output
{a:1,b:2}
error("put: not a record: 1")
```
