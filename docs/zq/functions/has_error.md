### Function

&emsp; **has_error** &mdash; test if a value has an error

### Synopsis

```
has_error(val: any [, ... val: any]) -> bool
```
### Description

The _has_error_ function returns true if its argument has an error.
_has_error_ is different from _is_error_ in that _has_error_ will recurse 
into value's leaves to determine if there is an error in the value.

### Examples

```mdtest-command
echo '{a:{b:"foo"}}' | zq -z 'yield has_error(this)' -
echo '{a:{b:"foo"}}' | zq -z 'a.x := a.y + 1 | yield has_error(this)' -
```
=>
```mdtest-output
false
true
```
