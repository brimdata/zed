### Function

&emsp; **crop** &mdash; remove fields from input value that are missing in a specified type

### Synopsis

```
crop(val: any, t: type) -> any
```

### Description

The _crop_ function operates on record values (or records within a nested value)
and returns a result such that any fields that are present in `val` but not in
record type `t` are removed.
Cropping is a useful when you want records to "fit" a schema tightly.

If `<val>` is a record (or if any of its nested value is a record):
* absent fields are ignored and omitted from the result,
* fields are matched by name and are order independent and the _input_ order is retained, and
* leaf types are ignored, i.e., no casting occurs.

If an `<val>` is not a record, it is returned unmodified.

### Examples

_Crop a record_
```mdtest-command
echo '{a:1,b:2}' | zq -z 'crop(this, <{a:int64}>)' -
```
produces
```mdtest-output
{a:1}
```

_Crop an array of records_
```mdtest-command
echo '[{a:1,b:2},{a:3,b:4}]' | zq -z 'crop(this, <[{a:int64}]>)' -
```
produces
```mdtest-output
[{a:1},{a:3}]
```

_Cropped primitives are returned unmodified_
```mdtest-command
echo '10.0.0.1 1 "foo"' | zq -z 'crop(this, <{a:int64}>)' -
```
produces
```mdtest-output
10.0.0.1
1
"foo"
```
