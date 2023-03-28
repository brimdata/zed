### Operator

&emsp; **head** &mdash; copy leading values of input sequence

### Synopsis

```
head [ <expr> ]
```
### Description

The `head` operator copies the first `n` values, evaluated from `<expr>`, from its input to its output
and ends the sequence thereafter. `<expr>` must evaluate to a positive integer at compile time.

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

_Grab first two values of arbitrary sequence, using a different representation of two_
```mdtest-command
echo '1 "foo" [1,2,3]' | zq -z 'head 1+1' -
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
