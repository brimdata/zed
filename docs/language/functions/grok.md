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

The _grok_ function by default includes a set of built-in named patterns
that can be referenced in any pattern. The included named patterns can be seen
[here](https://raw.githubusercontent.com/brimdata/zed/main/pkg/grok/base.go).

#### Comparison to Other Implementations

Although Grok functionality appears in several other open source tools, it
lacks a formal specification. As a result, example parsing configurations
found via web searches may not all be seamlessly portable to Zed's _grok_
function.

[Logstash](https://www.elastic.co/logstash) was the first tool to widely
promote the approach via its
[Grok filter plugin](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html),
so this serves as the de facto reference implementation. Many articles have
been published by Elastic and others that provide helpful guidance on becoming
proficient in Grok. To help you adjust details as you apply the concepts in
Zed, highlights important differences between the Logstash and Zed
implementations.

NOTE: As these represent areas of possible future enhancements, links to open
issues are provided. If you find a functional gap significantly impacts your
ability to use Zed's _grok_ function, please add a comment to the relevant
issue describing your use case.

1. Logstash's Grok offers an optional data type conversion syntax,
e.g., `%{NUMBER:num:int}` to store `num` as an integer type instead of as a
string. Zed currently accepts this syntax but effectively ignores it and stores
all values as strings. Downstream use of Zed's [`cast` function](cast.md) can
be used instead for data type conversion.
([zed/4928](https://github.com/brimdata/zed/issues/4928))

2. Some Logstash Grok examples use an optional square bracket syntax for
storing a parsed value in a nested field, e.g., `%{GREEDYDATA:[nested][field]}`
to store a value into `{"nested": {"field": ... }}`. With Zed the more common
dot-separated field naming convention (e.g., `nested.field`) can be combined
with the downstream use of the [`nest_dotted` function](nest_dotted.md) to
store values in nested fields.
([zed/4929](https://github.com/brimdata/zed/issues/4929))

3. Zed's regular expressions syntax does not currently support the
"named capture" syntax shown in the
[Logstash docs](https://www.elastic.co/guide/en/logstash/current/plugins-filters-grok.html#_custom_patterns).
Instead use the the approach shown later in that section by including a
"custom pattern" in the `definitions` argument, e.g.,

``````
$ echo '"Jan  1 06:25:43 mailserver14 postfix/cleanup[21403]: BEF25A72965: message-id=<20130101142543.5828399CCAF@mailserver14.example.com>"' |
zq -z 'yield grok("%{SYSLOGBASE} %{POSTFIX_QUEUEID:queue_id}: %{GREEDYDATA:syslog_message}", this, "POSTFIX_QUEUEID [0-9A-F]{10,11}")' -
```

(except this currently causes the panic from https://github.com/brimdata/zed/issues/5008#issuecomment-1911188925)

4. "embedded newline" example from https://brimdata.slack.com/archives/CTSMAK6G7/p1706198177405479?thread_ts=1706196396.370159&cid=CTSMAK6G7

5. Other Oniguruma regexp deltas. Build from the [included patterns](#included-patterns).

NOTE: If you absolutely require features of Logstash's Grok that are not
present in Zed's implementation, note that you could create a Logstash-based
ingest pipeline and send its JSON output to Zed tools. Issue
[zed/3151](https://github.com/brimdata/zed/issues/3151) provides some tips for
getting started. If you pursue this approach, please add a comment to the
issue describing your use case or come talk to us on community Slack.

#### Grok Debugging

Use a Grok debugger, but note that they also may have their own limitations to shows listed above.

Talk to us on Slack

#### Known Bugs

https://github.com/brimdata/zed/issues/5008

### Examples

Parsing a simple log line using the built-in named patterns:
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
