# Zed Lake Design

  * [Data Pools](#data-pools)
  * [Lake Semantics](#lake-semantics)
    + [Initialization](#initialization)
    + [New](#new)
    + [Load](#load)
    + [Query](#query)
    + [Add and Commit](#add-and-commit)
      - [Transactional Semantics](#transactional-semantics)
      - [Data Segmentation](#data-segmentation)
    + [Merge](#merge)
    + [Squash and Delete](#squash-and-delete)
    + [Purge and Vacate](#purge-and-vacate)
    + [Log](#log)
  * [Cloud Object Naming](#cloud-object-naming)
    + [Immutable Objects](#immutable-objects)
      - [Data Objects](#data-objects)
      - [Search Indexes](#search-indexes)
    + [Mutable Objects](#mutable-objects)
      - [Commit Journal](#commit-journal)
      - [Scaling a Journal](#scaling-a-journal)
      - [Journal Concurrency Control](#journal-concurrency-control)
      - [Configuration State](#configuration-state)
    + [Object Naming](#object-naming)
  * [Continuous Ingest](#continuous-ingest)
  * [Lake Introspection](#lake-introspection)
  * [Derived Analytics](#derived-analytics)
  * [Keyless Data](#keyless-data)
  * [Relational Model](#relational-model)
  * [Current Status](#current-status)


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

A lake has a configured sort order, either ascending or descending
and data is organized in the lake in accordance with this order.
Data scans may be either ascending or descending, and scans that
follow the configured order are generally more efficient than
scans that run in the opposing order.

Scans may also be range-limited but unordered.

If data loaded into a pool lacks the pool key, that data is still
imported but is not available to pool-key range scans.  Since it lacks
the pool key, such data is instead organized around its "this" value.

> TBD: What is the interface for accessing non-keyed data?  Should this
> show up in the Zed language somehow?

## Lake Semantics

The semantics of a Zed lake loosely follows the nomenclature and
design patterns of `git`.  In this approach,
* a _lake_ is like a GitHub organization,
* a _pool_ is like a `git` repository,
* a _load_ operation is like a `git add` followed by a `git commit`,
* and a pool _snapshot_ is like a `git checkout`.

A core theme of the Zed lake design is _ergonomics_.  Given the git metaphor,
our goal here is that the Zed lake tooling be as easy and familiar as git is
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

### New

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
Note that there may be multiple pool keys, where subsequent keys act as the secondary,
tertiary, and so forth sort key.

If a pool key is not specified, then it defaults to the whole record, which
in the Zed language is referred to as "this".

### Load

Data is then loaded into a lake with the `load` command, .e.g.,
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

Note that there is no need to define a schema or insert data into
a "table" as all Zed data is _self describing_ and can be queried in a
schema-agnostic fashion.  Data of any _shape_ can be stored in any pool
and arbitrary data _shapes_ can coexist side by side.

### Query

Data is read from one or more pools with the `query` command.  The pool names
are specified with `from` at the beginning the Zed query along with an optional
time range using `range` and `to`.  The default output format is ZNG though this
can be overridden with `-f` to specify one of the various supported output
formats.

This example reads every record from the full key range of the `logs` pool
and sends the results as ZSON to stdout.

```
zed lake query -f zson 'from logs'
```

Or we can narrow the span of the query by specifying the key range.
```
zed lake query -z 'from logs range 2018-03-24T17:36:30.090766Z to 2018-03-24T17:36:30.090758Z'
```

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
zed lake query 'from logs | count() by field'
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
shared cloud storage (while also accessing locally cached copies of data).

### Add and Commit

Continuing the `git` metaphor, the Zed lake `load` operation is actually
decomposed into two steps: an `add` operation and a `commit` operation.
These steps can be explicitly run in stages, e.g.,
```
zed lake add -p logs sample.json
(commit <tag> etc. printed to stdout)
zed lake commit -p logs <tag>
```
A commit `<tag>` refers to one or more data objects stored in the
data pool.  In general, a commit tag is simply a shortcut
for the set of object tags that comprise the commit and otherwise
has no meaningful semantics to the Zed execution engine.
Both commit and data tags are named using the same globally unique
allocation of [KSUIDs](https://github.com/segmentio/ksuid).

After an add operation, all pending commits are stored in a staging area
and the `zed lake status` command can be used to inspect the status of
all of the staged data.  The `zed lake squash` command may be used to
combine multiple staged commits into a single entity with a new
commit tag.  

The `zed lake clear` command removes commits from staging before they are
merged (planned implementation of this is tracked in
[zed/2579](https://github.com/brimdata/zed/issues/2579)).

Likewise, you can stack multiple adds and commit them all at once, e.g.,
```
zed lake add -p logs sample1.json
(<tag-1> etc. printed to stdout)
zed lake add -p logs sample2.parquet
(<tag-2> etc. printed to stdout)
zed lake add -p logs sample3.zng
(<tag-3> etc. printed to stdout)
zed lake commit -p logs <tag-1> <tag-2> <tag-3>
```
The commit command also takes an optional title and message that is stored
in the commit journal for reference.  For example,
```
zed lake commit -p logs -user user@example.com -message "new version of prod dataset" <tag>
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
except to provide information about the journal commit to the end user
and/or end application.  Any ZSON/ZNG data can be stored in the `Data` field
allowing external applications to implement arbitrary data provenance and audit
capabilities by embedding custom metadata in the commit journal.

#### Transactional Semantics

The `commit` operation is _transactional_.  This means that a query scanning
a pool sees its entire data scan as a fixed "snapshot" with respect to the
commit history.  In fact, the Zed language includes an `at` specification that
can be used to specify a commit ID or commit journal position from which to
query.

```
zed lake query -z 'from logs at 1tRxi7zjT7oKxCBwwZ0rbaiLRxb | count() by field'
```

In this way, a query can time-travel through the journal.  As long as the
underlying data has not been deleted, arbitrarily old snapshots of the Zed
lake can be easily queried.

If a writer commits data after a reader starts scanning, then the reader
does not see the new data since it's scanning the snapshot that existed
before these new writes occur.

Alternatively, a reader can scan a specific set of commits by enumerating
the commit tags in the scan/search API.

Also, arbitrary metadata can be committed to the log as described below,
e.g., to associate index objects or derived analytics to a specific
journal commit point potentially across different data pools in
a transactionally consistent fashion.

#### Data Segmentation

In an `add` operation, a commit is broken out into units called _data objects_
where a target objet size is configured into the pool,
typically 100-500MB.  The records within each object are sorted by the pool key(s).
A data object is presumed by the implementation
to fit into the memory of an intake worker node
so that such a sort can be trivially accomplished.

Data added to a pool can arrive in any order with respect to the pool key.
While each object is sorted before it is written,
the collection of objects is generally not sorted.

### Merge

To support _sorted scans_,
data from overlapping objects is read in parallel and merged in sorted order.

However, if many overlapping data objects arise, merging the scan in this fashion
on every read can be inefficient.
This can arise when
many random data `load` operations involving perhaps "late" data
(i.e., the pool key is `ts` and records with old `ts` values regularly
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

Continuing the `git` metaphor, the `merge` command (implementation tracked via [zed/2537](https://github.com/brimdata/zed/issues/2537))
is like a "squash" and performs the LSM-like compaction function, e.g.,
```
zed lake merge -p logs <tag>
(merged commit <tag> printed to stdout)
```
After the merge, all of the objects comprising the new commit are sorted
and non-overlapping.
Here, the objects from the given commit tag are read and compacted into
a new commit as an `add` operation.  Again, until the data is actually committed,
no readers will see any change.

Additionally, multiple commits can be merged all at once to sort all of the
objects of all of the commits that comprise the group, e.g.,
```
zed lake merge -p logs <tag-1> <tag-2> <tag-3>
(merged commit <tag> printed to stdout)
```

### Squash and Delete

After the merge phase, we have a new commit that combines the old commits
across non-overlapping objects, but they are not yet committed.
To avoid consistency issues here, the old commits need
to be deleted while simultaneously adding the new commit.

This can be done automatically by performing a merge, staging the deletes,
then committing the merge and delete together:
```
zed lake merge -p logs <tag-1> <tag-2> <tag-3>
(merged commit <merge-tag> printed to stdout)
zed lake delete -p logs <tag-1> <tag-2> <tag-3>
(delete commit <del-tag> printed to stdout)
zed lake commit -p logs <merge-tag> <del-tag>
```
Note that the data in commits `<tag-1>`, `<tag-2>`, and `<tag-3>` remains
in the pool and scans can be performed on older snapshots of the pool
as long as the data is not deleted.

When multiple commits are given to commit, they are automatically squashed
into a new commit.  The old message fields are lost and must be replaced
by a new message.  Since this is typically driven with automation
we do not yet have an edit cycle like `git commit` offers to merge squashed
commit messages into the new messages via an editor.

A squash may be separately executed without a journal commit using the
`zed lake squash` command, e.g.,
```
zed lake squash -p logs <tag-1> <tag-2> <tag-3>
(merged commit <squash-tag> printed to stdout)
zed lake delete -p logs <tag-1> <tag-2> <tag-3>
(delete commit <del-tag> printed to stdout)
zed lake commit -p logs <squash-tag> <del-tag>
```

Note that delete can be used separately from squash to delete entire commits
or individual data objects at any time.  This is handy when importing data by
mistake:
```
zed lake load -p logs oops.ndjson
(commit <tag> etc. printed to stdout)
zed lake delete -p logs -commit <tag>
```
In this case, the data will be deleted from any subsequent scans but still
exists in the lake and can be accessed via time travel.  Here, we used the
`-commit` flag on `delete` to automatically commit the delete operation to the
commit journal without having to run an explicit `commit` command.

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

### Log

Like `git log`, the command `zed lake log` prints the journal of commit
operations.

The journal represents the entire history of the lake.  Each entry contains
an action:

* `Add` to add a data object reference to a pool,
* `Delete` to delete a data object reference from a pool,
* `AddIndex` to bind an index object to a data object to prune the data object
from a scan when possible using the index,
* `DeleteIndex` to remove an index object reference to its data object, and
* `CommitMessage` for providing metadata about each commit.

The actions are not grouped directly by their commit tag but instead each
action embeds the KSUID of its commit tag.

Note that indexing of data objects is performed in a transactionally-consistent
fashion by including index operations in the commit journal.

By default, `zed lake log` outputs an abbreviated form of the log as text to
stdout, similar to the output of `git log`.

However, the log represents the definitive record of a pool's present
and historical content, and accessing its complete detail can provide
insights about data layout, provenance, history, and so forth.  Thus,
Zed lake provides a means to query a pool's entire journal in all its
detail.  To do so, simply query a pool's journal by referring to
the special sub-pool name `<pool>:journal`.

For example, to aggregate a count of each journal entry type of the pool
called `logs`, you can simply say:
```
zed lake query "from logs:journal | count() by typeof(this)"
```
Since the Zed system "typedefs" each journal record with a named type,
this kind of query gives intuitive results.  There is no need to implement
a long list of features for journal introspection since the data in its entirety
can be simply and efficiently processed as a ZNG stream.

> Note that `:journal` sub-pools are not yet implemented
> ([zed/2787](https://github.com/brimdata/zed/issues/2787)) but the
> `zed lake log` command is implemented and can provide a complete journal
> snapshot.

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
has been index for field "foo" and the query
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
to compute partial aggregations

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
This rule can then be applied to an data object having a given `<tag>`
in a pool, e.g.,
```
zed lake index apply -p logs IndexGroupEx <tag>
```
The index is created and a transaction put in staging.  Once this transaction
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
(primary, second, etc) typically the result of an aggregation.
The aggregation is specified as a Zed query.
For example,
```
zed lake index create IndexGroupEx agg "count() by _path"
```
creates a rule that creates an index keyed by the group-by keys whose
values are the partial-result aggregation given by the Zed expression.

> This is not yet implemented.  The query planner would replace any full object
> scan with the needed aggregation with the result given in the index.
> Where a filter is applied to match one row of the index, that result could be
> likewise and extracted instead of scanning the entire object.
> This capability is not generally useful for interactive search and analytics
> (except for optimizations that suit the interactive app) but rather is a powerful
> capability for application-specific workflows that know the pre-computed
> aggregations that they will use ahead of time, e.g., beacon analysis
> of zeek logs.

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
which is referred to below as `<tag>`.

These identifiers are [KSUIDs](https://github.com/segmentio/ksuid).
The KSUID allocation scheme
provides a decentralized solution for creating globally unique IDs.
KSUIDs have embedded timestamps so the creation time of
any object named in this way can be derived.  Also, a simple lexicographic
sort of the KSUIDs results in a creation-time ordering (though this ordering
is not relied on for causal relationships since clock skew can violate
such an assumption).

Data objects are referred to by zero or more commits, where the commits
are maintained in a commit journal described below.

> While a Zed lake is defined in terms of a cloud object store, it may also
> be realized on top of a file system, which provides a convenient means for
> local, small-scale deployments for test/debug workflows.  Thus, for simple use cases,
> the complexity of running an object-store service may be avoided.

#### Data Objects

An immutable object is created by a single writer using a globally unique name
with an embedded KSUID.  
New objects are written in their entirety.  No updates, appends, or modifications
may be made once an object exists.  Given these semantics, any such object may be
trivially cached as its name or content never changes.

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
|column data|`<pool-tag>/data/<tag>.zst`|
|row data|`<pool-tag>/data/<tag>.zng`|
|row seek index|`<pool-tag>/data/<tag>-seek.zng`|
|search index|`<pool-tag>/index/<tag>-<index-tag>.zng`|

`<tag>` is the KSUID of the data object.
`<index-tag>` is the KSUID of an index object created according to the
index rules described above.  Every index object is defined
with respect to a data object.

The seek index maps pool key values to seek offset in the ZNG file thereby
allowing a scan to do a partial GET of the ZNG object when scanning only
a subset of data.

> Note the ZST format will have seekable checkpoints based on the sort key that
> are encoded into its metadata section so there is no need to have a separate
> seek index for the columnar object.

### Transaction Journal

State that is mutable is built upon a transaction journal of immutable
collections of entries.  In this way, there are no objects in the
storage footprint that are ever modified.  Instead, the journal captures
changes and journal snapshots are used to provide synchronization points
for efficient access to the journal (so the entire journal need not be
read to create the current state) and old journal entries may be removed
based on retention policy.

#### Commit Journal

The pool's commit journal is the definitive record of the evolution of data in
that pool in a transactionally consistent fashion.

Each journal entry is identified with its `journal ID`,
a 64-bit, unsigned integer that begins at 0.
The journal may be updated concurrently by multiple writers so concurrency
controls are included (see [Journal Concurrency Control](#journal-concurrency-control)
below) to provide atomic updates.

A journal entry simply contains actions that modify the "state" of the pool
as described in the `zed lake commit` section above.
Each 'Add' entry includes metadata about the object committed to the pool,
including its pool-key range and commit timestamp.
Thus, data objects and journal entries can be purged with _either_ key-based
or time-based retention policies (or both).

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

A data scan may then be assembled at any point in the journal's history
by scanning, in key order, the objects that comprise all of the commits while merging
records from overlapping objects.  The snapshot is sorted by its pool key
range, where key-range values are sorted by the beginning key as the primary key
and the ending key is the secondary key.

For efficiency, a journal entry's snapshot may be stored as a "cached snapshot"
alongside the journal entry.  This way, the snapshot at HEAD may be
efficiently computed by locating the most recent cached snapshot and scanning
forward to HEAD.

#### Scaling a Journal

When the size of the snapshot file reaches a certain size (and thus becomes too large to
conveniently handle in memory), the journal is converted to an internal sub-pool
called the "journal pool".  The journal pool's
pool key is the "from" value (of its parent pool key) from each commit.
In this case, commits to the parent pool are made in the same fashion,
but instead of snapshotting updates into a snapshot ZNG file,
the snapshots are committed to the journal sub-pool.  In this way, commit histories
can be rolled up and organized by the pool key.  Likewise, retention policies
based on the pool key can remove not just data objects from the main pool but
also data objects in the journal pool comprising committed data that falls
outside of the retention boundary.

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
system and such round trips can be 10s of milliseconds, another approach
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
  R/
    HEAD
    TAIL
    1.zng
    2.zng
    ...
  <pool-tag-1>/
    J/
      HEAD
      TAIL
      1.zng
      2.zng
      ...
      20.zng
      20-snap.zng
      20-seek.zng
      21.zng
      ...
    D/
      <tag1>.{zng,zst}
      <tag2>.{zng,zst}
      ...
    index/
      <tag1>-<index-tag-1>.zng
      <tag1>-<index-tag-2>.zng
      ...
      <tag2>-<index-tag-1>.zng
      ...
  <pool-tag-2>/
  ...
```

## Continuous Ingest

While the description above is very batch oriented, the Zed lake design is
intended to perform scalably for continuous streaming applications.  In this
approach, many small commits may be continuously executed as data arrives and
after each commit, the data is immediately readable.

To handle this use case, the _journal_ of commits is designed
to scale to arbitrarily large footprints as described earlier.

## Lake Introspection

Commit history, metadata about data objects, lake and pool configuration,
etc. can all be queried and
returned as Zed data, which in turn, can be fed into Zed analytics.
This allows a very powerful approach to introspecting the structure of a
lake making it easy to measure, tune, and adjust lake parameters to
optimize layout for performance.

> TBD: define model for scanning metadata in this fashion.  It might be as
> easy as scanning virtual sub-pools that conform to the different types of
> metadata related to a pool, e.g., logs.$journal, logs.$indexes, etc.

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
* transaction journal incomplete (single writer only initially)
* no recursive journal pool
* no columnar support
* pool-key sorted scans only
* no keyless intake
