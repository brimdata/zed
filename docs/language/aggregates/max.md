### Aggregate Function

&emsp; **max** &mdash; maximum value of input values

### Synopsis
```
max(number) -> number
```

### Description

The _max_ aggregate function computes the maximum value of its input.

### Examples

Maximum value of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'max(this)' -
```
=>
```mdtest-output
4
```

Continuous maximum of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'yield max(this)' -
```
=>
```mdtest-output
1
2
3
4
```
Unrecognized types are ignored:
```mdtest-command
echo '1 2 3 4 "foo"' | zq -z 'max(this)' -
```
=>
```mdtest-output
4
```
