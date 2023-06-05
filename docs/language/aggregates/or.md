### Aggregate Function

&emsp; **or** &mdash; logical OR of input values

### Synopsis
```
or(bool) -> bool
```

### Description

The _or_ aggregate function computes the logical OR over all of its input.

### Examples

Ored value of simple sequence:
```mdtest-command
echo 'false true false' | zq -z 'or(this)' -
```
=>
```mdtest-output
true
```

Continuous OR of simple sequence:
```mdtest-command
echo 'false true false' | zq -z 'yield or(this)' -
```
=>
```mdtest-output
false
true
true
```
Unrecognized types are ignored and not coerced for truthiness:
```mdtest-command
echo 'false "foo" 1 true false' | zq -z 'yield or(this)' -
```
=>
```mdtest-output
false
false
false
true
true
```
