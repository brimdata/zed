### Function

&emsp; **grok** &mdash; parse a string using a grok pattern

### Synopsis

```
grok(pattern: string, s: string) -> any
grok(pattern: string, s: string, definitions: string) -> any
```

### Description

The _grok_ function parses a string using a grok pattern and returns
a record containing the parsed fields. The syntax for a grok pattern
is `{%pattern:field_name}` where _pattern_ is a the name of the pattern
to match text with and _field_name_ is resultant field name of the capture
value.

When provided with three arguments the third argument, definitions, is a string
of named patterns seperated by new lines in the format `PATTERN_NAME PATTERN`.
The named patterns can then be referenced in the grok pattern argument.

#### Included Patterns

The _grok_ function by default includes a set of builtin named patterns
that can be referenced in any pattern. The included named patterns can be seen
[here](https://raw.githubusercontent.com/brimdata/zed/main/pkg/grok/grok-patterns).

### Examples

Parsing a simple log line using the builtin named patterns:
```mdtest-command
echo '"2020-09-16T04:20:42.45+01:00 DEBUG This is a sample debug log message"' \
  | zq -Z 'yield grok("%{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:message}", this)' -
```
=>
```mdtest-output
{
    timestamp: "2020-09-16T04:20:42.45+01:00",
    level: "DEBUG",
    message: "This is a sample debug log message"
}
```
