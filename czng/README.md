# Columnar zng

> This directory contains initial and rough ideas for a columnar-oriented
> version of the zng data model.

motivated by parquet - very wide schemas so anything can fit,
the so-called uber schemas.
the problem is that you need to define the schema ahead of time.
even the uber schema.

when building the data lake, how can you possibly foresee everything
that might be in the offering?

zng says, instead, the data should be self-describing and the schema
should be embedded alongside and within the data itself.  perhaps the
types and schema information should be sprinkled throughout.  do you
really want central planning for each and all of your data presentations?

we think not.

json went a long way here but only has five simple types and
it doesn't provide an ordering constraint for the keys of an object, i.e.,
the order of column in a table are not implied in json objects.
and when everything is a string there is abundantly painful overhead
converting strings to machine data.

dremel said columnar is good (like motherhood and apple pie)
and provided a data structure for mapping
hierarchical records with variable-length array values into a traditional
database table.  i.e., extending the database table with recursively
embedded tables.  yo dawg.

dremel is like machine-code for a type system.  it's not the type system.
maybe the type system and columnar format should, in fact, go together?

instead of thinking inside out, how about we think outside in?

people complain that data is messy in the real world.  yes it is.

how about we accept things as is
and reflect this messy reality into the data model, but in a way that
is sensible and actionable?

instead of having a discrete collection of schema-rigid tables each
comprised of like rows with relational capability amongst them all,
what if we we had a continuous stream of schema-flexible rows, where
each row corresponded an arbitrary schema type?

in other words, a zng stream comprises an arbitrary but *ordered* sequence
of rows emanating from an arbitrary set of schema-rigid data tables.
in this way, zng is schema-flexible.
