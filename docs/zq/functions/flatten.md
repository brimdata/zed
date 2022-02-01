### Function

&emsp; **flatten** &mdash; transform a record into a flattened map

### Synopsis

```
flatten(val: record) -> |{[string]:<any>}|
```
### Description
The _flatten_ function returns a map of the flattened key/values of the record
argument `val` where the map key is a
string array of the path to each flattened non-record value. If there are
multiple types for the leaf values in `val`, then the map's value type is
a union of the types present.

### Examples

```mdtest-command
echo '{a:1,b:{c:"foo"}}' | zq -z 'yield flatten(this)' -
```
=>
```mdtest-output
|{["a"]:1((int64,string)),["b","c"]:"foo"((int64,string))}|
```
