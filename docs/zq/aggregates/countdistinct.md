### Aggregate Function

&emsp; **countdistinct** &mdash; count distinct input values

### Synopsis
```
countdistinct() -> uint64
```
### Description

The _countdistinct_ aggregate function computes the number of distinct values in its input.

### Examples

Anded value of simple sequence:
```mdtest-command-issue-3586
echo '1 2 2 3' | zq -z 'countdistinct()' -
```
=>
```mdtest-output-issue-3586
{count:3(uint64)}
```

Continuous count of simple sequence:
```mdtest-command-issue-3586
echo '1 2 2 3' | zq -z 'yield countdistinct()' -
```
=>
```mdtest-output-issue-3586
1(uint64)
2(uint64)
2(uint64)
3(uint64)
```
Mixed types are handled:
```mdtest-command-issue-3586
echo '1 "foo" 10.0.0.1' | zq -z 'yield countdistinct()' -
```
=>
```mdtest-output-issue-3586
1(uint64)
2(uint64)
3(uint64)
```
