### Function

&emsp; **grep** &mdash; search strings inside of values

### Synopsis

```
grep(<pattern> [, e: any]) -> bool
```
### Description

The _grep_ function searches all of the strings in its input value `e`
(or `this` if `e` is not given)
 using the `<pattern>` argument, which must be a
[regular expression](../README.md#regular-expressions),
[glob pattern](../README.md#globs), or string literal.
If the pattern matches for any string, then the result is `true`.  Otherwise, it is `false`.

> Note that string matches are case insensitive while regular expression
> and glob matches are case sensitive.  In a forthcoming release, case sensitivity
> will be a expressible for all three pattern types.

The entire input value is traversed:
* for records, each field name is traversed and each field value is traversed or descended
if a complex type,
* for arrays and sets, each element is traversed or descended if a complex type, and
* for maps, each key and value is traversed or descended if a complex type.

### Examples

_Reach into nested records_
```mdtest-command
echo '{foo:10}{bar:{s:"baz"}}' | zq -z 'grep("baz")' -
```
=>
```mdtest-output
{bar:{s:"baz"}}
```
_It only matches string fields_
```mdtest-command
echo '{foo:10}{bar:{s:"baz"}}' | zq -z 'grep("10")' -
```
=>
```mdtest-output
```
_Match a field name_
```mdtest-command
echo '{foo:10}{bar:{s:"baz"}}' | zq -z 'grep("foo")' -
```
=>
```mdtest-output
{foo:10}
```
_Regular expression_
```mdtest-command
echo '{foo:10}{bar:{s:"baz"}}' | zq -z 'grep(/foo|baz/)' -
```
=>
```mdtest-output
{foo:10}
{bar:{s:"baz"}}
```
_Glob with a second argument_

```mdtest-command
echo '{s:"bar"}{s:"foo"}{s:"baz"}{t:"baz"}' | zq -z 'grep(b*, s)' -
```
=>
```mdtest-output
{s:"bar"}
{s:"baz"}
```
