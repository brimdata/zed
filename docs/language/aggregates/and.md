### Aggregate Function

&emsp; **and** &mdash; logical AND of input values

### Synopsis
```
and(bool) -> bool
```
### Description

The _and_ aggregate function computes the logical AND over all of its input.

### Examples

Anded value of simple sequence:
```mdtest-command
echo 'true false true' | zq -z 'and(this)' -
```
=>
```mdtest-output
false
```

Continuous AND of simple sequence:
```mdtest-command
echo 'true false true' | zq -z 'yield and(this)' -
```
=>
```mdtest-output
true
false
false
```
Unrecognized types are ignored and not coerced for truthiness:
```mdtest-command
echo 'true "foo" 0 false true' | zq -z 'yield and(this)' -
```
=>
```mdtest-output
true
true
true
false
false
```
