### Operator

&emsp; **tail** &mdash; copy trailing values of input sequence

### Synopsis

```
tail [ <expr> ]
```
### Description

The `tail` operator copies the last `n` values, evaluated from `<expr>`, from its input to its output
and ends the sequence thereafter. `<expr>` must evaluate to a positive integer at compile time.

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

_Grab last two values of arbitrary sequence, using a different representation of two_
```mdtest-command
echo '1 "foo" [1,2,3]' | zq -z 'tail 1+1' -
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
