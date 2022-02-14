### Function

&emsp; **cast** &mdash; coerce a value to a different type

### Synopsis

```
cast(val: any, t: type) -> any
```

### Description

The _cast_ function performs type casts but handles both primitive types and
complex types.  If the input type `t` is a primitive type, then the result
is equivalent to
```
<t>(<val>)
```
e.g., the result of `cast(1, <string>)` is the same as `string(1)` which is `"1"`.

For complex types, the cast function visits each leaf value in `val` and
casts that value to the corresponding type in `t`.
When a complex value has multiple levels of nesting,
casting is applied recursively down the tree.  For example, cast is recursively
applied to each element in array of records and recursively applied to each record.

If `<val>` is a record (or if any of its nested value is a record):
* absent fields are ignored and omitted from the result,
* extra input fields are passed through unmodified to the result, and
* fields are matched by name and are order independent and the _input_ order is retained.

In other words, `cast` does not rearrange the order of fields in the input
to match the output type's order but rather just modifies the leaf values.

If a cast fails, an error is returned when casting to primitive types
and the input value is returned when casting to complex types.

### Examples

_Cast primitives to type `ip`_
```mdtest-command
echo '"10.0.0.1" 1 "foo"' | zq -z 'cast(this, <ip>)' -
```
produces
```mdtest-output
10.0.0.1
error("cannot cast 1 to type ip")
error("cannot cast \"foo\" to type ip")
```

_Cast a record to a different record type_
```mdtest-command
echo '{a:1,b:2}{a:3}{b:4}' | zq -z 'cast(this, <{b:string}>)' -
```
produces
```mdtest-output
{a:1,b:"2"}
{a:3}
{b:"4"}
```
