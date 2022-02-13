### Function

&emsp; **parse_zson** &mdash; parse ZSON text into a Zed value

### Synopsis

```
parse_zson(s: string) -> any
```
### Description

The _parse_zson_ function parses the `s` argument that must be in the form
of ZSON into a Zed value of any type.  This is analogous to JavaScript's
`JSON.parse()` function.


### Examples

```mdtest-command
echo '{foo:"{a:\"1\",b:2}"}' | zq -z 'foo := parse_zson(foo)' -
```

**Output:**
```mdtest-output
{foo:{a:"1",b:2}}
```
