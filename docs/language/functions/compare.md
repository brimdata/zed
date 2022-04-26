### Function

&emsp; **compare** &mdash; returns an integer comparing two values

### Synopsis

```
compare(a: any, b: any) -> int64
```

### Description

The _compare_ function returns an integer comparing two values. The result will
be 0 if a is equal to b, +1 if a is greater than b, and -1 if a is less than b.
_compare_ differs from `<`, `>`, `<=`, `>=`, `==`, and `!=` in that it will
work for any type (e.g., `compare(1, "1")`). `null` values are the max value.

### Examples

```mdtest-command
echo '{a: 2, b: "1"}' | zq -z 'yield compare(a, b)' -
```
=>
```mdtest-output
-1
```
