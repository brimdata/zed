### Operator

&emsp; **head** &mdash; copy leading values of input sequence

### Synopsis

```
head [ n ]
```
### Description

The `head` operator copies the first `n` values from its input to its output
and ends the sequence thereafter.  `n` must be an integer.

### Examples

_Grab first two values of arbitrary sequence_
```mdtest-command
echo '1 "foo" [1,2,3]' | zq -z 'head 2' -
```
=>
```mdtest-output
1
"foo"
```

_Grab the first record of a record sequence_
```mdtest-command
echo '{a:"hello"}{b:"world"}' | zq -z head -
```
=>
```mdtest-output
{a:"hello"}
```
