# Functions

## Table of Contents

- [Bytes](#bytes)
  - [`from_base64`](#from_base64)
  - [`from_hex`](#from_hex)
  - [`ksuid`](#ksuid)
  - [`to_base64`](#to_base64)
  - [`to_hex`](#to_hex)
- [IPs](#ips)
  - [`network_of`](#network_of)
- [Math](#math)
  - [`abs`](#abs)
  - [`ceil`](#ceil)
  - [`floor`](#floor)
  - [`log`](#log)
  - [`pow`](#pow)
  - [`round`](#round)
  - [`sqrt`](#sqrt)
- [Parse](#parse)
  - [`parse_uri`](#parse_uri)
  - [`parse_zson`](#parse_zson)
- [Records](#records)
  - [`fields`](#fields)
  - [`unflatten`](#unflatten)
- [Strings](#strings)
  - [`join`](#join)
  - [`replace`](#replace)
  - [`rune_len`](#rule_len)
  - [`split`](#split)
  - [`to_lower`](#to_lower)
  - [`to_upper`](#to_upper)
  - [`trim`](#trim)
- [Time](#time)
  - [`bucket`](#bucket)
  - [`every`](#every)
  - [`now`](#now)
- [Types](#types)
  - [`is`](#is)
  - [`iserr`](#iserr)
  - [`kind`](#kind)
  - [`nameof`](#nameof)
  - [`quiet`](#quiet)
  - [`typename`](#typename)
  - [`typeof`](#typeof)
  - [`typeunder`](#typeunder)
- [Value Introspection](#value-introspection)
  - [`has`](#has)
  - [`len`](#len)
  - [`missing`](#missing)
  - [`under`](#under)

### Pseudo Types

For brevity, pseudo types — represented by a name wrapped in `<>` — are
used for arguments that accept a range of value types. They are:

| Name | Types |
| --- | --- |
| `<any>` | Accepts a value as any type as the argument. |
| `<float>` | `float32`, `float64` |
| `<int>` | `int8`, `int16`, `int32`, `int64`<br />`uint8`, `uint16`, `uint32`, `uint64` |
| `<number>` | `<int>`, `<float>` |
| `<stringy>` | `string`, `err` |
| `<timey>` | `time`, `<number>` |

## Bytes

### `from_base64`

```
from_base64(s <stringy>) -> bytes
```

`from_base64` decodes Base64 <stringy> `s` into a byte sequence.

#### Example:

```mdtest-command
echo '{foo:"aGVsbG8gd29ybGQ="}' | zq -z 'foo := string(from_base64(foo))' -
```

**Output:**

```mdtest-output
{foo:"hello world"}
```

### `from_hex`

```
from_hex(s <stringy>) -> bytes
```

`from_hex` decodes hexadecimal <stringy> `s` into a byte sequence.

#### Example:

```mdtest-command
echo '{foo:"68656c6c6f20776f726c64"}' | zq -z 'foo := string(from_hex(foo))' -
```

**Output:**
```mdtest-output
{foo:"hello world"}
```

### `ksuid`

```
ksuid(bytes) -> string
```

`ksuid` encodes a [KSUID](https://github.com/segmentio/ksuid) (a byte sequence of length 20) into
a Base62 string.

#### Example:

```mdtest-command
echo  '{id:0x0dfc90519b60f362e84a3fdddd9b9e63e1fb90d1}' | zq -z 'id := ksuid(id)' -
```

**Output:**
```mdtest-output
{id:"1zjJzTWWCJNVrGwqB8kZwhTM2fR"}
```

### `to_base64`

```
to_base64(s <stringy>) -> string
```

`to_base64` encodes <stringy> `s` into a Base64 string.

#### Example:

```mdtest-command
echo '{foo:"hello word"}' | zq -z 'foo := to_base64(foo)' -
```

**Output:**
```mdtest-output
{foo:"aGVsbG8gd29yZA=="}
```

### `to_hex`

```
to_hex(b bytes) -> string
```

`to_hex` encodes byte sequence `b` into a hexadecimal string.

#### Example:

```mdtest-command
echo '{foo:0x68656c6c6f20776f726c64}' | zq -z 'foo := to_hex(foo)' -
```

**Output:**
```mdtest-output
{foo:"68656c6c6f20776f726c64"}
```

## IPs

### `network_of`

```
network_of(s ip, [m (net,int,uint)) -> net
```

With two arguments, `network_of` returns the net of ip address `s` under mask `m`. `m`
can be a net or an <int>. With one argument,
`network_of` returns the default net for `s`.

#### Example:

```mdtest-command
echo '{foo:10.1.2.129}' | zq -z 'foo := network_of(foo, 255.255.255.128/25)' -
echo '{foo:10.1.2.129}' | zq -z 'foo := network_of(foo, 24)' -
echo '{foo:10.1.2.129}' | zq -z 'foo := network_of(foo)' -
```

**Output:**
```mdtest-output
{foo:10.1.2.128/25}
{foo:10.1.2.0/24}
{foo:10.0.0.0/8}
```

## Math

### `abs`

```
abs(n <number>) -> <number>
```

`abs` returns the absolute value of number `n`.

#### Example:

```mdtest-command
echo '{foo:-1}' | zq -z 'foo := abs(foo)' -
```

**Output:**
```mdtest-output
{foo:1}
```

### `ceil`

```
ceil(n <number>) -> <number>
```

`ceil` returns number `n` rounded up to the nearest integer.

#### Example:

```mdtest-command
echo '{foo:1.3}' | zq -z 'foo := ceil(foo)' -
```

**Output:**
```mdtest-output
{foo:2.}
```

### `floor`

```
floor(n <number>) -> <number>
```

`floor` returns number `n` rounded down to the nearest integer.

#### Example:

```mdtest-command
echo '{foo:1.7}' | zq -z 'foo := floor(foo)' -
```

**Output:**
```mdtest-output
{foo:1.}
```

### `log`

```
log(n <number>) -> float64
```

`log` returns the natural logarithm of number `n`.

#### Example:

```mdtest-command
echo '{foo:4}' | zq -z 'foo := log(foo)' -
```

**Output:**
```mdtest-output
{foo:1.3862943611198906}
```

### `pow`

```
pow(x <number>, y <number>) -> float64
```

`pow` returns the base-`x` exponential of `y`.

#### Example:

```mdtest-command
echo '{foo:2}' | zq -z 'foo := pow(foo, 5)' -
```

**Output:**
```mdtest-output
{foo:32.}
```

### `round`

```
round(n <number>) -> <number>
```

`round` returns the number `n` rounded to the nearest integer value.

#### Example:

```mdtest-command
echo '{foo:3.14}' | zq -z 'foo := round(foo)' -
```

**Output:**
```mdtest-output
{foo:3.}
```

## Parse

### `parse_uri`

```
parse_uri(u <stringy>) -> record
```

`parse_uri` parses the [Universal Resource Identifier](https://en.wikipedia.org/wiki/Uniform_Resource_Identifier)
string `u` into a generic URI record. The returned record has the following
type:

```
({
  scheme: string,
  opaque: string,
  user: string,
  password: string,
  host: string,
  port: uint16,
  path: string,
  query: |{string:[string]}|,
  fragment: string
})
```

#### Example:

```mdtest-command
echo '{foo:"scheme://user:password@host:12345/path?a=1&a=2&b=3&c=#fragment"}' \
  | zq -Z 'foo := parse_uri(foo)' -
```

**Output:**
```mdtest-output
{
    foo: {
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

## Records

### `fields`

```
fields(r record) -> [string]
```

`fields` returns a string array of all the field names in record `r`.

#### Example:

```mdtest-command
echo '{foo:{a:1,b:2,c:3}}' | zq -z 'cut foo := fields(foo)' -
```

**Output:**
```mdtest-output
{foo:["a","b","c"]}
```

### `unflatten`

```
unflatten(r record) -> record
```

`unflatten` returns a copy of `r` with all dotted field names converted
into nested records. If no argument is supplied to `unflatten`, `unflatten`
operates on `this`.

#### Example:

```mdtest-command
echo '{"a.b.c":"foo"}' | zq -z 'yield unflatten()' -
```

**Output:**
```mdtest-output
{a:{b:{c:"foo"}}}
```

## Strings

### `join`

```
join(vals [<stringy>], sep <stringy>) -> string
```

`join` concatenates the elements of string array `vals` to create a single
string. The string `sep` is placed between each value in the resulting string.

#### Example:

```mdtest-command
echo '{foo:["a","b","c"]}' | zq -z 'foo := join(foo, ", ")' -
```

**Output:**
```mdtest-output
{foo:"a, b, c"}
```

### `replace`

```
replace(s <stringy>, old <stringy>, new <stringy>) -> string
```

`replace` replaces all instances of `old` occurring in string `s` with `new`.

#### Example:

```mdtest-command
echo '{foo:"oink oink oink"}' | zq -z 'foo := replace(foo, "oink", "moo")' -
```

**Output:**
```mdtest-output
{foo:"moo moo moo"}
```

### `rune_len`

```
rune_len(s <stringy>) -> int64
```

`rune_len` returns the number of runes in `p`. Erroneous and short encodings are
treated as single runes.

#### Example:

```mdtest-command
echo '{foo:"Yo! 😎"}' | zq -z 'foo := rune_len(foo)' -
```

**Output:**
```mdtest-output
{foo:5}
```

### `split`

```
split(s <stringy>, sep <stringy>) -> [string]
```

`split` slices `s` into all substrings separated by `sep` and returns an array
of the substrings between those separators.

#### Example:

```mdtest-command
echo '{foo:"apple;banana;pear;peach"}' | zq -z 'foo := split(foo,";")' -
```

**Output:**
```mdtest-output
{foo:["apple","banana","pear","peach"]}
```

### `to_lower`

```
to_lower(s <stringy>) -> string
```

`to_lower` lowercases all Unicode letters in `s`.

```mdtest-command
echo '{foo:"Zed"}' | zq -z 'foo := to_lower(foo)' -
```

**Output:**
```mdtest-output
{foo:"zed"}
```

### `to_upper`

```
to_upper(s <stringy>) -> string
```

`to_upper` uppercases all Unicode letters in `s`.

```mdtest-command
echo '{foo:"Zed"}' | zq -z 'foo := to_upper(foo)' -
```

**Output:**
```mdtest-output
{foo:"ZED"}
```

### `trim`

```
trim(s <stringy>) -> string
```

`trim` removes all leading and trailing whitespace from the string `s`.

```mdtest-command
echo '{foo:"   Zed   "}' | zq -z 'foo := trim(foo)' -
```

**Output:**
```mdtest-output
{foo:"Zed"}
```

## Time

### `now`

```
now() -> time
```

`now` returns the current UTC time.

```
echo '{}' | zq -z 'yield now()' -
```

**Output:**
```
2021-12-16T23:33:41.680643Z
```

### `bucket`

```
bucket(t (<timey>,duration), m (duration,<number>)) -> (time,duration)
```

<!-- XXX document time coercion rules  and link below-->

`bucket` returns time or duration `t` (or value that can be coerced to time) rounded down to
the nearest multiple of duration `m`.

```mdtest-command
echo '{ts:2020-05-26T15:27:47Z}' | zq -z 'ts := bucket(ts, 1h)' -
```

**Output:**
```mdtest-output
{ts:2020-05-26T15:00:00Z}
```

### `every`

```
every(d duration) -> time
```

`every` returns time of field `ts` rounded down to the nearest multiple of
duration `d`. The context value for each call to `every` must be a record with
a field `ts` in it. `every` is functionally equivalent to `bucket(ts, d)`.

```mdtest-command
echo '{ts:2020-05-26T15:27:47Z}' | zq -z 'ts := every(1h)' -
```

**Output:**
```mdtest-output
{ts:2020-05-26T15:00:00Z}
```

### `len`

```
len(v (record,array,set,map,type,bytes,string,ip,net,error)) -> int64
```

`len` returns the length of value `v`. Supported types:

- record
- array
- set
- map
- type
- bytes
- string
- ip
- net
- error

#### Example:

```mdtest-command
echo '{foo:[1,2,3]}' | zq -z 'foo := len(foo)' -
```

**Output:**
```mdtest-output
{foo:3}
```

## Types

### `is`

```
is([s <any>], t type) -> bool
```

`is` returns true if the subject value `s` has type `t`. `is` can accept either
one arguments or two: Value `s` is optional and if omitted the subject value is
the root value (i.e. equivalent to `is(this, type)`).

#### Example:

```mdtest-command
echo '{foo:1.}' | zq -z 'foo := is(foo, <float64>)' -
echo '{foo:1.}' | zq -z 'foo := is(<{foo:float64}>)' -
```

**Output:**
```mdtest-output
{foo:true}
{foo:true}
```

### iserr

```
iserr(v <any>) -> bool
```

`iserr` returns true if value `v` is of type error.

#### Example:

```mdtest-command
echo '{foo:error("this is an error")}' | zq -z 'foo := iserr(foo)' -
```

**Output:**
```mdtest-output
{foo:true}
```

### nameof

```
nameof(v <any>) -> string
```

`nameof` returns the string type name of `v` if `v` is a named type.

### kind

```
kind(v <any>) -> string
```

`kind` returns the category of the type of `v`, e.g., "record",
"set", "primitive", etc.  If `v` is a type value, then the type category
of the referenced type is returned.

### `quiet`

```
quiet(a <any>) -> type
```

`quiet` returns `a` unless `a` is `error("missing")` in which case
it returns `error("quiet")`.  Quiet errors are ignored by operators
`put`, `summarize`, and `yield`.

#### Example:

```mdtest-command
echo  '{x:error("missing"),y:"hello"}'  | zq -z 'cut x:=quiet(x), y:=quiet(y)' -
```

**Output:**
```mdtest-output
{y:"hello"}
```

### `typename`

```
typename(name <string>) -> type
```

`typename` returns the [type](../formats/zson.md#357-type-type) of the
named type give by `name` if it exists in the current context.  Otherwise,
`error("missing")` is returned.

#### Example:

```mdtest-command
echo  '80(port=int16)' | zq -z 'yield typename("port")' -
```

**Output:**
```mdtest-output
<port=int16>
```

### `typeof`

```
typeof(a <any>) -> type
```

`typeof` returns the [type](../formats/zson.md#357-type-type) of value `a`.

#### Example:

```mdtest-command
echo  '{foo:127.0.0.1}' \
  | zq -z 'foo := typeof(foo)' -
```

**Output:**
```mdtest-output
{foo:<ip>}
```

### `typeunder`

```
typeunder(a <any>) -> type
```

`typeunder` returns the [type](../formats/zson.md#357-type-type) of value `a`.
`typeunder` is similar to `typeof` except that if `a` is [named type](../formats/zson.md#357-type-type)
the type under `a` is returned.

#### Example:

```mdtest-command
echo  '{flavor:"chocolate"}(=flavor)' \
  | zq -z 'cut typeunder := typeunder(this)' -
```

**Output:**
```mdtest-output
{typeunder:<{flavor:string}>}
```

## Value Introspection

### `missing`

```
missing(e <expression>) -> bool
```

`missing` returns true if a value in [expression](expressions.md) `e` is
missing. Typically `e` is a selector expression (e.g., `foo.bar` or `foo[0]`,
`foo`) but `missing` will also return true if a variable in a generic
expression cannot be found (e.g., `foo+1`).

#### Example:

```mdtest-command
echo '{foo:[1,2,3]}' | zq -z 'cut yes := missing(foo[3]), no := missing(foo[0])' -
echo '{foo:{bar:"value"}}' | zq -z 'cut yes := missing(foo.baz), no := missing(foo.bar)' -
echo '{foo:10}' | zq -z 'cut yes := missing(bar), no := missing(foo)' -
echo '{foo:10}' | zq -z 'cut yes := missing(bar+1), no := missing(foo+1)' -
```

**Output:**
```mdtest-output
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
```

### `has`

```
has(e ...<expression>) -> bool
```

`has` returns true if a value exists for every [expression](expressions.md) in
the list `e`. `has(e)` is functionally equivalent to [`!missing(e)`](#missing).

#### Example:

```mdtest-command
echo '{foo:[1,2,3]}' | zq -z 'cut yes := has(foo[0]), no := has(foo[3])' -
echo '{foo:{bar:"value"}}' | zq -z 'cut yes := has(foo.bar), no := has(foo.baz)' -
echo '{foo:10}' | zq -z 'cut yes := has(foo), no := has(bar)' -
echo '{foo:10}' | zq -z 'cut yes := has(foo+1), no := has(bar+1)' -
```

**Output:**
```mdtest-output
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
{yes:true,no:false}
```


### `under`

```
under(e <expression>) -> <any>
```

`under` returns the value underlying the expression `e`:
* for unions, it returns the value as its elemental type of the union,
* for errors, it returns the value that the error wraps,
* for types, it returns the value typed as `typeunder()` indicates; ortherwise,
* the it returns the value unmodified.

#### Example:

Unions are unwrapped:

```mdtest-command
echo '1((int64,string)) "foo"((int64,string))' | zq -z 'yield this' -
echo '1((int64,string)) "foo"((int64,string))' | zq -z 'yield under(this)' -
```

**Output:**
```mdtest-output
1((int64,string))
"foo"((int64,string))
1
"foo"
```

Errors are unwrapped:

```mdtest-command
echo 'error("foo") error({err:"message"})' | zq -z 'yield this' -
echo 'error("foo") error({err:"message"})' | zq -z 'yield under(this)' -
```

**Output:**
```mdtest-output
error("foo")
error({err:"message"})
"foo"
{err:"message"}
```

Values of named types are unwrapped:

```mdtest-command
echo '80(port=uint16)' | zq -z 'yield this' -
echo '80(port=uint16)' | zq -z 'yield under(this)' -
```

**Output:**
```mdtest-output
80(port=uint16)
80(uint16)
```

Values that are not wrapped are return unmodified:
```mdtest-command
echo '1 "foo" <int16> {x:1}' | zq -z 'yield under(this)' -
```

**Output:**
```mdtest-output
1
"foo"
<int16>
{x:1}
```
