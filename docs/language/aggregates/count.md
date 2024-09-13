### Aggregate Function

&emsp; **count** &mdash; count input values

### Synopsis
```
count() -> uint64
```

### Description

The _count_ aggregate function computes the number of values in its input.

### Examples

Count of values in a simple sequence:
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

Count of values in buckets grouped by key:
```mdtest-command
echo '{a:1,k:1} {a:2,k:1} {a:3,k:2}' | zq -z 'count() by k | sort' -
```
=>
```mdtest-output
{k:1,count:2(uint64)}
{k:2,count:1(uint64)}
```

Note that the number of input values are counted, unlike the [`len` function](../functions/len.md) which counts the number of elements in a given value:
```mdtest-command
echo '[1,2,3]' | zq -z 'count()' -
```
=>
```mdtest-output
1(uint64)
```
