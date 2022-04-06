### Operator

&emsp; **fork** &mdash; copy values to parallel paths

### Synopsis

```
fork {
  => <leg>
  => <leg>
  ...
}
```
### Description

The `fork` operator copies each input value to multiple, parallel legs of
the dataflow path.

The output of a fork consists of multiple legs that must be merged.
If the downstream operator expects a single input, then the output legs are
merged with an automatically inserted [combine operator](combine.md).

### Examples

_Copy input to two paths and merge_
```mdtest-command
echo '1 2' | zq -z 'fork (=>pass =>pass) | sort this' -
```
=>
```mdtest-output
1
1
2
2
```
