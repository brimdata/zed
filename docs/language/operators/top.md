### Operator

&emsp; **top** &mdash; get top n sorted values of input sequence

### Synopsis

```
top <uint> <expr> [ <expr> ...]
```
### Description

The `top` operator returns the top n values from a sequence sorted in descending
order by one or more expressions. `top` is functionally similar to `sort` except
only the top n values are stored in memory (i.e., values less than the minimum
are discarded).

### Examples

_Grab top two values from a sequence of integers
```mdtest-command
echo '1 5 3 9 23 7' | zq -z 'top 2 this' -
```
=>
```mdtest-output
23
9
```
