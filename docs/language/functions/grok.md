### Function

&emsp; **grok** &mdash; parse a string using a grok pattern

### Synopsis

```
grok(p: string, s: string) -> any
grok(p: string, s: string, definitions: string) -> any
```

### Description

The _grok_ function parses a string `s` using grok pattern `p` and returns
a record containing the parsed fields. The syntax for pattern `p`
is `{%pattern:field_name}` where _pattern_ is the name of the pattern
to match in `s` and _field_name_ is the resultant field name of the capture
value.

When provided with three arguments, `definitions` is a string
of named patterns in the format `PATTERN_NAME PATTERN` each separated by newlines.
The named patterns can then be referenced in argument `p`.

#### Included Patterns

The _grok_ function by default includes a set of builtin named patterns
that can be referenced in any pattern. The included named patterns can be seen
[here](https://raw.githubusercontent.com/brimdata/zed/main/pkg/grok/base.go).

### Examples

Parsing a simple log line using the builtin named patterns:
```mdtest-command
echo '"2020-09-16T04:20:42.45+01:00 DEBUG This is a sample debug log message"' |
  zq -Z 'yield grok("%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:message}", this)' -
```
=>
```mdtest-output
{
    timestamp: "2020-09-16T04:20:42.45+01:00",
    level: "DEBUG",
    message: "This is a sample debug log message"
}
```
