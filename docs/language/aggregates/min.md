### Aggregate Function

&emsp; **min** &mdash; minimum value of input values

### Synopsis
```
min(...number) -> number
```
### Description

The _min_ aggregate function computes the minimum value of its input.

### Examples

Minimum value of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'min(this)' -
```
=>
```mdtest-output
1
```

Continuous minimum of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'yield min(this)' -
```
=>
```mdtest-output
1
1
1
1
```
Unrecognized types are ignored:
```mdtest-command
echo '1 2 3 4 "foo"' | zq -z 'min(this)' -
```
=>
```mdtest-output
1
```
