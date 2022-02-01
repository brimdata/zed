### Function

&emsp; **parse_zson** &mdash; parse ZSON text into a Zed value

### Synopsis

```
parse_zson(s: string) -> any
```
### Description

The _parse_zson_ function parses the `s` argument that must be in the form
of ZSON into a Zed value of any type.  This is analogous to Javascript's
`JSON.parse()` function.


### Examples

```mdtest-command
echo '"scheme://user:password@host:12345/path?a=1&a=2&b=3&c=#fragment"' | zq -Z 'yield parse_uri(this)' -
```
=>
```mdtest-output
{
    scheme: "scheme",
    opaque: null (string),
    user: "user",
    password: "password",
    host: "host",
    port: 12345 (uint16),
    path: "/path",
    query: |{
        "a": [
            "1",
            "2"
        ],
        "b": [
            "3"
        ],
        "c": [
            ""
        ]
    }|,
    fragment: "fragment"
}
```


### `parse_zson`

```
parse_zson(s <stringy>) -> <any>
```

`parse_zson` returns the value of the parsed ZSON string `s`.

#### Example:

```mdtest-command
echo '{foo:"{a:\"1\",b:2}"}' | zq -z 'foo := parse_zson(foo)' -
```

**Output:**
```mdtest-output
{foo:{a:"1",b:2}}
```
