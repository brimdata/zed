### Function

&emsp; **fill** &mdash; add null values for missing record fields

### Synopsis

```
fill(val: any, t: type) -> any
```

### Description

The _fill_ function adds to the input record `val` any fields that are
present in the output type `t` but not in the input.

Filled fields are added with a `null` value.  Filling is useful when
you want to be sure that all fields in a schema are present in a record.

If `val` is not a record, it is returned unmodified.

### Examples

_Fill a record_
```mdtest-command
echo '{a:1}' | zq -z 'fill(this, <{a:int64,b:string}>)' -
```
produces
```mdtest-output
{a:1,b:null(string)}
```

_Fill an array of records_
```mdtest-command
echo '[{a:1},{a:2}]' | zq -z 'fill(this, <[{a:int64,b:int64}]>)' -
```
produces
```mdtest-output
[{a:1,b:null(int64)},{a:2,b:null(int64)}]
```

_Non-records are returned unmodified_
```mdtest-command
echo '10.0.0.1 1 "foo"' | zq -z 'fill(this, <{a:int64,b:int64}>)' -
```
produces
```mdtest-output
10.0.0.1
1
"foo"
```
