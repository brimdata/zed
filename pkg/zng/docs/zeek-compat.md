# ZNG Compatibility with Zeek Logs

THIS SECTION IS A WORK IN PROGRRESS.  WE DECIDED TO FOREGO COMPLETE
FORMAT-COMPATIBILITY BETWEEN ZNG AND ZEEK AND WILL INSTEAD DOCUMENT HERE THE
RELATIONSHIP BETWEEN ZEEK AND ZNG FORMAT.


## Zeek Type Mappings

| Type     | Alias    |
|----------|----------|
| int64    | int      |
| uint64   | count    |
| float64  | double   |
| ip       | addr     |
| net      | subnet   |
| duration | interval |
| array    | vector   |


## Legacy Directives

The legacy directives are backward compatible with the Zeek log format:
```
#separator<separator><char>
#set_separator<separator><char>
#empty_field<separator><string>
#unset_field<separator><string>
#path<separator><string>
#open<separator><zeek-ts>
#close<separator><zeek-ts>
#fields[<separator><string>]+
#types[<separator><type>]+
```
where `<separator>` is intialized to a space character at the beginning of the
stream, then overridden by the `#separator` directive.

In the legacy format, the `#separator` character and the `#set_separator` character
define how to parse both a legacy value line and a legacy descriptor.

Every legacy value line corresponds to a `record` type defined by the
fields and types directives possibly modified for the `#path` directive
as described below.

Record types may not be used in the `#types` directive,
which means there is no need to recursively parse the `set` and `vector`
container values (`set` and `vector` values are split according to
the `#set_separator` character).



## Legacy Values

If a value line begins with `#`, then this
character must be escaped.
(Such lines can only exist as legacy values and not regular values
as regular values always begin with an integer descriptor.)



A legacy value exists on any line that is not a directive and whose most
recent directive was a legacy directive.  The legacy value is parsed by simply
splitting the line using the `#separator` character, then splitting each container
value by the `#set_separator` character.
For example,
```
#separator \t
#set_separator  ,
#fields msg     list
#types  string  set[int]
hello\, world   1,2,3
```
represents the value
```
record(
    msg: string("hello, world")
    list: set[int](1, 2, 3)
)
```
The special characters that must be escaped if they appear within a value are:
* newline (`\n`)
* backslash (`\`)
* the `#separator` character (usually `\t`)
* the `#set_separator` character inside of set and vector values (usually `,`)
* `#unset_field` (usually `-`) if it appears as a value not be interpreted as "unset",

Similarly, a `set` with no values must be specified by the `#empty_field` string (usually `(empty)`)
to distinguish it from a `set` containing only a zero-length `string`, and this must be escaped if it
is a single-element set with the value `(empty)`, i.e., escaped as `\x28empty)`.

When processing legacy values, a column of type `string` named `_path` is
inserted into each value, provided a `#path` directive previously appeared in the
stream.  The contents of this `_path` field is set to the string value indicated
in the `#path` directive. It becomes the leftmost column in the value and all the other columns are shifted one space to
the right.

For example,
```
#separator \t
#set_separator  ,
#path   foo
#fields msg     list
#types  string  set[int]
hello, world    1,2,3
```
represents the value
```
record(
    _path: string("foo")
    msg: string("hello, world")
    list: set[int](1, 2, 3)
)
```
This allows existing Zeek log files to be easily
ingested into ZNG-aware systems while retaining the [Zeek log type](https://docs.zeek.org/en/stable/script-reference/log-files.html) as the
`_path` field.

To maintain backward compatibility where Zeek uses `.` in columns that
came from Zeek records, e.g., `id.orig_h`, such columns shall be converted by
a legacy parser into ZNG records.  Likewise, emitters of legacy
Zeek files shall flatten any records in the output by converting each sub-field
of a record to the corresponding flattened field name using dotted notation.
e.g.,
```
#separator ,
#fields id.orig_h,id.orig_p,id.resp_h,id.resp_p,message
#types addr,port,addr,port,string
```
would be interpreted as the following ZNG `record`:
```
record[id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port],message:string]
```

When nested records are flattened in the legacy format in this manner,
all sub-fields of a nested record must appear consecutively in the
`#fields` and `#types` directives.
Additionally, only one level of nesting is valid.  In other words,
flattened field names may contain a maximum of one dot (".") character.
