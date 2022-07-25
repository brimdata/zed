### Function

&emsp; **typeof** &mdash; the type of a value

### Synopsis

```
typeof(val: any) -> type
```
### Description

The _typeof_ function returns the [type](../../formats/zson.md#25-types) of
its argument `val`.  Types in Zed are first class so the returned type is
also a Zed value.  The type of a type is type `type`.

### Examples

The types of various values:

```mdtest-command
echo  '1 "foo" 10.0.0.1 [1,2,3] {s:"foo"} null error("missing")' | zq -z 'yield typeof(this)' -
```
=>
```mdtest-output
<int64>
<string>
<ip>
<[int64]>
<{s:string}>
<null>
<error(string)>
```
The type of a type is type `type`:
```mdtest-command
echo null | zq -z 'yield typeof(typeof(this))' -
```
=>
```mdtest-output
<type>
```
