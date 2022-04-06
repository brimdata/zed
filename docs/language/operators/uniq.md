### Operator

&emsp; **uniq** &mdash; deduplicate adjacent values

### Synopsis

```
uniq [-c]
```
### Description

Inspired by the traditional Unix shell command of the same name,
the `uniq` operator copies its input to its output but removes duplicate values
that are adjacent to one another.  

This operator is most often used with `cut` and `sort` to find and eliminate
duplicate values.

When run with the `-c` option, each value is output as a record with the
type signature `{value:any,count:uint64}`, where the `value` field contains the
unique value and the `count` field indicates the number of consecutive duplicates
that occurred in the input for that output value.

### Examples

_Simple deduplication_
```mdtest-command
echo '1 2 2 3' | zq -z uniq -
```
=>
```mdtest-output
1
2
3
```

_Simple deduplication with -c_
```mdtest-command
echo '1 2 2 3' | zq -z 'uniq -c' -
```
=>
```mdtest-output
{value:1,count:1(uint64)}
{value:2,count:2(uint64)}
{value:3,count:1(uint64)}
```
_Use sort to deduplicate non-adjacent values_
```mdtest-command
echo '"hello" "world" "goodbye" "world" "hello" "again"' | zq -z 'sort | uniq' -
```
=>
```mdtest-output
"again"
"goodbye"
"hello"
"world"
```
