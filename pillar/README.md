# Pillar

> This directory contains initial and rough ideas for a columnar-oriented
> version of the zng data model.  We're drafting this with the idea that
> the name "pillar" may end up naming the overall approach (the data model,
> the type system, the serialization format, the column structure, the
> row structure, the query language, etc) maybe subsuming the name "zng".

> There are many advantages to this
> approach of vertical integration and one could argue the time has come
> for a holistic approach like this.  Of course, this is not about reinventing
> the wheel, but rather about taking the best ideas that exist today in many
> different and diverse systems and designs, and leveraging them all with
> some new stuff to create a new and better overall
> approach to data organization, archive, analytics, and search.

A "pillar" is a column and thus an apt name for storing data in columnar format.
As you know, columnar format makes for efficient analytics.  The devil's
in the details and it turns out that making columnar format work for real-world,
heterogenous data is not at all straightforward.

The pillar architecture is all about making this hard stuff easy.
Bear with us here, because it's hard to make things easy.

Computers love columns.  Random-access memory, after all, is one big column.
But the real world progresses as rows.  It takes a bit of work to
put these rows into the one big column.
If you do the work, you can create columns to abstractly represent an organization
of the rows and make things more efficient.

But the world works in rows.

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

## the logical mind and the relational table model

Edgar Codd knew all this in the late 1960s.  He put down his prescient ideas
in his famous 1970 paper.  His ideas were quite theorictal but he influenced
the creation of the table-based database model that survives to this day.

## dataframes

Really?  this is your response

## enter pillar



The problem underlying all of this...

people tried putting shit in sql databases.  it didn't go well.

how do you know everything ahead of time?  you don't.  the data needs to be
self-describing and the query language needs to adapt to the data.  this might
sound impossible but it isn't.

strongly typed and self describing and suitable for analytics.

# Mice and elephants

parquet problem with column groups
