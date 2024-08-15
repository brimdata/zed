### Operator

&emsp; **debug** &mdash; write intermediate values to stderr

### Synopsis

```
debug [ <expr> ]
```
### Description

The `debug` operator writes the value of `expr` to the debug channel. If no
`expr` is provided, `this` is written to the debug channel. If the query is
run on the command line via `zq` or `zed` all output written to the debug
channel is displayed on stderr.

The `debug` operator is useful to view intermediate values when debugging a
complex Zed query.

### Examples

The following query uses expressions containing [f-strings](../expressions.md#formatted-string-literals)
to display `"debug: foo"` on stderr whereas `"foo_bar"` will display
on stdout.
```
echo '"foo"' | zq -z 'debug f"debug: {this}" | yield f"{this}_bar"' -
```
