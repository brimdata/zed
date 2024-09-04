### Function

&emsp; **has** &mdash; test existence of values

### Synopsis

```
has(val: any [, ... val: any]) -> bool
```

### Description

The _has_ function returns false if any of its arguments are `error("missing")`
and otherwise returns true.
`has(e)` is a shortcut for [`!missing(e)`](missing.md).

This function is most often used to test the existence of certain fields in an
expected record, e.g., `has(a,b)` is true when `this` is a record and has
the fields `a` and `b`, provided their values are not `error("missing")`.

It's also useful in shaping when applying conditional logic based on the
presence of certain fields:
```
switch (
  case has(a) => ...
  case has(b) => ...
  default => ...
)
```

### Examples

```mdtest-command
echo '{foo:10}' | zq -z 'yield {yes:has(foo),no:has(bar)}' -
echo '{foo:[1,2,3]}' | zq -z 'yield {yes: has(foo[0]),no:has(foo[3])}' -
echo '{foo:{bar:"value"}}' |
  zq -z 'yield {yes:has(foo.bar),no:has(foo.baz)}' -
echo '{foo:10}' | zq -z 'yield {yes:has(foo+1),no:has(bar+1)}' -
echo 1 | zq -z 'yield has(bar)' -
echo '{x:error("missing")}' | zq -z 'yield has(x)' -
```
=>
```mdtest-output
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
false
false
```
