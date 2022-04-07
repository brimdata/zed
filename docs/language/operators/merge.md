### Operator

&emsp; **merge** &mdash; combine parallel paths into a single, ordered output

### Synopsis

```
( => ... => ...) | merge <expr> [, <expr>, ...]
```
### Description

The `merge` operator merges inputs from multiple upstream legs of
the dataflow path into a single output.  The order of values in the combined
output is determined by the `<expr>` arguments, which act as sort expressions
where the values from the upstream paths are forwarded based on these expressions.

### Examples

_Copy input to two paths and combine
```mdtest-command
echo '1 2' | zq -z 'fork (=>pass =>pass) | merge this' -
```
=>
```mdtest-output
1
1
2
2
```
