# Zesty - A New Data Architecture for Hybrid Logs and Analytics Processing

> Work in progress.  Rough thoughts on what we're doing, how it's different,
> and why it's important.

## Background - HLAP

Why do we need a new approach to search and analytics?

Here's the thing.  Data is messy.  If it wasn't, you could just load everything
up into a SQL database or data warehouse and query what you're looking for
and analyze the results.

But this doesn't work like magic.  So, people create clean schemas that they
can precisely reason about and
shoehorn all the messy data into the rigid structure of their schemas via
so-called ETL pipelines that land data into OLAP schemas.
There is a whole industry built around ETL.  If you can manage to get all your data
into a schema-rigid OLAP system like clickhouse, analytics processing
can work really well.

The problem is that maintaining schemas and ETL pipelines is hard and expensive.
If data formats change, sometimes unexpectedly, you have to update the ETL rules
and maybe even change your queries.  Or you pay service providers to do the
ETL for you.

Given these challenges, there is a whole different approach based on log search and analytics.
Here, the idea is to just throw everything you have into a massive log store
and run ad hoc searches and analytics using on-the-fly schema inference.
Here, people realized a little schema goes along way, so you can configure rules at
ingest that can transform raw forms of data into somewhat richer forms (e.g., turning
strings that look like IP addresses in to native IP types), but still, there
is no requirement to define tables up front with rigid schemas in which
all data must fit.

Okay, we have OLAP and logs, but wait there's more.  At some point, technologists began
to realize that the semi-structured nature of modern application data was
hard to fit in a schema-rigid relational table, so transactional data stores
based on JSON data stores emerged, like mongo and cassandra.  These systems
provide ACID semantics of document updates and rich DSLs for querying and
searching document data for analytics and BI.  In a sense, they provide the flexibility
of ad hoc search systems with the advantage of semi-structured representations
of the JSON data type.  Mongo went so far as to create BSON, which is a more
binary-efficent and type-rich variation of the JSON object model.
Along with other more traditional relational approaches that combined ACID
transactions with analytics processing, this approach became known as
"hybrid transaction/analytics processing" systems or HTAP.
