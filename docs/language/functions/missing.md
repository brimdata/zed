### Function

&emsp; **missing** &mdash; test for the "missing" error

### Synopsis

```
missing(val: any) -> bool
```

### Description

The _missing_ function returns true if its argument is `error("missing")`
and false otherwise.

This function is often used to test if certain fields do not appear as
expected in a record, e.g., `missing(a)` is true either when `this` is not a record
or when `this` is a record and the field `a` is not present in `this`.

It's also useful in shaping when applying conditional logic based on the
absence of certain fields:
```
switch (
  case missing(a) => ...
  case missing(b) => ...
  default => ...
)
```

### Examples

```mdtest-command
echo '{foo:10}' | zq -z 'yield {yes:missing(bar),no:missing(foo)}' -
echo '{foo:[1,2,3]}' | zq -z 'yield {yes:has(foo[3]),no:has(foo[0])}' -
echo '{foo:{bar:"value"}}' |
  zq -z 'yield {yes:missing(foo.baz),no:missing(foo.bar)}' -
echo '{foo:10}' | zq -z 'yield {yes:missing(bar+1),no:missing(foo+1)}' -
echo 1 | zq -z 'yield missing(bar)' -
echo '{x:error("missing")}' | zq -z 'yield missing(x)' -
```
=>
```mdtest-output
{yes:true,no:false}
{yes:false,no:true}
{yes:true,no:false}
{yes:true,no:false}
true
true
```
