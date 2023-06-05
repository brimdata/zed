### Aggregate Function

&emsp; **avg** &mdash; average value

### Synopsis
```
avg(number) -> number
```

### Description

The _avg_ aggregate function computes the mathematical average value of its input.

### Examples

Average value of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'avg(this)' -
```
=>
```mdtest-output
2.5
```

Continuous average of simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'yield avg(this)' -
```
=>
```mdtest-output
1.
1.5
2.
2.5
```
Unrecognized types are ignored:
```mdtest-command
echo '1 2 3 4 "foo"' | zq -z 'avg(this)' -
```
=>
```mdtest-output
2.5
```
