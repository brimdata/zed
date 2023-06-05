### Function

&emsp; **shape** &mdash;  apply cast, fill, and order

### Synopsis

```
shape(val: any, t: type) -> any
```

### Description

The _shape_ function applies the
[cast](cast.md),
[fill](fill.md), and
[order](order.md) functions to its input to provide an
overall data shaping operation.

Note that _shape_ does not perform a _crop_ function so
extra fields in the input are propagated to the output.

### Examples

_Shape input records_
```mdtest-command
echo '{b:1,a:2}{a:3}{b:4,c:5}' | zq -z 'shape(this, <{a:int64,b:string}>)' -
```
produces
```mdtest-output
{a:2,b:"1"}
{a:3,b:null(string)}
{a:null(int64),b:"4",c:5}
```
