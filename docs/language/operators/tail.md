### Operator

&emsp; **tail** &mdash; copy trailing values of input sequence

### Synopsis

```
tail [ n ]
```
### Description

The `tail` operator copies the last `n` values from its input to its output
and ends the sequence thereafter.  `n` must be an integer.

### Examples

_Grab last two values of arbitrary sequence_
```mdtest-command
echo '1 "foo" [1,2,3]' | zq -z 'tail 2' -
```
=>
```mdtest-output
"foo"
[1,2,3]
```

_Grab the last record of a record sequence_
```mdtest-command
echo '{a:"hello"}{b:"world"}' | zq -z tail -
```
=>
```mdtest-output
{b:"world"}
```
