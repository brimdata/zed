# Zesty: A new model for structured but heterogeneous data

> This directory contains initial and rough ideas for a columnar-oriented
> version of the zng data model.    Much of this text is currently background
> oriented and this document likely will be split into an architecture white paper
> and a columnar format specification.
> For now, while this is a work in progress, these ramblings will live here.

> There are many advantages to this
> approach of vertical integration and one could argue the time has come
> for a holistic approach like this.  We hope we're not reinventing
> the wheel, but rather we're taking the best ideas that exist today in many
> different and diverse systems and designs, and leveraging them all with
> some new stuff to create a new and better overall
> approach to data organization, archive, analytics, and search.
>

Zst, pronounced "zest", is a new approach to columnar data.
Zst is the "stacked" version of zng, where the fields from a stream of
zng records are stacked into columns.

We call this data architecture --- where we merge the zng data model with the zst
column structure -- the _zesty_ architecture.

As you know, columnar format makes for efficient analytics.  The devil's
in the details and it turns out that making columnar format work for real-world,
heterogenous data is not at all straightforward.

The zesty architecture is all about making this hard stuff easy.

## an example

Computers love columns.  Random-access memory, after all, is one big column.

But the real world progresses in rows.  It takes a bit of work to
put these rows into the one big column.
If you do the work, you can create columns to abstractly represent an organization
of the rows and make things more efficient.

For example, say some code running on a server encounters an error, then reports
that error by writing  an "event" to a log file or log service.  This error might
comprise a timestamp, an error type, a subtype, a message, and so forth.
Say another system is monitoring the performance of servers and polls
the servers regularly to get a bunch of measurements of different things, each
measurement comprising a timestamp, a metric name, a metric value, and so forth.

The industry likes to calls these logs "events" and these measurements "metrics",
where each such thing has a list of named values, and each such value conforms
to a data type.  Database people look at this and laugh and say those things
are just "rows" in a relational table.

## the relational table model

Edgar Codd knew all of this in the late 1960s.  He put down his prescient ideas
in his famous 1970 paper.  His ideas were quite theoretical but had huge
influence over the creation and evolution of the table-based database model
that survives to this day.

XXX people tried putting messy stuff in sql databases.  it didn't go well...

people also tried making streaming data base based sql and tables.  that went
better but it still had a major problem: you have to know the schema
ahead of time.

## the messy world won't go away

There is great value in assembling lots of different data sources in a central place.
This becomes the go-to place.
You can search for things, run analysis, perform security audits, etc, etc.
But you can't expect everything to conform to a magic schema that you set up
ahead of time.  That's just too hard.

## ad hoc log search

So you give up and you take all this messy data and shove it into a searchable
data store.  Then you realize you want to do other stuff with the data so you
do ad hoc analytics on search results.  It all works really well in practice,
but it's a bit messy, error-prone, and expensive.

What if we could have the best of both worlds?
Precise data modeling and semantics like sql queries but with a data
model that embraces the messy world and doesn't make you predict the future of
all possible schemas?

Moreover, search systems suck at analytics and analytics systems suck at search.
What if there were a system that was good at both?

## zesty: a holistic approach

We believe an approach that brings search and analytics together in a
unified approach without the need to define schemas ahead of time is not only
possible, but that a very promising direction is zesty.

Once you start working with zng as your underlying data layer,
you realize all sorts of things are possible.  For example, instead of
having a distributed Lucene index for search, parquet files for columnar
storage, spark jobs that convert parquet files as scala tuples for scale-out
analytics, and so forth, you can instead think about architecting a really
efficient data pipeline based on the zng data model end-to-end.

Why all these square pegs and round holes?

In the zesty approach, the data pipeline all hangs together:
* snapshots of zng row data efficiently stored
as zst columnar storage objects on s3,
* arbitrary zng data embedded as metadata inside
of zst serves as hooks to prune a scan,
* zng "microindex" lookups based on
a search term can further serve to prune a scan,
* zng-based analytics carried out efficiently on the stream of results from the
scan,
* zng-based query optimizations can be easily pushed into the zst scanner
(so called "predicate pushdown") since everything is based on zng, and
* scale-out group-by aggregations using continuous shuffles based on the
efficient transmission of zng rows from one worker to another.

We think the holistic approach will be worth the audacious effort entailed here.

Zst is a columnar representation of one zng stream.

You can convert this zng stream to zst and convert it back to zng
and the input and output will match.

build on zng rows: keep it simple.  zst just represents rows after all and
it would be nice to have tooling that can extract views of a zst file

The beauty of zng is that data is
self-describing and the query language needs to adapt to the data.  this might
sound impossible but it isn't.

strongly typed and self describing and suitable for analytics.

I'm not the Michael Jordan of data structures but I'm not dumb and this
whole thing with repetitions and definitions from Dremel just makes my head spin.
I'm all for clever and cute, but not so happy with cryptic.
Look, Dremel had pioneering ideas and has my utter and complete respect
as a novel and important research contribution, but can't we move onto
something maybe a bit easier to understand?
I would compare and contrast it to
sigma-fields probably theory...

Lucene also has really sophisticated stuff.  But come on.  It's an index.
Like at the back of a book.

## Why not parquet?

We thought long and hard about using parquet verbatim in zesty.  As we tried
to adapt it to the zng row model and the zesty approach, we encountered problems.

At first, we presumed that it would not work to map the zng row model onto
parquet row groups because zng presumes a sequence of rows from arbitrary
schemas.  But we realized we could create a column scheme that comprised the
union of the unique record types found in the zng rows, each as an "optional"
field in the top-level row, e.g., named "schema1", "schema2", etc.

While clunky, this would work.

Since zng expects that have data conforming to many different schemas all
with varying volume statistics, we wanted to separate the mice from the elephants
(for scan efficiency) while preserving the order of the zng rows (for scan correctness).
If a "mouse" represents a low-volume row type and an "elephant" represents a
very-high-volume row type, we don't want the layout of elephant columns to
make the scanning of mice columns a bunch of inefficient seeks and small reads.
XXX actually mice can be laid out in large parquet columns and read into memory
efficiently, then accessed as small reads of the cached column.

 could not have an order-preserving sequence of

then mice and elephants.


Finally, in place analytics


## Mice and elephants

parquet problem with column groups

## the zng data model

The zng data model comprises an unbounded sequence of zng streams, where
each stream comprises a finite sequence of "rows", and each row conforms to
an arbitrary schema.  In this way, you can think of a zng stream as a diverse
collection of sql-like tables where the rows from each table are interspersed
amongst each other in a deterministic order.  (In the zq implementation, a zng row
is a zng.Record and a schema is a zng.TypeRecord.)

Each zng stream has it's own type context that defines the schema that each
row corresponds to. A zng stream is fully self-contained and
self describing.   There is no need for external schema definitions or a
centralized schema registry to decode a zng stream.

Zng streams can represent data at rest on a storage system or data in flight,
e.g., as the foundation of a data communication layer.  Like FlexBuffers,
the zng format was designed so that serialized form of the data is the same
as the in-memory form of the data so there is no need to unmarshal zng data
into native programming data structures.  Thus, analytics may be carried out
directly on the zng row data.  We call this "in-place analytics".

## enter zst

storage object.
convert a zng stream to a zst file and back.

layout is:
* a vector of type IDs
* a collection of records for each type ID
* each collection...
