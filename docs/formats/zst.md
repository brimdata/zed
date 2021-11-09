# ZST - ZNG stacked format

ZST, pronounced "zest", is an object storage format for columnar data based on
[the Zed data model](zdm.md).
ZST is the "stacked" version of Zed, where the fields from a stream of
Zed records are stacked into vectors that form columns.
Its purpose is to provide for efficient analytics and search over
bounded-length sequences of ZNG data that is stored in columnar form.

Like [Parquet](https://github.com/apache/parquet-format),
ZST provides an efficient columnar representation for semi-structured data,
but unlike Parquet, ZST is not based on schemas and does not require
a schema to be declared when writing data to an object.  Instead,
ZST exploits the superstructured nature of Zed data: columns of data
self-organize around their type structure.

## ZST Objects

A ZST object encodes a bounded, ordered sequence of Zed values.
To provide for efficient access to subsets of ZST-encoded data (e.g., columns),
the ZST object is presumed to be accessible via random access
(e.g., range requests to a cloud object store or seeks in a Unix file system)
and ZST is therefore not intended as a streaming or communication format.

A ZST object can be stored entirely as one storage object
or split across separate objects that are treated
together as a single ZST entity.  While the ZST format provides much flexibility
for how data is laid out, it is left to an implementation to layout data
in intelligent ways for efficient sequential read accesses of related data.

## Column Streams

The ZST data abstraction is built around a collection of _column streams_.

There is one column stream for each type encountered in the input where
each column stream is encoded according to its type.  For example,
a record column encodes a "presence" vector encoding any null value for
each field then encodes each non-null field recursively, whereas
an array column encodes a "lengths" vector and encodes each
element recursively.

Values are reconstructed one by one from the column streams by picking values
from each appropriate column stream based on the type structure of the value and
its relationship to the various column streams.  For hierarchical records
(i.e., records inside of records, or records inside of arrays inside of records, etc),
the reconstruction process is recursive (as described below).

## The Physical Layout

The overall layout of a ZST object is comprised of the following sections,
in this order:
* the data section,
* the reassembly section,
* and the trailer.

This layout allows an implementation to buffer metadata in
memory while writing column data in a natural order to the
data section (based on the volume statistics of each column),
then write the metadata into the reassembly section along with the trailer
at the end.  This allows a ZNG stream to be converted to a ZST object
in a single pass.

> That said, the layout is
> flexible enough that an implementation may optimize the data layout with
> additional passes or by writing the output to multiple files then then
> merging them together (or even leaving the ZST object as separate files).

### The Data Section

The data section contains raw data values organized into _segments_,
where a segment is a seek offset and byte length relative to the
data section.  Each segment contains a sequence of
[primitive-type Zed values](zdm.md#5-1-primitive-types),
encoded as counted-length byte sequences where the counted-length is
variable-length encoded as in the ZNG spec.

There is no information in the data section for how segments relate
to one another or how they are reconstructed into columns.  They are just
blobs of ZNG data.

> Unlike Parquet, there is no explicit arrangement the column chunks into
> row groups but rather they are allowed to grow at different rates so a
> high-volume column might be comprised of many segments while a low-volume
> column must just be one or several.  This allows scans of low-volume record types
> (the "mice") to perform well amongst high-volume record types (the "elephants"),
> i.e., there are not a bunch of seeks with tiny reads of mice data interspersed
> throughout the elephants.

> TBD: The mice/elephants model creates an interesting and challenging layout
> problem.  If you let the row indexes get too far apart (call this "skew"), then
> you have to buffer very large amounts of data to keep the column data aligned.
> This is the point of row groups in Parquet, but the model here is to leave it
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
independently of each other, similar to ZNG compression framing.

> TBD: use the
> [same compression format](zng.md#312-compressed-value-message-block)
> exactly?

> The intent here is that segments are sized so that sequential read access
> performs well (e.g., 5MB) while frames are comparatively smaller (say 32KB)
> so that they can be decompressed and processed in a multi-threaded fashion where
> search and analytics can be performed on the decompressed buffer by the same
> thread that decompressed the frame enhancing read-locality and L1/L2 cache
> performance.

### The Reassembly Section

The reassembly section provides the information needed to reconstruct
column streams from segments, and in turn, to reconstruct the original Zed values
from column streams, i.e., to map columns back to composite values.

> Of course, the reassembly section also provides the ability to extract just subsets of columns
> to be read and searched efficiently without ever needing to reconstruct
> the original rows.  How performant this is all done is up to any particular
> ZST implementation.

> Also, the reassembly section is in generally vastly smaller than the data section
> so the goal here isn't to express information in cute and obscure compact forms
> but rather to represent data in easy-to-digest, programmer-friendly form that
> leverages ZNG.

The reassembly section is a ZNG stream.  Unlike Parquet,
which uses an externally described schema
(via [Thrift](https://thrift.apache.org/)) to describe
analogous data structures, we simply reuse ZNG here.

#### The Super Types

This reassembly stream encodes 2*N+1 Zed values, where N is equal to the number
of top-level Zed types that are present in the encoded input.
To simplify terminology, we call a top-level Zed type a "super type",
e.g., there are N unique super types encoded in the ZST object.

These N super types are defined by the first N values of the reassembly stream
and are encoded as a null value of the indicated super type.
A super type's integer position in this sequence defines its identifier
encoded in the super column (defined below).  This identifier is called
the super ID.

> Change the first N values to type values instead of nulls?

The next N+1 records contain reassembly information for each of the N super types
where each record defines the column streams needed to reconstruct the original
Zed values.

#### Segment Maps

The foundation of column reconstruction is based on _segment maps_.
A segment map is a list of the segments from the data area that are
concatenated to form the data for a column stream.

Each segment map that appears within the reassembly records is represented
with a Zed array of records that represent seek ranges conforming to this
type signature:
```
[{offset:uint64,length:uint32}]
```
In the rest this document, we will refer to this type as `<segmap>` for
shorthand and refer to the concept as a "segmap".

> We use the type name "segmap" to emphasize that this information represents
> a set of byte ranges where data stored and must be read from *rather than*
> the data itself.

#### The Super Column

The first of the N+1 reassembly records defines the "super column", where this column
represents the sequence of super types of each original Zed value, i.e., indicating
which super type's column stream to select from to pull column values to form
the reconstructed value.
The sequence of super types is defined by each type's super ID (as defined above),
0 to N-1, within set of N super types.

The super column stream is encoded as a sequence of ZNG-encoded int32 primitive values.
While there are a large number entries in the super column (one for each original row),
the cardinality of super IDs is small in practice so this column
will compress very significantly, e.g., in the special case that all the
values in the ZST object have the same super ID,
the super column will compress trivially.

The type of the reassembly record for the super column stream has the
type signature:
```
{root:<segmap>}
```

#### The Reassembly Records

The remaining N records in the reassembly stream define the reassembly
maps for each super type.

Each reassembly record is a record of type `<any_column>`, as defined below,
where each reassembly record appears in the same sequence as the original N schemas.
Note that there is no "any" type in Zed, but rather this terminology is used
here to refer to any of the concrete type structures that would appear
in a given ZST object.

In other words, the reassembly record of the super column
combined with the N reassembly records collectively define the original sequence
of Zed data values in the original order.
Taken in pieces, the reassembly records allow efficient access to sub-ranges of the
rows, to subset of columns of the rows, to sub-ranges of columns of the rows, and so forth.

This simple top-down arrangement, along with the definition of the other
column structures below, is all that is needed to reconstruct all of the
original data.

> Note that each row reassembly record has its own layout of columnar
> values and there is no attempt made to store like-typed columns from different
> schemas in the same physical column.

The notation `<any_column>` refers to any instance of the five column types:
* `<record_column>`,
* `<array_column>`,
* `<union_column>`,
* `<map_column>`, or
* `<primitive_column>`.

Note that when decoding a column, all type information is known
from the super type in question so there is no need
to encode the type information again in the reassembly record.

#### Record Column

A `<record_column>` is defined recursively in terms of the column types of
its fields, i.e., other types that represent arrays, unions, or primitive types
and has the form:
```
{
        <fld1>:{column:<any_column>,presence:<segmap>},
        <fld2>:{column:<any_column>,presence:<segmap>},
        ...
        <fldn>:{column:<any_column>,presence:<segmap>}
}        
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
corresponding data stored in the data section.  Since a `<segmap>` is a Zed
array, an empty `<segmap>` is simply the empty array value `[]`.

> Note that the only place a value can be "unset" is in the context of
> a field of a record.  Hence, the only place presence columns are needed
> is on the context of a record field.

#### Array Column

An `<array_column>` has the form:
```
{values:<any_column>,lengths:<segmap>}
```
where
* `values` represents a continuous sequence of values of the array elements  
that are sliced into array values based on the length information, and
* `lengths` encodes a Zed int32 sequence of values that represent the length
 of each array value.

The `<array_column>` structure is used for both Zed arrays and sets.

#### Union Column

A `<union_column>` has the form:
```
{columns:[<any_column>],tags:<segmap>}
```
where
* `columns` is an array containing the reassembly information for each tagged union value
in the same column order implied by the union type, and
* `tags` is a column of int32 values where each subsequent value encodes
the tag of the union type indicating which column the value falls within.

> TBD: change code to conform to columns array instead of record{c0,c1,...}

The number of times each value of `tags` appears must equal the number of values
in each respective column.

#### Map Column

A `<map_column>` has the form:
```
{key:<any_column>,value:<any_column>}
```
where
* `key` encodes the column of map keys, and
* `value` encodes the column of map values.

#### Primitive Column

A `<primitive_column>` is a `<segmap>` that defines a column stream of
primitive values.

#### Presence Columns

The presence column is logically a sequence of booleans, one for each position
in the original column, indicating whether a value is unset or present.
The number of values in the encoded column is equal to the number of values
present so that unset values are not encoded.

Instead the presence column is encoded as a sequence of alternating runs.
First, the number of values present is encoded, then the number of values not present,
then the number of values present, and so forth.   These runs are the stored
as Zed `int32` values in the presence column (which may be subject to further
compression based on segment framing).

### The Trailer

After the reassembly section is a ZNG stream with a single record defining
the "trailer" of the ZST object.  The trailer provides a magic field
indicating the "zst" format, a version number,
the size of the segment threshold for decomposing segments into frames,
the size of the skew threshold for flushing all segments to storage when
the memory footprint roughly exceeds this threshold,
and an array of sizes in bytes of the sections of the zst object.

This type of this record has the format
```
{magic:string,version:int32,skew_thresh:int32,segment_thresh:int32,sections:[int64]}
```
The trailer can be efficiently found by scanning backward from the end of the
ZST object to the find a valid ZNG stream containing a single record value
conforming to the above type.

## Decoding

To decode a entire ZST object into rows, the trailer is read to find the sizes
of the sections, then the ZNG stream of the reassembly section is read,
typically in its entirety.

Since this data structure is relatively small compared to all of the columnar
data in the ZST object,
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
and the results are encoded into the corresponding ZNG record value using
ZNG type information from the corresponding schema.

For a `<primitive_column>` a value is determined by reading the next
value from its segmap.

For an `<array_column>`, a length is read from its `lengths` segmap as an `int32`
and that many values are read from its the `values` sub-column,
encoding the result as a ZNG array value.

For an `<union_column>`, a value is read from its `selector` segmap
and that value is used to the select to corresponding column stream
`c0`, `c1`, etc.  The value read is then encoded as a ZNG union value
using the same selector value within the union value.

## Examples

### Hello, world

Start with this ZNG data (shown as human-readable [ZSON](zson.md)):
```
{a:"hello",b:"world"}
{a:"goodnight",b:"gracie"}
```

To convert to ZST format:
```
zq -f zst hello.zson > hello.zst
```

Segments in the ZST format would be laid out like this:
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

To see the detailed ZST structure described as ZSON, you can use the `zst`
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
    magic: "zst",                    // Finally, the trailer as a new ZNG stream.
    version: 1 (int32),
    skew_thresh: 26214400 (int32),
    segment_thresh: 5242880 (int32),
    sections: [
        31,
        94
    ]
} (=5)
```

> Note finally, if there were 10MB of ZNG row data here, the reassembly section
> would be basically the same size, with perhaps a few segmaps.  This emphasizes
> just how small this data structure is compared to the data section.
