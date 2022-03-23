# zq - command-line Zed query

## TL;DR

> `zq` is a command-line tool that uses the [Zed language](language.md)
for pipeline-style search and analytics.  `zq` can query a variety
of data formats in files, over HTTP, or in S3 storage.
It is particularly fast when operating on data in the Zed-native [ZNG](../formats/zng.md) format.
>
> The `zq` design philosophy blends the query/search-tool approach
of `jq`, `awk`, and `grep` with the command-line, embedded database approach
of `sqlite` and `duckdb`.

## Contents

* [1. Usage](#1-usage)
* [2. Input Formats](#2-input-formats)
  + [2.1 Hardwired Input Format](#21-hard-wired-input-format)
  + [2.2 Auto-detection](#22-auto-detection)
  + [2.3 ZSON-JSON Auto-detection](#23-zson-json-auto-detection)
* [3. Output Formats](#3-output-formats)
  + [3.1 Output Format Selection](#31-output-format-selection)
  + [3.2 ZSON Pretty Printing](#32-zson-pretty-printing)
  + [3.3 Pipeline-friendly ZNG](#33-pipeline-friendly-zng)
  + [3.4 Schema-rigid Outputs](#34-schema-rigid-outputs)
    - [3.4.1 Fusing Schemas](#341-fusing-schemas)
    - [3.4.2 Splitting Schemas](#342-splitting-schemas)
* [4. Query Debugging](#4-query-debugging)
* [5. Error Handling](#5-error-handling)
* [6. Examples](#6-examples)
* [7. Performance](#7-performance)
  + [7.1 Fast Pattern Matching](#71-fast-pattern-matching)
  + [7.2 Efficient JSON Processing](#72-efficient-json-processing)
  + [7.3 Performance Comparisons](#73-performance-comparisons)
    - [7.3.1 Test Data](#731-test-data)
    - [7.3.2 File Sizes](#732-file-sizes)
    - [7.3.3 Tests](#733-tests)
    - [7.3.4 Results](#734-results)

## 1. Usage

```
zq [ options ] [ query ] input [ input ... ]
```
`zq` is a command-line tool for processing data in diverse input
formats, providing search, analytics, and extensive transformations
using the [Zed language](language.md). A query typically applies Boolean logic
or keyword search to filter the input, then transforms or analyzes
the filtered stream.  Output is written to one or more files or to
standard output.

Each `input` argument must be a file path, an HTTP or HTTPS URL,
an S3 URL, or standard input specified with `-`.

For built-in command help and a listing of all available options,
simply run `zq` with no arguments.

`zq` supports a [number of formats](#2-input-formats) but [ZNG](../formats/zng.md)
tends to be the most space-efficient and most performant.  ZNG has efficiency similar to
[Avro](https://avro.apache.org/docs/current/spec.html)
and [Protocol Buffers](https://developers.google.com/protocol-buffers)
but its comprehensive [Zed type system](../formats/zed.md) obviates
the need for schema specification or registries.
Also, the ZSON format is human-readable and entirely one-to-one with ZNG
so there is no need to represent non-readable formats like Avro or Protocol Buffers
in a clunky JSON encapsulated form.  

`zq` typically operates on ZNG-encoded data and when you want to inspect
human-readable bits of output, you merely format it as ZSON, which is the
default format when output is directed to the terminal.  ZNG is the default
when redirecting to a non-terminal output like a file or pipe.

When run with input arguments, each input's format is automatically inferred
([as described below](#21-auto-detection)) and each input is scanned
in the order appearing on the command line forming the input stream.

A query expressed in the [Zed language](language.md)
may be optionally specified and applied to the input stream.

If no query is specified, the inputs are scanned without modification
and output in the desired format as described below.  This latter approach
provides a convenient means to convert files from one format to another.

To determine whether the first argument is a query or an input,
`zq` checks the local file system for the existence of a file by that name
or whether the name is an URL.
If no such file or URL exists, it attempts to parse the text as a Zed program.
If both checks fail, then an error is reported and `zq` exits.

This heuristic is convenient but can result in a rare surprise when a simple
Zed query (like a keyword search) happens to correspond with a file of the
same name in the local directory.
To avoid this, you can provide the query with the `-query` flag, which specifies
the Zed program to run and forces all arguments to be interpreted as inputs.

When `zq` is run with a query and no input arguments, then the query must
begin with a
* a [from, file, or get operator](operators/from.md), or
* an explicit or implied [yield operator](operators/yield.md).

In the case of a `yield` with no inputs, the query is run with
a single input value of `null`.  This provides a convenient means to run in a
"calculator mode" where input is produced by the yield and can be operated upon
by the Zed query, e.g.,
```mdtest-command
zq -z '1+1'
```
emits
```mdtest-output
2
```
Note here that the query `1+1` [implies](language.md#26-implied-operators)
`yield 1+1`.

## 2. Input Formats

`zq` currently supports the following input formats:

|  Option   | Auto | Specification                            |
|-----------|------|------------------------------------------|
| `json`    |  yes | [JSON RFC 8259](https://www.rfc-editor.org/rfc/rfc8259.html) |
| `csv`     |  yes | [CSV RFC 4180](https://www.rfc-editor.org/rfc/rfc4180.html) |
| `parquet` |  no  | [Apache Parquet](https://github.com/apache/parquet-format) |
| `zson`    |  yes | [ZSON - Human-readable Format](../formats/zson.md) |
| `zng`     |  yes | [ZNG - Binary Row Format](../formats/zson.md) |
| `zst`     |  no  | [ZST - Binary Columnar Format](../formats/zst.md) |
| `zjson`   |  yes | [ZJSON - Zed over JSON](../formats/zjson.md) |
| `zeek`    |  yes | [Zeek Logs](https://docs.zeek.org/en/master/logs/index.html) |

The input format is typically detected automatically and the formats for which
`Auto` is `yes` in the table above support _auto detection_.
Formats without auto detection require the `-i` option.

### 2.1 Hard-wired Input Format

The input format is specified with the `-i` flag.

When `-i` is specified, all of the inputs on the command-line must be
in the indicated format.

### 2.2 Auto-detection

When using _auto detection_, each input's format is independently determined
so it is possible to easily blend different input formats into a unified
output format.

For example, suppose this content is in a file `sample.csv`:
```mdtest-input sample.csv
a,b
1,foo
2,bar
```
and this content is in `sample.json`
```mdtest-input sample.json
{"a":3,"b":"baz"}
```
then the command
```mdtest-command
zq -z sample.csv sample.json
```
would produce this output in the default ZSON format
```mdtest-output
{a:1.,b:"foo"}
{a:2.,b:"bar"}
{a:3,b:"baz"}
```

### 2.3 ZSON-JSON Auto-detection

Since ZSON is a superset of JSON, `zq` must be careful in whether it
interprets input as ZSON as JSON.  While you can always clarify your intent
with the `-i zson` or `-i json`, `zq` attempts to "just do the right thing"
when you run it with JSON vs. ZSON.

While `zq` can parse any JSON using its built-in ZSON parser this is typically
not desirable because (1) the ZSON parser is not particularly performant and
(2) all JSON numbers are floating point but the ZSON parser will parse as
JSON any number that appears as an integer as an integer type.

> The reason `zq` is not particularly performant for ZSON is that the ZNG or
> ZST formats are semantically equivalent to ZSON but much more efficient and
> the design intent is that these efficient binary formats should be used in
> use cases where performance matters.  ZSON is typically used only when
> data needs to be human-readable in interactive settings or in automated tests.

To this end, `zq` uses a heuristic to select between ZSON in JSON when the
`-i` option is not specified. Specifically, JSON is selected when the first values
of the input are parsable as valid JSON and includes a JSON object either
as an outer object or as a value nested somewhere within a JSON array.

This heuristic almost always works in practice because ZSON records
typically omit quotes around field names.

## 3. Output Formats

The output format defaults to either ZSON or ZNG and may be specified
with the `-f` option.  The supported output formats include all of
the input formats along with text and table formats, which are useful
for displaying data.  (They do not capture all the information required
to reconstruct the original data so they are not supported input formats.)

Since ZSON is a common format choice, the `-z` flag is a shortcut for
`-f zson.`  Also, `-Z` is a shortcut for `-f zson` with `-pretty 4` as
described below.

### 3.1 Output Format Selection

When the format is not specified with `-f`, it defaults to ZSON if the output
is a terminal and to ZNG otherwise.

While this can cause an occasional surprise (e.g., forgetting `-f` or `-z`
in a scripted test that works fine on the command line but fails in CI),
we felt that the design of having a uniform default had worse consequences:
* If the default format were ZSON, it would be very easy to create pipelines
and deploy to production systems that were accidentally using ZSON instead of
the much more efficient ZNG format because the `-f zng` had been mistakenly
omitted from some command.  The beauty of Zed is that all of this "just works"
but it would otherwise perform poorly.
* If the default format were ZNG, then users would be endlessly annoyed by
binary output to their terminal when forgetting to type `-f zson`.

In practice, we have found that the output defaults
"just do the right thing" almost all of the time.

### 3.2 ZSON Pretty Printing

ZSON text may be "pretty printed" with the `-pretty` option, which takes
the number of spaces to use for indentation.  As this is a common option,
the `-Z` option is a shortcut for `-f zson -pretty 4`.

For example,
```mdtest-command
echo '{a:{b:1,c:[1,2]},d:"foo"}' | zq -Z -
```
produces
```mdtest-output
{
    a: {
        b: 1,
        c: [
            1,
            2
        ]
    },
    d: "foo"
}
```
and
```mdtest-command
echo '{a:{b:1,c:[1,2]},d:"foo"}' | zq -f zson -pretty 2 -
```
produces
```mdtest-output
{
  a: {
    b: 1,
    c: [
      1,
      2
    ]
  },
  d: "foo"
}
```

When pretty printing, colorization is enabled by default when writing to a terminal,
and can be disabled with `-color false`.

### 3.3 Pipeline-friendly ZNG

Though it's a compressed binary format, ZNG data is self-describing and stream-oriented
and thus is pipeline friendly.

Since data is self-describing you can simply take ZNG output
of one command and pipe it to the input of another.  It doesn't matter if the value
sequence is scalars, complex types, or records.  There is no need to declare
or register schemas or "protos" with the downstream entities.

In particular, ZNG data can simply be concatenated together, e.g.,
```mdtest-command
zq -f zng 'yield 1,[1,2,3]' > a.zng
zq -f zng 'yield {s:"hello"},{s:"world"}' > b.zng
cat a.zng b.zng | zq -z -
```
produces
```mdtest-output
1
[1,2,3]
{s:"hello"}
{s:"world"}
```
And while this ZSON output is human readable, the ZNG files are binary, e.g.,
```mdtest-command
zq -f zng 'yield 1,[1,2,3]' > a.zng
hexdump -C a.zng
```
produces
```mdtest-output
00000000  02 00 01 09 1b 00 09 02  02 1e 07 02 02 02 04 02  |................|
00000010  06 ff                                             |..|
00000012
```

### 3.4 Schema-rigid Outputs

Certain data formats like Parquet are "schema rigid" in the sense that they
require a schema to be defined before values can be written into the file and
all the values in the file must conform to this schema.

Zed, however, has a fine-grained type system instead of schemas and a sequence
of data values are completely self-describing and may be heterogeneous in nature.
This creates a challenge converting the type-flexible Zed formats to a schema-rigid
format like Parquet.

For example, this seemingly simple conversion:
```mdtest-command fails
echo '{x:1}{s:"hello"}' | zq -o out.parquet -f parquet -
```
causes this error
```mdtest-output
Parquet output requires uniform records but multiple types encountered (consider 'fuse')
```

#### 3.4.1 Fusing Schemas

As suggested by the error above, the Zed `fuse` operator can merge different record
types into a blended type, e.g., here we create the file and read it back:
```mdtest-command
echo '{x:1}{s:"hello"}' | zq -o out.parquet -f parquet fuse -
zq -z -i parquet out.parquet
```
but the data was necessarily changed (by inserting nulls):
```mdtest-output
{x:1,s:null(string)}
{x:null(int64),s:"hello"}
```

#### 3.4.2 Splitting Schemas

Another common approach to dealing with the schema-rigid limitation of Parquet
is to create a separate file for each schema.

`zq` can do this too with the `-split` option, which specifies a path
to a directory for the output files.  If the path is `.`, then files
are written to the current directory.

The files are named using the `-o` option as a prefix and the suffix is
`-<n>.<ext>` where the `<ext>` is determined from the output format and
where `<n>` is a unique integer for each distinct output file.

For example, the example above would produce two output files,
which can then be read separately to reproduce the original data, e.g.,
```mdtest-command
echo '{x:1}{s:"hello"}' | zq -o out -split . -f parquet -
zq -z -i parquet out-*.parquet
```
produces the original data
```mdtest-output
{x:1}
{s:"hello"}
```

While the `-split` option is most useful for schema-rigid formats, it can
be used with any output format.

## 4. Query Debugging

If you are ever stumped about how the `zq` compiler is parsing your query,
you can always run `zq -C` to compile and display your query in canonical form
without running it.
This can be especially handy when you are learning the language and
[its shortcuts](language.md#26-implied-operators).

For example, this query
```mdtest-command
zq -C 'has(foo)'
```
is an implied [where operator](operators/where.md), which matches values
that have a field `foo`, i.e.,
```mdtest-output
where has(foo)
```
while this query
```mdtest-command
zq -C 'lower(foo)'
```
is an implied [yield operator](operators/yield.md), which produces the lower case
version of the presumed string in field `foo`, i.e.,
```mdtest-output
yield lower(foo)
```

## 5. Error Handling

Fatal errors like "file not found" or "file system full" are reported
as soon as they happen and cause the `zq` process to exit.

On the other hand,
runtime errors resulting from the Zed query itself
do not halt execution.  Instead, these error conditions produce
[first-class Zed errors](language.md#53-first-class-errors)
in the data output stream interleaved with any valid results.
Such errors are easily queried with the
[is_error function](functions/is_error.md).

This approach provides a robust technique for debugging complex query pipelines,
where errors can be wrapped in one another providing stack-trace-like debugging
output alongside the output data.  This approach has emerged as a more powerful
alternative to the traditional technique of looking through logs for errors
or trying to debug a halted program with a vague error message.

For example, this query
```
echo '1 2 0 3' |  zq '10.0/this' -
```
produces
```
10.
5.
error("divide by zero")
3.3333333333333335
```
and
```
echo '1 2 0 3' |  zq '10.0/this' - | zq 'is_error(this)' -
```
produces just
```
error("divide by zero")
```

## 6. Examples

As you may have noticed, many examples of the [Zed language](language.md)
are illustrated using this pattern
```
echo <values> | zq <query> -
```
which is used throughout the [language documentation](language.md)
and [operator reference](reference.md).

The language documentation and [tutorials directory](../tutorials)
have many examples, but here are a few more simple `zq` use cases.

_Hello, world_
```
echo '"hello, world"' | zq -z 'yield this' -
```
produces this ZSON output
```
"hello, world"
```

_Some values of available data types_
```
echo '1 1.5 [1,"foo"] |["apple","banana"]|' | zq -z 'yield this' -
```
produces
```
1
1.5
[1,"foo"]
|["apple","banana"]|
```
_The types of various data_
```
echo '1 1.5 [1,"foo"] |["apple","banana"]|' | zq -z 'yield typeof(this)' -
```
produces
```
<int64>
<float64>
<[(int64,string)]>
<|[string]|>
```
_A simple aggregation_
```
echo '{key:"foo",val:1}{key:"bar",val:2}{key:"foo",val:3}' | zq -z 'sum(val) by key | sort key' -
```
produces
```
{key:"bar",sum:2}
{key:"foo",sum:4}
```
_Convert CSV to Zed and cast a to an integer from default float_
```
printf "a,b\n1,foo\n2,bar\n" | zq 'a:=int64(a)' -
```
produces
```
{a:1,b:"foo"}
{a:2,b:"bar"}
```
_Convert JSON to Zed and cast to an integer from default float_
```
echo '{"a":1,"b":"foo"}{"a":2,"b":"bar"}' | zq 'a:=int64(a)' -
```
produces
```
{a:1,b:"foo"}
{a:2,b:"bar"}
```
_Make a schema-rigid Parquet file using fuse and turn it back into Zed_
```
echo '{a:1}{a:2}{b:3}' | zq -f parquet -o tmp.parquet fuse -
zq -z -i parquet tmp.parquet
```
produces
```
{a:1,b:null(int64)}
{a:2,b:null(int64)}
{a:null(int64),b:3}
```

## 7. Performance

Your mileage may vary, but many new users of `zq` are surprised by its speed
compared to tools like `jq`, `grep`, `awk`, or `sqlite` especially when running
`zq` over files in the ZNG format.

### 7.1 Fast Pattern Matching

One important technique that helps `zq` run fast is to take advantage of queries
that involve fine-grained searches.

When a query begins with a logical expression containing either a search
or a predicate match with a constant value, and presuming the input data format
is ZNG, then the runtime optimizes the query by performing an efficient,
byte-oriented "pre-search" of the values required in the predicate.  This pre-search
scans the bytes that comprise a large buffer of values and looks for these values
and, if they are not present, the entire buffer is discarded knowing no individual
value in that buffer could match because the required serialized
values were not present in the buffer.

For example, if the Zed query is
```
"http error" and ipsrc==10.0.0.1 | count()
```
then the pre-search would look for the string "http error" and the Zed encoding
of the IP address 10.0.0.1 and unless both those values are present, then the
buffer is discarded.

Moreover, ZNG data is compressed and arranged into frames that can be decompressed
and processed in parallel.  This allows the decompression and pre-search to
run in parallel very efficiently across a large number of threads.  When searching
for sparse results, many frames are discarded without their uncompressed bytes
having to be processed any further.

While this pre-search technique results in very fast brute-force pattern matching,
[search indexes](../zed/README.md#search-indexes)
can also be created when Zed data is managed by a Zed lake
thereby avoiding scans of data altogether as the index pinpoints the locations
of specific values in the lake.

### 7.2 Efficient JSON Processing

While processing data in the ZNG format is far more efficient than JSON,
there is substantial JSON data in the world and it is important for JSON
input to perform well.

This proved a challenge as `zq` is written in Go and Go's JSON package
is not particularly performant.  To this end, `zq` has its own lean and simple
[JSON tokenizer](https://pkg.go.dev/github.com/brimdata/zed/pkg/jsonlexer),
which performs quite well,
and is
[integrated tightly](https://github.com/brimdata/zed/blob/main/zio/jsonio/reader.go)
with Zed's internal data representation.
Moreover, like `jq`,
`zq`'s JSON parser does not require objects to be newline delimited and can
incrementally parse the input to minimize memory overhead and improve
processor cache performance.

The net effect is a JSON parser that is typically a bit faster than the
native C implementation in `jq`.

### 7.3 Performance Comparisons

To provide a rough sense of the performance tradeoffs between `zq` and
other tooling, this section provides results of a few simple speed tests.

#### 7.3.1 Test Data

These tests are easy to reproduce.  The input data comes from the
[Zed sample data repository](https://github.com/brimdata/zed-sample-data),
where we used a semi-structured Zeek "conn" log from the `zeek-default` directory.

It is easy to convert the Zeek logs to a local ZNG file using
zq's built-in `get` operator:
```
zq -o conn.zng 'get https://raw.githubusercontent.com/brimdata/zed-sample-data/main/zeek-default/conn.log.gz'
```
This creates a new file `conn.zng` from the Zeek log file fetched from GitHub.

Note that this data is a gzip'd file in the Zeek format and `zq`'s auto-detector
figures out both that it is gzip'd and that the uncompressed format is Zeek.
There's no need to specify flags for this.

Next, a JSON file can be converted from ZNG using:
```
zq -f json conn.zng > conn.json
```
Note here that we lose information in this conversion because the rich data types
of Zed (that were [translated from the Zeek format](../../zeek/Data-Type-Compatibility.md)) are lost.

We'll also make a SQLite database in the file `conn.db` as the table named `conn`.
One easy way to do this is to install
[sqlite-utils](https://sqlite-utils.datasette.io/en/stable/)
and run
```
sqlite-utils insert conn.db conn conn.json --nl
```
(If you need a cup of coffee, a good time to get it would be when
loading the JSON into SQLite.)

#### 7.3.2 File Sizes

Note the resulting file sizes:
```
% du -h conn.json conn.db conn.zng
416M	conn.json
192M	conn.db
 38M	conn.zng
```
Much of the performance of ZNG derives from an efficient, parallelizable
structure where frames of data are compressed
(currently with [LZ4](http://lz4.github.io/lz4/) though the
specification supports multiple algorithms) and the sequence of values
can be processed with only partial deserialization.

That said, there are quite a few more opportunities to further improve
the performance of `zq` and the Zed system and we have a number of projects
forthcoming on this front.

#### 7.3.3 Tests

We ran three styles of tests on a Mac quad-core 2.3GHz i7:
* `count` - compute the number of values present
* `search` - find a value in a field
* `agg` - sum a field grouped by another field

Each test was run for `jq`, `zq` on JSON, `sqlite3`, and `zq` on ZNG.

We used the Bash `time` command to measure elapsed time.

The command lines for the `count` test were:
```
jq -s length conn.json
sqlite3 conn.db 'select count(*) from conn'
zq 'count()' conn.zng
zq 'count()' conn.json
```
The command lines for the `search` test were:
```
jq 'select(.id.orig_h=="10.47.23.5")' conn.json
sqlite3 conn.db 'select * from conn where json_extract(id, "$.orig_h")=="10.47.23.5"'
zq 'id.orig_h==10.47.23.5' conn.zng
zq 'id.orig_h==10.47.23.5' conn.json
```
Here, we look for an IP address (10.47.23.5) in a specific
field `id.orig_h` in the semi-structured data.  Note when using ZNG,
the IP is a native type whereas for `jq` and SQLite it is a string.
Note that `sqlite` must use its `json_extract` function since nested JSON objects
are stored as minified JSON text.

The command lines for the `agg` test were:
```
jq -n -f agg.jq conn.json
sqlite3 conn.db 'select sum(orig_bytes),json_extract(id, "$.orig_h") as orig_h from conn group by orig_h'
zq "sum(orig_bytes) by id.orig_h" conn.zng
zq "sum(orig_bytes) by id.orig_h" conn.json
```
where the `agg.jq` script is:
```
def adder(stream):
  reduce stream as $s ({}; .[$s.key] += $s.val);
adder(inputs | {key:.id.orig_h,val:.orig_bytes})
| to_entries[]
| {orig_h: (.key), sum: .value}
```

#### 7.3.4 Results

The following table summarizes the results of each test as a column and
each tool as a row with the speed-up factor (relative to `jq`)
shown in parentheses:

|  | `count` | `search` | `agg` |
|------|---------------|---------------|---------------|
| `jq` | 11,540ms (1X) | 10,730ms (1X) | 20,175ms (1X) |
| `zq-json` | 7,150ms (1.6X) | 7,230ms (1.5X)  | 7,390ms (2.7X) |
| `sqlite` | 100ms (115X) | 620ms (17X) | 1,475ms (14X) |
| `zq-zng` | 110ms (105X) | 135ms (80X) | 475ms (42X) |

To summarize, `zq` with ZNG is consistently fastest though `sqlite`
was a bit faster counting rows.

In particular, `zq` is substantially faster (40-100X) than `jq` with the efficient
ZNG format but more modestly faster (50-170%) when processing the bulky JSON input.
This is expected because parsing JSON becomes the bottleneck.

While SQLite is much faster than `jq`, it is not as fast as `zq`.  The primary
reason for this is that SQLite stores its semi-structured columns as minified JSON text,
so it must scan and parse the JSON when executing the _where_ clause above
as well as the aggregated fields.

Also, note that the inferior performance of `sqlite` is in areas where databases
perform extraordinarily well if you do the work to
(1) transform semi-structured columns to relational columns by flattening
nested JSON objects (which are not indexable by `sqlite`) and
(2) configuring database indexes.

In fact, if you implement these changes, `sqlite` performs better than `zq` on these tests.

However, the benefit of Zed is that no flattening is required.  And unlike `sqlite`,
`zq` is not intended to be a database.  That said, there is no reason why database
performance techniques cannot be applied to the Zed model and this is precisely what the
open-source Zed project intends to do.  As a first step, with a
[Zed lake](../zed/README.md), you can build type-flexible
[search indexes](../zed/README.md#search-indexes)
to scale searches across very large stores of Zed data.

Stay tuned!
