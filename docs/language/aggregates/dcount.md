### Aggregate Function

&emsp; **dcount** &mdash; count distinct input values

### Synopsis
```
dcount(<any>) -> uint64
```

### Description

The _dcount_ aggregation function uses hyperloglog to estimate distinct values
of the input in a memory efficient manner.

### Examples

Anded value of simple sequence:
```mdtest-command
echo '1 2 2 3' | zq -z 'dcount(this)' -
```
=>
```mdtest-output
3(uint64)
```

Continuous count of simple sequence:
```mdtest-command
echo '1 2 2 3' | zq -z 'yield dcount(this)' -
```
=>
```mdtest-output
1(uint64)
2(uint64)
2(uint64)
3(uint64)
```
Mixed types are handled:
```mdtest-command
echo '1 "foo" 10.0.0.1' | zq -z 'yield dcount(this)' -
```
=>
```mdtest-output
1(uint64)
2(uint64)
3(uint64)
```
