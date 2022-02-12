### Function

&emsp; **flatten** &mdash; transform a record into a flattened map

### Synopsis

```
flatten(val: record) -> |{[string]:<any>}|
```
### Description
The _flatten_ function returns a map where each map key is a
string array of the path of each record field of `val` and the map value
is  the corresponding value of that field.
If there are multiple types for the leaf values in `val`, then the map's value type is
a union of the types present.

Note that maps do not have an order for their keys so converting a record
to a map in this fashion provides no means to retain the record's field order.

### Examples

```mdtest-command
echo '{a:1,b:{c:"foo"}}' | zq -z 'yield flatten(this)' -
```
=>
```mdtest-output
|{["a"]:1((int64,string)),["b","c"]:"foo"((int64,string))}|
```
