### Aggregate Function

&emsp; **count** &mdash; count input values

### Synopsis
```
count() -> uint64
```
### Description

The _count_ aggregate function computes the number of values in its input.

### Examples

Anded value of simple sequence:
```mdtest-command
echo '1 2 3' | zq -z 'count()' -
```
=>
```mdtest-output
3(uint64)
```

Continuous count of simple sequence:
```mdtest-command
echo '1 2 3' | zq -z 'yield count()' -
```
=>
```mdtest-output
1(uint64)
2(uint64)
3(uint64)
```
Mixed types are handled:
```mdtest-command
echo '1 "foo" 10.0.0.1' | zq -z 'yield count()' -
```
=>
```mdtest-output
1(uint64)
2(uint64)
3(uint64)
```
