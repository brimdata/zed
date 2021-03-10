# Zst - *z*ng-*st*acked format

Zst, pronounced "zest", is a format for columnar data based on
[zng](../zng/docs/spec.md).
Zst is the "stacked" version of zng, where the fields from a stream of
zng records are stacked into vectors that form columns.

You can convert a zng stream to zst and back to zng,
and the input and output will match exactly.

Zst is much inspired by [parquet](https://github.com/apache/parquet-format).
We thought long and hard about using
parquet directly with zng but felt a new format was warranted and important
(as is justified elsewhere).

> TBD: add a link to the "justified elsewhere" document.

> TBD: check that zng aliases "just work" as alias bindings should be emitted
> when any data of using a type alias is decoded and written/transmitted

> TBD: add support for multi-file zst objects.

> TBD: test support for zng set types.

> TBD: add support for more efficient seekability (compared to brute force
> parsing of all the root ids).  However we do this, it will probably be the
> right opportunity to add arbitrary zng meta data for pruning (e.g., numeric
> min,max,sum,small-set summaries,cardinality).  We also probably need to add
> column counts and byte sizes in various places to optimize things and provide
> for scanning stats.  Also, being able to do push-down with multiple, parallel
> queries could be useful when the app wants to assemble a splash page or
> run about of complex, interrelated queries to populate sophisticated views etc.

> TBD: add more examples.

## The Zng Data Model

The zng data model is an unbounded sequence of zng streams, where
each stream comprises a finite sequence of "rows", and each row conforms to
an arbitrary schema.  In this way, you can think of a zng stream as a diverse
collection of sql-like tables where the rows from each table are interspersed
amongst each other in a deterministic order.

> In the [zq implementation](https://github.com/brimsec/zq), a zng row is a
> [zng.Record](https://github.com/brimsec/zq/blob/42103ef6a15b3ee53fbcd980604e75c42ea3308d/zng/recordval.go#L39)
> and a schema is a
> [zng.TypeRecord](https://github.com/brimsec/zq/blob/42103ef6a15b3ee53fbcd980604e75c42ea3308d/zng/record.go#L10).

Each zng stream has its own, embedded "type context" that defines the schema
of each row value. A zng stream is fully self-contained and self describing.
There is no need for external schema definitions or for accesses to a
centralized schema registry to decode a zng stream.

Zng streams can represent data at rest on a storage system, data in memory
for online-analytics processing, or data in flight,
e.g., as the foundation of a data communication layer.  Like
[FlexBuffers](https://google.github.io/flatbuffers/flexbuffers.html),
the zng format was designed so that serialized form of the data is the same
as the in-memory form of the data so there is no need to unmarshal zng data
into native programming data structures.  Thus, analytics may be carried out
directly on the zng data types.  We call this "in-place analytics".

Unlike parquet, which presumes the out-of-band definition of schemas with
optional fields, the zng data model is self-describing and presumes there is
often no good way ahead of time to know which fields may or may not be optional.
Thus, the encoding here is a bit different and new, as any field in any zng row can
always be optional.  You don't need to say ahead of times what things are optional.
For many important use cases, you can't know this anyway.

> Ad an example: zeek script change that adds a new column (maybe this discussion
> belongs elsewhere).

A value that is not present in a field is called "unset".

> An "unset" field should not be confused with a value that is present
> but is null/empty/undefined/nil/etc.

## Zst

Zst (pronounced "zest") is a storage object format that represents a fixed-size
zng stream.  Its purpose is to provide for efficient analytics and search over
snapshots of zng row data that is stored in columnar form.

While a zst object can be built on the fly from a stream of zng records,
it generally cannot be read or queried until the final end-of-stream
is received on the stream of zng records that are to be treated as a
zst object and everything about the zst object is flushed to storage.

A zst object can be stored entirely as one seekable object (e.g., an s3 object)
or split into separate seekable objects (e.g., unix files) that are treated
together as a single zst entity.  While the zst format provides much flexibility
for how data is laid out, it's up to implementations to layout data
in intelligent ways for efficient sequential read access in spite of occasional
seeks.

## The "Column Stream" Abstraction

The zst data abstraction is built around a collection of "column streams".

There is one column stream for each field of a zng record type definition
that appears collectively throughout the zng input data including records
embedded within records.

For example, a field that is an array of records containing other fields would
have a column stream for the top-level array field as well as a column
for each field inside of the array values.

Each column stream represents a sequence of values (inclusive of unset values)
comprising each occurrence where that value appears with respect to its context.

Records are reconstructed one by one from the column streams by picking values
from each appropriate column stream based on the type structure of the record and
its relationship to the various column streams.  For hierarchical records
(i.e., records inside of records, or records inside of arrays inside of records, etc),
the reconstruction process is recursive (as described below).

## The Physical Layout

Given the above logical design of zst column streams, we now describe how
zng rows are physically laid out across zst columns as a storage
data structure.

The overall layout of a zst object is comprised of the following sections,
in this order:
* the data section,
* the reassembly section,
* and the trailer.

This layout is designed so that an implementation can buffer metadata in
memory while writing column data in a natural order to the
data section (based on the volume statistics of each column),
then write the metadata into the reassembly section along with the trailer
at the end.  This allows a zng stream to be converted to a zst object
in a single pass.

> That said, the layout is
> flexible enough that an implementation may optimize the data layout with
> additional passes or by writing the output to multiple files then then
> merging them together (or even leaving the zst object as separate files
> that can be collectively read and scanned by a zst implementation).

### The Data Section

The data section contains raw data values organized into "segments",
where a segment is simply a seek offset and byte length relative to the
data section.  Each segment contains a sequence of
[primitive-type zng values](../zng/docs/spec.md#5-primitive-types),
encoded as counted-length byte sequences where the counted-length is
variable-length encoded as in the zng spec.

There is no information in the data section for how segments relate
to one another or how they are reconstructed into columns.  They are just
blobs of zng data.

> Unlike parquet, there is no explicit arrangement the column chunks into
> row groups but rather they are allowed to grow at different rates so a
> high-volume column might be comprised of many segments while a low-volume
> column must just be one or several.  This allows scans of low-volume record types
> (the "mice") to perform well amongst high-volume record types (the "elephants"),
> i.e., there are not a bunch of seeks with tiny reads of mice data interspersed
> throughout the elephants.

> TBD: The mice/elephants model creates an interesting and challenging layout
> problem.  If you let the row indexes get too far apart (call this "skew"), then
> you have to buffer very large amounts of data to keep the column data aligned.
> This is the point of row groups in parquet, but the model here is to leave it
> up to the implementation to do layout as it sees fit.  You can also fall back
> to doing lots of seeks and that might work perfectly fine when using SSDs but
> this also creates interesting optimization problems when sequential reads work
> a lot better.  There could be a scan optimizer that lays out how the data is
> read that lives under the column stream reader.  Also, you can make tradeoffs:
> if you use lots of buffering on ingest, you can write the mice in front of the
> elephants so the read path requires less buffering to align columns.  Or you can
> do two passes where you store segments in separate files them merge them at close
> according to an optimization plan.

Segments are sub-divided into frames where each frame is compressed
independently of each other, similar to zng compression framing.

> TBD: use the
> [same compression format](../zng/docs/spec.md#312-compressed-value-message-block)
> exactly?

> The intent here is that segments are sized so that sequential read access
> performs well (e.g., 5MB) while frames are comparatively smaller (say 32KB)
> so that they can be decompressed and processed in a multi-threaded fashion where
> search and analytics can be performed on the decompressed buffer by the same
> thread that decompressed the frame enhancing read-locality and L1/L2 cache
> performance.

### The Reassembly Section

The reassembly section provides the information needed to reconstruct
column streams from segments, and in turn, to reconstruct the original zng rows
from column streams, i.e., to map columns back to rows.

> Of course, the reassembly section also provides the ability to extract just subsets of columns
> to be read and searched efficiently without ever needing to reconstruct
> the original rows.  How performant this is all done is up to any particular
> zst implementation.

> Also, the reassembly section is in generally vastly smaller than the data section
> so the goal here isn't to express information in cute and obscure compact forms
> but rather to represent data in easy-to-digest, programmer-friendly form that
> leverages zng.

The reassembly section is a zng stream.  Unlike parquet,
which uses an externally described schema
(via [Thrift](https://thrift.apache.org/)) to describe
analogous data structures, we simply reuse zng here.

> So, we are using zng to encode zng in column format.  Yo dawg.

#### The Schema Definitions

This reassembly zng stream encodes 2*N+1 zng records, where N is equal to the number
of top-level zng record types that are present in the encoded input.
To simplify terminology, we call a top-level zng record type a "schema",
e.g., there are N unique schemas encoded in the zst object.

Each of these N schemas gets defined as a record value, comprised
of unset fields appearing in the same order
as they were encountered in the original zng stream.
In this way, a fresh zng type context can be created to read these N records and
that type context will have precisely the same structure as the type context
from the original zng stream.  This is guaranteed by the deterministic nature
of embedded zng type definitions.

The next N+1 records contain reassembly information for each of the N schemas
where each record is used to create column streams to reconstruct the original
zng records.  The schemas do not overlap as columns from a record from any one
schema are not intermixed with columns of another.

#### Segment Maps

The foundation of column reconstruction is based on segment maps, which is simply
a list of the segments from the data area that are concatenated to form the
data for a column stream.

Each segment map that appears within the reassembly records is represented
with a zng array of records that represent seek ranges, i.e., specifically
this zng type:
```
array[record[offset:uint64,length:uint32]]
```
In the rest this document, we will refer to this type as `<segmap>` for
shorthand and refer to the concept as a "segmap".

> We use the type name "segmap" to emphasize that this information represents
> a set of seek ranges where data stored and must be read from *rather than*
> the data itself.

#### The Root Reassembly Record

The first of the N+1 reassembly records defines the "root column", where this column
represents the sequence of schemas of each original zng record, i.e., indicating
which schema's column stream to select from to pull column values to form the zng row.
The sequence of schemas is defined by each row's small-integer positional index,
0 to N-1, within the N schemas.

The root column stream is encoded as zng int32 values.
While there are a large number entries in the root column (one for each original row),
the cardinality of the set of root identifiers is small in practice so this column
will compress very significantly, e.g., in the special case that all the
rows have the same schema, this column will compress to practically nothing.

The type of the reassembly record for the root column stream is simply:
```
record[root:<segmap>]
```

#### The Row Reassembly Records

The remaining N records in the reassembly stream define the reassembly
maps for each schema.

Each such row reassembly record is record of type `<record_column>`, as defined below,
where each row assembly record appears in the same sequence as the original N schemas.
This simple top-level arrangement, along with the definition of
the `<record_column>` type below, is all that is needed to reconstruct all of the
original data.

In other words, these root reassembly record combined with the N row reassembly
records collectively define the original zng row structure.
Taken in pieces, the reassembly records allow efficient access to sub-ranges of the
rows, to subset of columns of the rows, to sub-ranges of columns of the rows, and so forth.

> Note that each row reassembly record has its own layout of columnar
> values and there is no attempt made to store like-typed columns from different
> schemas in the same physical column.

A `<record_column>` is defined recusively in terms of other `<record_column>'s`
and other type that represent arrays, unions, or primitive types, which will
be referred to as follows:
* `<array_column>`,
* `<union_column>`, or
* `<primitive_column>`.

In addition,the notation `<any_column>` refers to any instance of the four
column types:
* `<record_column>`,
* `<array_column>`,
* `<union_column>`, or
* `<primitive_column>`.

#### Record Column

The `<record_column>` type has the form:
```
record[
        <fld1>:record[column:<any_column>,presence:<segmap>],
        <fld2>:record[column:<any_column>,presence:<segmap>],
        ...
        <fldn>:record[column:<any_column>,presence:<segmap>]
]        
```
where
* `<fld1>` through `<fldn>` are the names of the top-level fields of the
original row record,
* the `column` fields are column stream definitions for each field, and
* the `presence` columns are int32 zng column streams comprised of a
run-length encoding the locations of column values in their respective rows,
when there are unset values (as described below).

If there are no unset values, then the `presence` field contains an empty `<segmap>`.
If all of the values are unset, then then `column` field is unset (and the `presence`
both contain empty `<segmap>'s`).  For empty `<segmap>'s`, there is no
corresponding data stored in the data section.  Since a `<segmap>` is a zng
array, an empty `<segmap>` is simply the empty array value `[]`.

> Note that the only place a value can be "unset" is in the context of
> a field of a record.  Hence, the only place presence columns are needed
> is on the context of a record field.

#### Array Column

An `<array_column>` has the form:
```
record[values:<any_column>,lengths:<segmap>]
```
where
* `values` represents a continuous sequence of values of the array elements  
that are sliced into array values based on the length information, and
* `lengths` encodes a zng int32 sequence of values that represent the length
 of each array value.

#### Union Columns

A `<union_column>` has the form:
```
record[c0:<any_column>,c1:<any_column>,...,selector:<segmap>]
```
where
* `c0`, `c1` etc, up to the number of types in the union, are column values for
each the type ordered by the implied type in the union type, and
* `selector` is a column of int32 where each subsequent value indicates which
of the union types is to be used for each respective column.

The number of times each value of `selector` appears must equal the number of values
in each respective column.

#### Primitive Column

A `<primitive_column>` is just a `<segmap>` that defines a column stream of
primitive values.  There is no need to encode the primitive type here
as it can be obtained from the corresponding schema.

> Note that when locating a column, all type information is known
> from the root type context of the record in question so there is no need
> to encode the type information redundantly here.  An implementation would
> typically pass down the appropriate types from the root type context recursively
> when landing at a primitive_column type encoded in the local type context.

#### Presence Columns

The presence column is logically a sequence of booleans, one for each position
in the original column, indicating whether a value is unset or present.
The number of values in the encoded column is equal to the number of values
present so that unset values are not encoded.

Instead the presence column is encoded as a sequence of alternating runs.
First, the number of values present is encoded, then the number of values not present,
then the number of values present, and so forth.   These runs are the stored
as zng int32 values in the presence column (which may be subject to further
compression based on segment framing).

### The Trailer

After the reassembly section is a zng stream with a single record defining
the "trailer" of the zst object.  The trailer provides a magic field
indicating the "zst" format, a version number,
the size of the segment threshold for decomposing segments into frames,
the size of the skew threshold for flushing all segments to storage when
the memory footprint roughly exceeds this threshold,
and an array of sizes in bytes of the sections of the zst object.

This type of this record has the format
```
record[magic:string,version:int32,skew_thresh:int32,segment_thresh:int32,sections:array[int64]]
```
The trailer can be efficiently found by scanning backward from the end of the
zst object to the find a valid zng stream containing a single record value
conforming to the above type.

## Decoding

To decode a entire zst object into rows, the trailer is read to find the sizes
of the sections, then the zng stream of the reassembly section is read,
typically in its entirety.

Since this data structure is relatively small compared to all of the columnar
data in the zst object,
it will typically fit comfortably in memory and it can be very fast to scan the
entire reassembly structure for any purpose.

> For example, for a given query, a "scan planner" could traverse all the
> reassembly records to figure out which segments will be needed, then construct
> an intelligent plan for reading the needed segments and attempt to read them
> in mostly sequential order, which could serve as
> an optimizing intermediary between any underlying storage API and the
> zst decoding logic.

To decode the "next" row, its schema index is read from the root reassembly
column stream.

This schema index then determines which reassembly record to fetch
column values from.

The top-level reassembly fetches column values as a `<record_column>`.

For any `<record_column>`, a value from each field is read from each field's column,
accounting for the presence column indicating nil,
and the results are encoded into the corresponding zng record value using
zng type information from the corresponding schema.

For a `<primitive_column>` a value is determined by reading the next
value from its segmap.

For an `<array_column>`, a length is read from its `lengths` segmap as an int32
and that many values are read from its the `values` sub-column,
encoding the result as a zng array value.

For an `<union_column>`, a value is read from its `selector` segmap
and that value is used to the select to corresponding column stream
`c0`, `c1`, etc.  The value read is then encoded as a zng union value
using the same selector value within the union value.

## Examples

### Hello, world

Start with this zng data (shown as human-readable [ZSON](../zng/docs/zson.md)):
```
{a:"hello",b:"world"}
{a:"goodnight",b:"gracie"}
```

To convert to zst format:
```
zq -f zst hello.zson > hello.zst
```

Segments in the zst format would be laid out like this:
```
=== column for a
hello
goodnight
=== column for b
world
gracie
=== column for schema IDs
0
0
===
```

To see the detailed zst structure described as ZSON, you can use the `zst`
command like this:
```
zst inspect -Z -trailer hello.zst
```

which provides the output (comments added with explanations):

```
{
    a: null (string),                // First, the schemas are defined (just one here).
    b: null (string)
}
{
    root: [                          // Then, the root reassembly record.
        {
            offset: 29,
            length: 2 (int32)
        } (=0)
    ] (=1)
} (=2)
{
    a: {                             // Next comes the column assembly records.
        column: [                    // (Again, only one schema in this example, so only one such record.)
            {
                offset: 0,
                length: 16
            }
        ] (1),
        presence: [] (1)
    } (=3),
    b: {
        column: [
            {
                offset: 16,
                length: 13
            }
        ],
        presence: []
    } (3)
} (=4)
{
    magic: "zst",                    // Finally, the trailer as a new zng stream.
    version: 1 (int32),
    skew_thresh: 26214400 (int32),
    segment_thresh: 5242880 (int32),
    sections: [
        31,
        94
    ]
} (=5)
```

> Note finally, if there were 10MB of zng row data here, the reassembly section
> would be basically the same size, with perhaps a few segmaps.  This emphasizes
> just how small this data structure is compared to the data section.
