### Operator

&emsp; **yield** &mdash; emit values from expressions

### Synopsis

```
yield <expr> [, <expr>...]
```
### Description

The `yield` operator produces output values by evaluating one or more
expressions on each input value and sending each result to the output
in left-to-right order.  Each `<expr>` may be any valid
[Zed expression](../README.md#expressions).

The _yield_ keyword may be omitted when `<expr>` is a
[record literal](../README.md#record-literal).

### Examples

_Hello, world_
```mdtest-command
echo null | zq -z 'yield "hello, world"' -
```
=>
```mdtest-output
"hello, world"
```
_Yield evaluates each expression for every input value_
```mdtest-command
echo 'null null null' | zq -z 'yield 1,2' -
```
=>
```mdtest-output
1
2
1
2
1
2
```
_Yield typically operates on its input_
```mdtest-command
echo '1 2 3' | zq -z 'yield this*2+1' -
```
=>
```mdtest-output
3
5
7
```
_Yield is often used to transform records_
```mdtest-command
echo '{a:1,b:2}{a:3,b:4}' | zq -z 'yield [a,b],[b,a] | collect(this) | yield collect' -
```
=>
```mdtest-output
[[1,2],[2,1],[3,4],[4,3]]
```
