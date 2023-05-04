---
sidebar_position: 10
sidebar_label: Type Fusion
---

# Type Fusion

Type fusion is another important building block of data shaping.
Here, types are operated upon by fusing them together, where the
result is a single fused type.
Some systems call a related process "schema inference" where a set
of values, typically JSON, is analyzed to determine a relational schema
that all the data will fit into.  However, this is just a special case of
type fusion as fusion is fine-grained and based on Zed's type system rather
than having the narrower goal of computing a schema for representations
like relational tables, Parquet, Avro, etc.

Type fusion utilizes two key techniques.

The first technique is to simply combine types with a type union.
For example, an `int64` and a `string` can be merged into a common
type of union `(int64,string)`, e.g., the value sequence `1 "foo"`
can be fused into the single-type sequence:
```
1((int64,string))
"foo"((int64,string))
```
The second technique is to merge fields of records, analogous to a spread
expression.  Here, the value sequence `{a:1}{b:"foo"}` may be
fused into the single-type sequence:
```
{a:1,b:null(string)}
{a:null(int64),b:"foo"}
```

Of course, these two techniques can be powerfully combined,
e.g., where the value sequence `{a:1}{a:"foo",b:2}` may be
fused into the single-type sequence:
```
{a:1((int64,string)),b:null(int64)}
{a:"foo"((int64,string)),b:2}
```

To perform fusion, Zed currently includes two key mechanisms
(though this is an active area of development):
* the [`fuse` operator](../operators/fuse.md), and
* the [`fuse` aggregate function](../aggregates/fuse.md).

## Fuse Operator

The `fuse` operator reads all of its input, computes a fused type using
the techniques above, and outputs the result, e.g.,
```mdtest-command
echo '{x:1} {y:"foo"} {x:2,y:"bar"}' | zq -z fuse -
```
produces
```mdtest-output
{x:1,y:null(string)}
{x:null(int64),y:"foo"}
{x:2,y:"bar"}
```
whereas
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"}{x:2,y:"bar"}' | zq -z fuse -
```
requires a type union for field `x` and produces:
```mdtest-output
{x:1((int64,string)),y:null(string)}
{x:"foo"((int64,string)),y:"foo"}
{x:2((int64,string)),y:"bar"}
```

## Fuse Aggregate Function

The `fuse` aggregate function is most often useful during data exploration and discovery
where you might interactively run queries to determine the shapes of some new
or unknown input data and how those various shapes relate to one another.

For example, in the example sequence above, we can use the `fuse` aggregate function to determine
the fused type rather than transforming the values, e.g.,
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"}' | zq -z 'fuse(this)' -
```
results in
```mdtest-output
<{x:(int64,string),y:string}>
```
Since the `fuse` here is an aggregate function, it can also be used with
group-by keys.  Supposing we want to divide records into categories and fuse
the records in each category, we can use a group-by.  In this simple example, we
will fuse records based on their number of fields using the
[`len` function:](../functions/len.md)
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"}' | zq -z 'fuse(this) by len(this) | sort len' -
```
which produces
```mdtest-output
{len:1,fuse:<{x:int64}>}
{len:2,fuse:<{x:(int64,string),y:string}>}
```
Now, we can turn around and write a "shaper" for data that has the patterns
we "discovered" above, e.g., if this Zed source text is in `shape.zed`
```mdtest-input shape.zed
switch len(this) (
    case 1 => pass
    case 2 => yield shape(this, <{x:(int64,string),y:string}>)
    default => yield error({kind:"unrecognized shape",value:this})
)
```
when we run
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"} {a:1,b:2,c:3}' | zq -z -I shape.zed '| sort -r this' -
```
we get
```mdtest-output
{x:1}
{x:"foo"((int64,string)),y:"foo"}
{x:2((int64,string)),y:"bar"}
error({kind:"unrecognized shape",value:{a:1,b:2,c:3}})
```
