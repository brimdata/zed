# Zed Lake Design

  * [Data Pools](#data-pools)
  * [Lake Semantics](#lake-semantics)
    + [Initialization](#initialization)
    + [Create](#create)
    + [Branch](#branch)
    + [Load](#load)
      - [Data Segmentation](#data-segmentation)
    + [Log](#log)
    + [Merge](#merge)
    + [Rebase](#rebase)
    + [Query](#query)
      - [Meta-queries](#meta-queries)
      - [Transactional Semantics](#transactional-semantics)
      - [Time Travel](#time-travel)
    + [Merge Scan and Compaction](#merge-scan-and-compaction)
    + [Delete](#delete)
    + [Revert](#revert)
    + [Purge and Vacate](#purge-and-vacate)
  * [Search Indexes](#search-indexes)
    + [Index Rules](#index-rules)
      - [Field Rule](#field-rule)
      - [Type Rule](#type-rule)
      - [Aggregation Rule](#aggregation-rule)
  * [Cloud Object Architecture](#cloud-object-architecture)
    + [Immutable Objects](#immutable-objects)
      - [Data Objects](#data-objects)
      - [Commit History](#commit-history)
    + [Transaction Journal](#transaction-journal)
      - [Scaling a Journal](#scaling-a-journal)
      - [Journal Concurrency Control](#journal-concurrency-control)
      - [Configuration State](#configuration-state)
    + [Object Naming](#object-naming)
  * [Continuous Ingest](#continuous-ingest)
  * [Derived Analytics](#derived-analytics)
  * [Keyless Data](#keyless-data)
  * [Relational Model](#relational-model)
  * [Current Status](#current-status)
  * [CLI tool naming conventions](#cli-tool-naming-conventions)

A Zed lake is a cloud-native arrangement of data,
optimized for search, analytics, ETL, and data discovery
at very large scale based on data represented in accordance
with the [Zed data model](../formats).

## Data Pools

A lake is composed of _data pools_.  Each data pool is organized
according to its configured _pool key_.  Different data pools can have
different pool keys but all of the data in a pool must have the same
pool key.

The pool key is often a timestamp.  In this case, retention policies
and storage hierarchy decisions can be efficiently associated with
ranges of data over the pool key.

Data can be efficiently accessed via range scans composed of a
range of values conforming to the pool key.

A pool also has a configured sort order, either ascending or descending
and data is organized in the pool in accordance with this order.
Data scans may be either ascending or descending, and scans that
follow the configured order are generally more efficient than
scans that run in the opposing order.

Scans may also be range-limited but unordered.

If data loaded into a pool lacks the pool key, that data is still
imported but is not available to pool-key range scans.  Since it lacks
the pool key, all data without a key is grouped together as a "null" key
and cannot be efficiently range scanned.

Data may be indexed by field.  In this case field comparisons (i.e.,
searches for values in a particular field) are optimized by pruning
the data objects from a scan that do not contain the value(s) being searched.

> Future plans for indexing include full-text keyword indexing and
> type-based indexing (e.g., index all values that are IP addresses
> including values inside arrays, sets, and sub-records).

> Indexes may also hold aggregation partials so that configured aggregations
> or search-based aggregations can be greatly accelerated.

## Lake Semantics

The semantics of a Zed lake very loosely follows the nomenclature and
design patterns of `git`.  In this approach,
* a _lake_ is like a GitHub organization,
* a _pool_ is like a `git` repository,
* a _branch_ of a _pool_ is like a `git` branch,
* a _commit_ operation is like a `git commit`,
* and a pool _snapshot_ is like a `git checkout`.

A core theme of the Zed lake design is _ergonomics_.  Given the Git metaphor,
our goal here is that the Zed lake tooling be as easy and familiar as Git is
to a technical user.

While this design document is independent of any particular implementation,
we will illustrate the design concepts here with examples of `zed lake` commands.
Where the example commands shown are known to not yet be fully implemented in
the current Zed code, links are provided to open GitHub Issues.
Note that while this CLI-first approach provides an ergonomic way to experiment with
and learn the Zed lake building blocks, all of this functionality is also
exposed through an API to a cloud-based service.  Most interactions between
a user and a Zed lake would be via an application like
[Brim](https://github.com/brimdata/brim) or a
programming environment like Python/Pandas rather than via direct interaction
with `zed lake`.


### Initialization

A new lake is initialized with
```
zed lake init [path]
```

In all these examples, the lake identity is implied by its path (e.g., an S3
URI or a file system path) and may be specified by the `ZED_LAKE_ROOT`
environment variable when running `zed lake` commands on a local host.  In a
cloud deployment or running queries through an application, the lake path is
determined by an authenticated connection to the Zed lake service, which
explicitly denotes the lake name (analogous to how a GitHub user authenticates
access to a named GitHub organization).

### Create

A new pool is created with
```
zed lake create -p <name> [-orderby key[,key...][:asc|:desc]]
```
where `<name>` is the name of the pool within the implied lake instance,
`<key>` is the Zed language representation of the pool key, and `asc` or `desc`
indicate that the natural scan order by the pool key should be ascending
or descending, respectively, e.g.,
```
zed lake create -p logs -orderby ts:desc
```
Note that there may be multiple pool keys (implementation tracked in
[zed/2657](https://github.com/brimdata/zed/issues/2657)), where subsequent keys
act as the secondary sort key, tertiary sort key, and so forth.

If a pool key is not specified, then it defaults to the whole record, which
in the Zed language is referred to as "this".

The create command initiates a new pool with a single branch called `main`.

> Zed lakes can be used without thinking about branches.  When referencing a pool without
> a branch, the tooling presumes the "main" branch as the default, and everything
> can be done on main without having to think about branching.

### Branch

A branch is simply a named pointer to a commit object in the Zed lake.
Similar to Git, Zed commit objects are arranged into a tree and
represent the entire commit history of the lake.  (Technically speaking,
Git allows merging from multiple parents and thus Git commits form a
directed acyclic graph instead of a tree; Zed does not currently support
multiple parents in the commit object history.)

A branch is created with the `branch` command, e.g.,
```
zed lake branch -p logs@main staging
```
This creates a new branch called "staging" in pool "logs", which points to
the same commit object as the "main" branch.  Commits to the "staging" branch
will be added to the commit history without affecting the "main" branch
and each branch can be queried independently at any time.

### Load

Data is loaded and committed into a branch with the `load` command, e.g.,
```
zed lake load -p logs sample.ndjson
```
where `sample.ndjson` contains logs in NDJSON format.  Any supported format
(NDJSON, ZNG, ZSON, etc.) as well multiple files can be used here, e.g.,
```
zed lake load -p logs sample1.ndjson sample2.zng sample3.zson
```
CSV, JSON, Parquet, and ZST formats are not auto-detected so you must currently
specify `-i` with these formats, e.g.,
```
zed lake load -p logs -i parquet sample4.parquet
zed lake load -p logs -i zst sample5.zst
```

By default, the data is committed into the `main` branch of the pool.
An alternative branch may be specified using the `@` separator,
i.e., `<pool>@<branch>`.  Supposing there was a branch called `updates`,
data can be committed into this branch as follows:
```
zed lake load -p logs@updates sample.zng
```

Note that there is no need to define a schema or insert data into
a "table" as all Zed data is _self describing_ and can be queried in a
schema-agnostic fashion.  Data of any _shape_ can be stored in any pool
and arbitrary data _shapes_ can coexist side by side.

#### Data Segmentation

In a `load` operation, a commit is broken out into units called _data objects_
where a target object size is configured into the pool,
typically 100-500MB.  The records within each object are sorted by the pool key(s).
A data object is presumed by the implementation
to fit into the memory of an intake worker node
so that such a sort can be trivially accomplished.

Data added to a pool can arrive in any order with respect to the pool key.
While each object is sorted before it is written,
the collection of objects is generally not sorted.

### Log

Like `git log`, the command `zed lake log` prints a history of commit objects
starting from any commit.  The log can be displayed with the `log` command,
e.g.,
```
zed lake log -p pool@branch
```
To understand the log contents, the `load` operation is actually
decomposed into two steps under the covers:
an "add" step stores one or more
new immutable data objects in the lake and a "commit" step
materializes the objects into a branch with an ACID transaction.
This updates the branch pointer to point at a new commit object
referencing the data objects where the new commit object's parent
points at the branch's previous commit object, thus forming a path
through the object tree.

> Note that following the pointers of a sequence of commit objects
> each stored independently in cloud storage can have tremendously
> high latency.  Fortunately, all of these objects are immutable and
> any commit object thus has a predetermined state that can be computed
> from its predecessors and persisted as a snapshot and cached
> in memory (or in redis etc).

Every commit object and data object is named by and referenced
using globally unique [KSUIDs](https://github.com/segmentio/ksuid),
called a `commit ID` or a data `object ID`, respectively.

The log command prints the commit ID of each commit object in that path
from the current pointer back through history to the first commit object.

A commit object includes
an optional author and message, along with a required timestamp,
that is stored in the commit journal for reference.  These values may
be specified as options to the `load` command, and are also available in the
API for automation.  For example,
```
zed lake load -p logs -user user@example.com -message "new version of prod dataset" ...
```
This metadata is carried in a description record attached to
every journal entry, which has a Zed type signature as follows:
```
{
    Author: string,
    Date: time,
    Description: string,
    Data: <any>
}
```
None of the fields are used by the Zed lake system for any purpose
except to provide information about the commit object to the end user
and/or end application.  Any ZSON/ZNG data can be stored in the `Data` field
allowing external applications to implement arbitrary data provenance and audit
capabilities by embedding custom metadata in the commit journal.

> The Data field is not yet implemented.

### Merge

Data is merged from one branch into another with the `merge` command, e.g.,
```
zed lake merge -p logs@updates main
```
where the "updates" branch is being merged into the "main" branch.

A merge operation finds a common ancestor in the commit history then
computes the set of changes needed for the target branch to reflect the
data additions and deletions in the source branch.
While the merge operation is performed, data can still be written
to both branches and queries performed.  Newly written data remains in the
branch while all of the data present at merge initiation is merged into the
parent.

This Git-like behavior for a data lake provides a clean solution to
the live ingest problem.
For example, data can be continuously ingested into a branch of main called `live`
and orchestration logic can periodically merge updates from branch `live` to
branch `main`, possibly compacting and indexing data after the merge
according to configured policies and logic.

### Rebase

> TBD


### Query

Data is read from one or more pools with the `query` command.  The pool/branch names
are specified with `from` at the beginning of the Zed query along with an optional
time range using `range` and `to`.  The default output format is ZSON for
terminals and ZNG otherwise, though this can be overridden with
`-f` to specify one of the various supported output formats.

If a pool name is provided to `from` without a branch name, then branch
"main" is assumed.

This example reads every record from the full key range of the `logs` pool
and sends the results as ZSON to stdout.

```
zed lake query -f zson 'from logs'
```

We can narrow the span of the query by specifying the key range, where these
values refer to the pool key:
```
zed lake query -z 'from logs range 2018-03-24T17:36:30.090766Z to 2018-03-24T17:36:30.090758Z'
```
These range queries are efficiently implemented as the data is laid out
according to the pool key and seek indexes keyed by the pool key
are computed for each data object.

A much more efficient format for transporting query results is the
row-oriented, compressed binary format ZNG.  Because ZNG
streams are easily merged and composed, query results in ZNG format
from a pool can be can be piped to another `zed query` instance, e.g.,
```
zed lake query -f zng 'from logs' | zed query -f table 'count() by field' -
```
Of course, it's even more efficient to run the query inside of the pool traversal
like this:
```
zed lake query -f table 'from logs | count() by field'
```
By default, the `query` command scans pool data in pool-key order though
the Zed optimizer may, in general, reorder the scan to optimize searches,
aggregations, and joins.
An order hint can be supplied to the `query` command to indicate to
the optimizer the desired processing order, but in general, `sort` operators
should be used to guarantee any particular sort order.

Arbitrarily complex Zed queries can be executed over the lake in this fashion
and the planner can utilize cloud resources to parallelize and scale the
query over many parallel workers that simultaneously access the Zed lake data in
shared cloud storage (while also accessing locally- or cluster-cached copies of data).

#### Meta-queries

Commit history, metadata about data objects, lake and pool configuration,
etc. can all be queried and
returned as Zed data, which in turn, can be fed into Zed analytics.
This allows a very powerful approach to introspecting the structure of a
lake making it easy to measure, tune, and adjust lake parameters to
optimize layout for performance.

These structures are introspected using meta-queries that simply
specify a metadata source using an extended syntax in the `from` operator.
There are three types of meta-queries:
* `from :<meta>` - lake level
* `from pool:<meta>` - pool level
* `from pool@branch<:meta>` - branch level

`<meta>` is the name of the metadata being queried. The available metadata
sources vary based on level.

For example, a list of pools with configuration data can be obtained
in the ZSON format as follows:
```
zed lake query -Z "from :pools"
```
This meta-query produces a list of branches in a pool called `logs`:
```
zed lake query -Z "from logs:branches"
```
Since this is all just Zed, you can filter the results just like any query,
e.g., to look for particular branch:
```
zed lake query -Z "from logs:branches | branch.name=='main'"
```

This meta-query produces a list of the data objects in the `live` branch
of pool `logs`:
```
zed lake query -Z "from logs@live:objects"
```

You can also pretty-print in human-readable form most of the metadata Zed records
using the "lake" format, e.g.,
```
zed lake query -f lake "from logs@live:objects"
```

> TODO: we need to document all of the meta-data sources somewhere.

#### Transactional Semantics

The "commit" operation is _transactional_.  This means that a query scanning
a pool sees its entire data scan as a fixed "snapshot" with respect to the
commit history.  In fact, the Zed language allows a commit object (created
at any point in the past) to be specified with the `@` suffix to a
pool reference, e.g.,
```
zed lake query -z 'from logs@1tRxi7zjT7oKxCBwwZ0rbaiLRxb | count() by field'
```
In this way, a query can time-travel through the journal.  As long as the
underlying data has not been deleted, arbitrarily old snapshots of the Zed
lake can be easily queried.

If a writer commits data after a reader starts scanning, then the reader
does not see the new data since it's scanning the snapshot that existed
before these new writes occurred.

Also, arbitrary metadata can be committed to the log as described below,
e.g., to associate index objects or derived analytics to a specific
journal commit point potentially across different data pools in
a transactionally consistent fashion.

#### Time Travel

While time travel through commit history provides one means to explore
past snapshots of the commit history, another means is to use a timestamp.
Because the entire history of branch updates is stored in a transaction journal
and each entry contains a timestamp, branch references can be easily
navigated by time.  For example, a list of branches of a pool's past
can be created by scanning the branches log and stopping at the largest
timestamp less than or equal to the desired timestamp.  Likewise, a branch
can be located in a similar fashion, then its corresponding commit object
can be used to construct that data of that branch at that past point in time.

### Merge Scan and Compaction

To support _sorted scans_,
data from overlapping objects is read in parallel and merged in sorted order.
This is called the _merge scan_.

However, if many overlapping data objects arise, performing a merge scan
on every read can be inefficient.
This can arise when
many random data `load` operations involving perhaps "late" data
(e.g., the pool key is a timestamp and records with old timestamp values regularly
show up and need to be inserted into the past).  The data layout can become
fragmented and less efficient to scan, requiring a scan to merge data
from a potentially large number of different objects.

To solve this problem, the Zed lake design follows the
[LSM](https://en.wikipedia.org/wiki/Log-structured_merge-tree) design pattern.
Since records in each data object are stored in sorted order, a total order over
a collection of objects (e.g., the collection coming from a specific set of commits)
can be produced by executing a sorted scan and rewriting the results back to the pool
in a new commit.  In addition, the objects comprising the total order
do not overlap.  This is just the basic LSM algorithm at work.

To perform an LSM rollup, the `compact` command (implementation tracked
via [zed/2977](https://github.com/brimdata/zed/issues/2977))
is like a "squash" to perform LSM-like compaction function, e.g.,
```
zed lake compact -p logs <tag>
(merged commit <tag> printed to stdout)
```
After compaction, all of the objects comprising the new commit are sorted
and non-overlapping.
Here, the objects from the given commit tag are read and compacted into
a new commit.  Again, until the data is actually committed,
no readers will see any change.

Unlike other systems based on LSM, the rollups here are envisioned to be
run by orchestration agents operating on the Zed lake API.  Using
meta-queries, an agent can introspect the layout of data, perform
some computational geometry, and decide how and what to compact.
The nature of this orchestration is highly workload dependent so we plan
to develop a family of data-management orchestration agents optimized
for various use cases (e.g., continuously ingested logs vs. collections of
metrics that should be optimized with columnar form vs. slowly-changing
dimensional datasets like threat intel tables).

An orchestration layer outside of the Zed lake is responsible for defining
policy over
how data is ingested and committed and rolled up.  Depending on the
use case and workflow, we envision that some amount of overlapping data objects
would persist at small scale and always be "mixed in" with other overlapping
data during any key-range scan.

> Note: since this style of data organization follows the LSM pattern,
> how data is rolled up (or not) can control the degree of LSM write
> amplification that occurs for a given workload.  There is an explicit
> tradeoff here between overhead of merging overlapping objects on read
> and LSM write amplification to organize the data to avoid such overlaps.

> Note: we are showing here manual, CLI-driven steps to accomplish these tasks
> but a live data pipeline would automate all of this with orchestration that
> performs these functions via a service API, i.e., the same service API
> used by the CLI operators.

### Delete

Data objects can be deleted with the `delete` command.  This command
simply removes the data from the branch without actually deleting the
underlying data objects thereby allowing time travel to work in the face
of deletes.

For example, this command deletes the three objects referenced
by the data object IDs:
```
zed lake delete -p logs <id> <id> <id>
```

> TBD: when a scan encounters an object that was physically deleted for
> whatever reason, it should simply continue on and issue a warning on
> the query endpoint "warnings channel".

### Revert

The actions in a commit can be reversed with the `revert` command.  This
command applies the inverse steps in a new commit to the tip of the indicated
branch.  Any data loaded in a reverted commit remains in the lake but no longer
appears in the branch.  The new commit may recursively be reverted by an
additional revert operation.

For example, this command reverts the commit referenced by commit ID
`<commit>`.
```
zed lake revert -p logs <commit>
```

### Purge and Vacate

Data can be deleted with the DANGER-ZONE command `zed lake purge`
(implementation tracked in [zed/2545](https://github.com/brimdata/zed/issues/2545)).
The commits still appear in the log but scans at any time-travel point
where the commit is present will fail to scan the deleted data.

Alternatively, old data can be removed from the system using a safer
command (but still in the DANGER-ZONE), `zed lake vacate` (also
[zed/2545](https://github.com/brimdata/zed/issues/2545)) which moves
the tail of the commit journal forward and removes any data no longer
accessible through the modified commit journal.

## Search Indexes

Unlike traditional indexing systems based on an inverted-keyword index,
indexing in Zed is decentralized and incremental.  Instead of rolling up
index data structures across many data objects, a Zed lake stores a small
amount of index state for each data object.  Moreover, the design relies on
indexes only to enhance performance, not to implement the lake semantics.
Thus, indexes need not exist to operate and can be incrementally added or
deleted without large indexing jobs needing to rebuild a monolithic index
after each configuration change.

To optimize pool scans, the lake design relies on the well-known pruning
concept to skip any data object that the planner determines can be skipped
based on one or more indexes of that object.  For example, if an object
has been indexed for field "foo" and the query
```
foo == "bar" | ...
```
is run, then the scan will consult the "foo" index and skip the data object
if the value "bar" is not in that index.

This approach works well for "needle in the haystack"-style searches.  When
a search hits every object, this style of indexing would not eliminate any
objects and thus does not help.

While an individual index lookup involves latency to cloud storage to lookup
a key in each index, each lookup is cheap and involves a small amount of data
and the lookups can all be run in parallel, even from a single node, so
the scan schedule can be quickly computed in a small number of round-trips
(that navigate very wide B-trees) to cloud object storage.

### Index Rules

An index of an object is created by applying an _index rule_ to a data object
and recording the binding to the pool's commit journal.  Once the index is
available, the query planner can use it to optimize Zed lake scans.

Rules come in three flavors:
* field rule - index all values of a named field
* type rule - index all values of all fields of a given type
* aggregation rule - index any results computed by any Zed script run
over the data object and keyed by one or more named fields, typically used
to compute partial aggregations.

Rules are organized into groups by name and defined at the lake level
so that any named group of rules can be applied to data objects from
any pool.  The group name provides no meaning beyond a reference to
a set of index rules at any given time.

Rules are created with `zed lake index create`,
deleted with `zed lake index drop`, and applied with
`zed lake index apply`.

#### Field Rule

A field rule indicates that all values of a field be indexed.
For example,
```
zed lake index create IndexGroupEx field foo
```
adds a field rule for field `foo` to the index group named `IndexGroupEx`.
This rule can then be applied to a data object having a given `<tag>`
in a pool, e.g.,
```
zed lake index apply -p logs IndexGroupEx <tag>
```
The index is created and a transaction put (somewhere).  Once this transaction
has been committed to the pool's journal, the index is available for use
by the query planner.

#### Type Rule

A type rule indicates that all values of any field of a specified type
be indexed where the type signature uses Zed type syntax.
For example,
```
zed lake index create IndexGroupEx type ip
```
creates a rule that indexes all IP addresses appearing in fields of type `ip`
in the index group `IndexGroupEx`.

#### Aggregation Rule

An aggregation rule allows the creation of any index keyed by one or more fields
(primary, second, etc.), typically the result of an aggregation.
The aggregation is specified as a Zed query.
For example,
```
zed lake index create IndexGroupEx agg "count() by field"
```
creates a rule that creates an index keyed by the group-by keys whose
values are the partial-result aggregation given by the Zed expression.

> This is not yet implemented.  The query planner would replace any full object
> scan with the needed aggregation with the result given in the index.
> Where a filter is applied to match one row of the index, that result could
> likewise be extracted instead of scanning the entire object.
> This capability is not generally useful for interactive search and analytics
> (except for optimizations that suit the interactive app) but rather is a powerful
> capability for application-specific workflows that know the pre-computed
> aggregations that they will use ahead of time, e.g., beacon analysis
> of network security logs.

## Cloud Object Architecture

The Zed lake semantics defined above are achieved by mapping the
lake and pool abstractions onto a key-value cloud object store.

Every data element in a Zed lake is either of two fundamental object types:
* a single-writer _immutable object_, or
* a multi-writer _transaction journal_.

### Immutable Objects

All imported data in a data pool is composed of immutable objects, which are organized
around a primary data object.  Each data object is composed of one or more immutable objects
all of which share a common, globally unique identifier,
which is referred to below generically as `<id>` below.

These identifiers are [KSUIDs](https://github.com/segmentio/ksuid).
The KSUID allocation scheme
provides a decentralized solution for creating globally unique IDs.
KSUIDs have embedded timestamps so the creation time of
any object named in this way can be derived.  Also, a simple lexicographic
sort of the KSUIDs results in a creation-time ordering (though this ordering
is not relied on for causal relationships since clock skew can violate
such an assumption).

> While a Zed lake is defined in terms of a cloud object store, it may also
> be realized on top of a file system, which provides a convenient means for
> local, small-scale deployments for test/debug workflows.  Thus, for simple use cases,
> the complexity of running an object-store service may be avoided.

#### Data Objects

An immutable object is created by a single writer using a globally unique name
with an embedded KSUID.  
New objects are written in their entirety.  No updates, appends, or modifications
may be made once an object exists.  Given these semantics, any such object may be
trivially cached as its name nor content ever change.

Since the object's name is globally unique and the
resulting object is immutable, there is no possible write concurrency to manage
with respect to a given object.

A data object is composed of
* the primary data object stored as one or two objects (for row and/or column layout),
* an optional seek index, and
* zero or more search indexes.

Data objects may be either in row form (i.e., ZNG) or column form (i.e., ZST),
or both forms may be present as a query optimizer may choose to use whatever
representation is more efficient.
When both row and column data objects are present, they must contain the same
underlying Zed data.

Immutable objects are named as follows:

|object type|name|
|-----------|----|
|column data|`<pool-id>/data/<id>.zst`|
|row data|`<pool-id>/data/<id>.zng`|
|row seek index|`<pool-id>/data/<id>-seek.zng`|
|search index|`<pool-id>/index/<id>-<index-id>.zng`|

`<id>` is the KSUID of the data object.
`<index-id>` is the KSUID of an index object created according to the
index rules described above.  Every index object is defined
with respect to a data object.

The seek index maps pool key values to seek offset in the ZNG file thereby
allowing a scan to do a partial GET of the ZNG object when scanning only
a subset of data.

> Note the ZST format will have seekable checkpoints based on the sort key that
> are encoded into its metadata section so there is no need to have a separate
> seek index for the columnar object.

#### Commit History

A pool's commit history is the definitive record of the evolution of data in
that pool in a transactionally consistent fashion.

Each commit object entry is identified with its `commit ID`.
Objects are immutable and uniquely named so there is never a concurrent write
condition.

The "add" and "commit" operations are transactionally stored
in a chain of commit objects.  Any number of adds (and deletes) may appear
in a commit object.  All of the operations that belong to a commit are
identified with a commit identifier (ID).

As each commit object points to its parent (except for the initial commit
in main), the collection of commit objects in a pool forms a tree.

Each commit object contains a sequence of _actions_:

* `Add` to add a data object reference to a pool,
* `Delete` to delete a data object reference from a pool,
* `AddIndex` to bind an index object to a data object to prune the data object
from a scan when possible using the index,
* `DeleteIndex` to remove an index object reference to its data object, and
* `Commit` for providing metadata about each commit.

The actions are not grouped directly by their commit tag but instead each
action embeds the KSUID of its commit tag.

By default, `zed lake log` outputs an abbreviated form of the log as text to
stdout, similar to the output of `git log`.

However, the log represents the definitive record of a pool's present
and historical content, and accessing its complete detail can provide
insights about data layout, provenance, history, and so forth.  Thus,
Zed lake provides a means to query a pool's configuration state as well,
thereby allowing past versions of the complete pool and branch configurations
as well as all of their underlying data to be subject to time travel.
To interrogate the underlying transaction history of the branches and
their pointers, simply query a pool's "branchlog" via the syntax `<pool>:branchlog`.

For example, to aggregate a count of each journal entry type of the pool
called `logs`, you can simply say:
```
zed lake query "from logs:branchlog | count() by typeof(this)"
```
Since the Zed system "typedefs" each journal record with a named type,
this kind of query gives intuitive results.  There is no need to implement
a long list of features for journal introspection since the data in its entirety
can be simply and efficiently processed as a ZNG stream.

> Note that the branchlog meta-query source is not yet implemented.

### Transaction Journal

State that is mutable is built upon a transaction journal of immutable
collections of entries.  In this way, there are no objects in the
storage footprint that are ever modified.  Instead, the journal captures
changes and journal snapshots are used to provide synchronization points
for efficient access to the journal (so the entire journal need not be
read to create the current state) and old journal entries may be removed
based on retention policy.

The journal may be updated concurrently by multiple writers so concurrency
controls are included (see [Journal Concurrency Control](#journal-concurrency-control)
below) to provide atomic updates.

A journal entry simply contains actions that modify the visible "state" of
the pool by changing branch name to commit object mappings.  Note that
adding a commit object to a pool changes nothing until a branch pointer
is mutated to point at that object.

Each atomic journal commit object is a ZNG file numbered 1 to the end of journal (HEAD),
e.g., `1.zng`, `2.zng`, etc., each number corresponding to a journal ID.
The 0 value is reserved as the null journal ID.
The journal's TAIL begins at 1 and is increased as journal entries are purged.
Entries are added at the HEAD and removed from the TAIL.
Once created, a journal entry is never modified but it may be deleted and
never again allocated.
There may be 1 or more entries in each commit object.

Each journal entry implies a snapshot of the data in a pool.  A snapshot
is computed by applying the transactions in sequence from entry TAIL to
the journal entry in question, up to HEAD.  This gives the set of commit tags
that comprise a snapshot.

The set of branch pointers in a pool is assembled at any point in the journal's history
by scanning a journal that includes ADD, UPDATE, and DELETE actions for the
mapping of a branch name to a commit object.  A timestamp is recorded in
each action to provide for time travel.

For efficiency, a journal entry's snapshot may be stored as a "cached snapshot"
alongside the journal entry.  This way, the snapshot at HEAD may be
efficiently computed by locating the most recent cached snapshot and scanning
forward to HEAD.

#### Scaling a Journal

When the sizes of the journal snapshot files exceed a certain size
(and thus becomes too large to conveniently handle in memory),
the snapshots can be converted to and stored
in an internal sub-pool called the "snapshot pool".  The snapshot pool's
pool key is the "from" value (of its parent pool key) from each commit action.
In this case, commits to the parent pool are made in the same fashion,
but instead of snapshotting updates into a snapshot ZNG file,
the snapshots are committed to the journal sub-pool.  In this way, commit histories
can be rolled up and organized by the pool key.  Likewise, retention policies
based on the pool key can remove not just data objects from the main pool but
also data objects in the journal pool comprising committed data that falls
outside of the retention boundary.

> Note we currently record a delete using only the object ID.  In order to
> organize add and delete actions around key spans, we need to add the span
> metadata to the delete action just as it exists in the add action.

#### Journal Concurrency Control

To provide for atomic commits, a writer must be able to atomically update
the HEAD of the log.  There are three strategies for doing so.

First, if the cloud service offers "put-if-missing" semantics, then a writer
can simply read the HEAD file and use put-if-missing to write to the
journal at position HEAD+1.  If this fails because of a race, then the writer
can simply write at position HEAD+2 and so forth until it succeeds (and
then update the HEAD object).  Note that there can be a race in updating
HEAD, but HEAD is always less than or equal to the real end of journal,
and this condition can be self-corrected by probing for HEAD+1 whenever
the HEAD of the journal is accessed.

> Note that put-if-missing can be emulated on a local file system by opening
> a file for exclusive access and checking that it has zero length after
> a successful open.

Second, strong read/write ordering semantics (as exists in Amazon S3)
can be used to implement transactional journal updates as follows:
* _TBD: this is worked out but needs to be written up_

Finally, since the above algorithm requires many round trips to the storage
system and such round trips can be tens of milliseconds, another approach
is to simply run a lock service as part of a cloud deployment that manages
a mutex lock for each pool's journal.

#### Configuration State

Configuration state describing a lake or pool is also stored in mutable objects.
Zed lakes simply use a commit journal to store configuration like the
list of pools and pool attributes, indexing rules used across pools,
etc.  Here, a generic interface to a commit journal manages any configuration
state simply as a key-value store of snapshots providing time travel over
the configuration history.

### Object Naming

```
<lake-path>/
  lake.zng
  pools/
    HEAD
    TAIL
    1.zng
    2.zng
    ...
  index_rules/
    HEAD
    TAIL
    1.zng
    2.zng
    ...
    ...
  <pool-id-1>/
    branches/
      HEAD
      TAIL
      1.zng
      2.zng
      ...
    commits/
      <id1>.zng
      <id2>.zng
      ...
    data/
      <id1>.{zng,zst}
      <id2>.{zng,zst}
      ...
    index/
      <id1>-<index-id-1>.zng
      <id1>-<index-id-2>.zng
      ...
      <id2>-<index-id-1>.zng
      ...
  <pool-id-2>/
  ...
```

## Continuous Ingest

While the description above is very batch oriented, the Zed lake design is
intended to perform scalably for continuous streaming applications.  In this
approach, many small commits may be continuously executed as data arrives and
after each commit, the data is immediately readable.

To handle this use case, the _journal_ of branch commits is designed
to scale to arbitrarily large footprints as described earlier.

## Derived Analytics

To improve the performance of predictable workloads, many use cases of a
Zed lake pre-compute _derived analytics_ or a particular set of _partial
aggregations_.

For example, the Brim app displays a histogram of event counts grouped by
a category over time.  The partial aggregation for such a computation can be
configured to run automatically and store the result in a pool designed to
hold such results.  Then, when a scan is run, the Zed analytics engine
recognizes when the DAG of a query can be rewritten to assemble the
partial results instead of deriving the answers from scratch.

When and how such partial aggregations are performed is simply a matter of
writing Zed queries that take the raw data and produce the derived analytics
while conforming to a naming model that allows the Zed lake to recognize
the relationship between the raw data and the derived data.

> TBD: Work out these details which are reminiscent of the analytics cache
> developed in our earlier prototype.

## Keyless Data

This is TBD.  Data without a key should be accepted some way or another.
One approach is to simply assign the "zero-value" as the pool key; another
is to use a configured default value.  This would make key-based
retention policies more complicated.

Another approach would be to create a sub-pool on demand when the first
keyless data is encountered, e.g., `pool-name.$nokey` where the pool key
is configured to be "this".  This way, an app or user could query this pool
by this name to scan keyless data.

## Relational Model

Since a Zed lake can provide strong consistency, workflows that manipulate
data in a lake can utilize a model where updates are made to the data
in place.  Such updates involve creating new commits from the old data
where the new data is a modified form of the old data.  This provides
emulation of row-based updates and deletes.

If the pool-key is chosen to be "this" for such a use case, then unique
rows can be maintained by trivially detected duplicates (because any
duplicate row will be adjacent when sorted by "this") so that duplicates are
trivially detected.

Efficient upserts can be accomplished because each data object is sorted by the
pool key.  Thus, an upsert can be sorted then merge-joined with each
overlapping object.  Where data objects produce changes and additions, they can
be forwarded to a streaming add operator and the list of modified objects
accumulated.  At the end of the operation, then new commit(s) along with
the to-be-deleted objects can be added to the journal in a single atomic
operation.  A write conflict occurs if there are any other deletes added to
the list of to-be-deleted objects.  When this occurs, the transaction can
simply be restarted.  To avoid inefficiency of many restarts, an upsert can
be partitioned into smaller objects if the use case allows for it.

> TBD: Finish working through this use case, its requirements, and the
> mechanisms needed to implement it.  Write conflicts will need to be
> managed at a layer above the journal or the journal extended with the
> needed functionality.

## Current Status

The initial prototype has been simplified as follows:
* transaction journal incomplete
* no recursive journal pool
* no columnar support

## CLI tool naming conventions

> This is a work in progress.

Now that we have a comprehensive branching model, the `-p` argument to the
`zed lake` commands needs to be revisited.

We need to have a consistent syntax to refer to pools, branches, and commits.
Some commands require branch names (e.g., branch, rename, merge, etc) while
others require a commit (e.g., log or a scan in the Zed `from` operator).

The proposal on the table is to refer to commits things and branch references
both as an `@` character following the pool being referenced, e.g.,
```
pool@commit
```
where commit is a KSUID of the commit object, or
```
pool@branch
```
where branch is a name (that cannot be a KSUID).

Confusion arises because a commit can be referred to by a branch name
and, depending on context, a branch can refer to a commit or to
a mutable reference of that branch.  For the technical discussion here, we will
refer to the branch/commit dichotomy as a `commitish` and the l-value
nature of a branch as a `ref`.

Both a `commitish` and a `ref` may drop omit `@` suffix
and refer solely to the pool name, in which case it becomes pool@main.

Complicating matters further, a `ref` may be specified as a simple name
without the `pool@` qualifier when the pool context is known from
another argument as in merge or rename.  We will refer to this
plain name as `sref` below (short `ref`).

In some contexts (e.g., `create` and `drop`),
only a pool name makes sense.  We will refer to this as `pool` here.

Also, we want some way to set the "current branch" so that the pool
and "branch" is implied and the implied branch can be either a `commitish`
or a `ref`.

* branch `commitish` `sref` - create branch `sref` with parent commit object `commitish`
* branch -d `ref` - delete branch
* create `pool` - create the pool named pool
* delete `ref` `id` `id` ... - delete data objects from branch `ref` and update the branch to point to the new commit object
* drop `pool` - delete a data pool
* index `ref` `id` `id` ... - create index objects if they don't exist and record them in branch `ref`
* load `ref` `file` ... - load the files or stdin into data objects, create a commit object for the new data that points backward to the head of `ref`, and update branch `ref` to point to the new commit
* log `commitish`- display the commit object history starting at `commitish`
* ls - list pools in a lake
* ls `pool` - list branch names of a pool
* merge `commitish` `sref` - merge a commit history rooted at `commitish` into a branch `sref`
* query: `from commitish` - data scan
* query: `from commitish:meta` branch or commit meta-scan (no defaulting to main here as that would be pool-level meta)
* query: `from pool:meta`
* query: `from :meta`
* rebase `ref` `commitish` - rebase a branch `ref` onto the commit object history starting at `commitish` (this is peculiar because `ref` could imply the pool name for `commitish` though this will usually be a branch name)
* rename `pool` `pool` - change name of a data pool
* revert `ref` `commitish`- undo the commit at `commitish` with a new commit object and updates the branch `ref` to point at the new commit; typically this commit would be in the branches history but it doesn't have to be (but would typically fail a consistency check if it isn't in the history).  We could prevent this condition with a check.

> Note we can simplify the rules about meta-query not allowing the default
> to "main" by partitioning the meta names and using the name to disambiguate
> between the pool-level, branch-level, and commit-level meta-queries...
