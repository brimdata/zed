### Aggregate Function

&emsp; **any** &mdash; select an arbitrary input value

### Synopsis
```
any(any) -> any
```
### Description

The _any_ aggregate function returns an arbitrary element from its input.
The semantics of how the item is selected is not defined.

### Examples

Any picks the first one in this scenario but this behavior is undefined:
```mdtest-command
echo '1 2 3 4' | zq -z 'any(this)' -
```
=>
```mdtest-output
1
```

Continuous any over a simple sequence:
```mdtest-command
echo '1 2 3 4' | zq -z 'yield any(this)' -
```
=>
```mdtest-output
1
1
1
1
```
Any is not sensitive to mixed types as it just picks one:
```mdtest-command
echo '"foo" 1 2 3 ' | zq -z 'any(this)' -
```
=>
```mdtest-output
"foo"
```
