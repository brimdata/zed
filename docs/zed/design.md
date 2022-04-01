# Zed Lake Design Notes

  * [Cloud Object Architecture](#cloud-object-architecture)
    + [Immutable Objects](#immutable-objects)
      - [Data Objects](#data-objects)
      - [Commit History](#commit-history)
    + [Transaction Journal](#transaction-journal)
      - [Scaling a Journal](#scaling-a-journal)
      - [Journal Concurrency Control](#journal-concurrency-control)
      - [Configuration State](#configuration-state)
    + [Merge on Read](#merge-on-read)
    + [Object Naming](#object-naming)
  * [Continuous Ingest](#continuous-ingest)
  * [Derived Analytics](#derived-analytics)
  * [Keyless Data](#keyless-data)
  * [Relational Model](#relational-model)
  * [Type Rule](#type-rule)
  * [Aggregation Rule](#aggregation-rule)
  * [Vacuum Support](#vacuum-support)

## Cloud Object Architecture

The Zed lake semantics are achieved by mapping the
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
trivially cached as neither its name nor content ever change.

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

The seek index maps pool key values to seek offsets in the ZNG file thereby
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

> TBD: when a scan encounters an object that was physically deleted for
> whatever reason, it should simply continue on and issue a warning on
> the query endpoint "warnings channel".

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

By default, `zed log` outputs an abbreviated form of the log as text to
stdout, similar to the output of `git log`.

However, the log represents the definitive record of a pool's present
and historical content, and accessing its complete detail can provide
insights about data layout, provenance, history, and so forth.  Thus, the
Zed lake provides a means to query a pool's configuration state as well,
thereby allowing past versions of the complete pool and branch configurations
as well as all of their underlying data to be subject to time travel.
To interrogate the underlying transaction history of the branches and
their pointers, simply query a pool's "branchlog" via the syntax `<pool>:branchlog`.

For example, to aggregate a count of each journal entry type of the pool
called `logs`, you can simply say:
```
zed query "from logs:branchlog | count() by typeof(this)"
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

### Merge on Read

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
zed compact <id> [<id> ...]
(merged commit <id> printed to stdout)
```
After compaction, all of the objects comprising the new commit are sorted
and non-overlapping.
Here, the objects from the given commit IDs are read and compacted into
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
>
> Note: we are showing here manual, CLI-driven steps to accomplish these tasks
> but a live data pipeline would automate all of this with orchestration that
> performs these functions via a service API, i.e., the same service API
> used by the CLI operators.

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

## Type Rule

A type rule indicates that all values of any field of a specified type
be indexed where the type signature uses Zed type syntax.
For example,
```
zed index create IndexGroupEx type ip
```
creates a rule that indexes all IP addresses appearing in fields of type `ip`
in the index group `IndexGroupEx`.

## Aggregation Rule

An aggregation rule allows the creation of any index keyed by one or more fields
(primary, second, etc.), typically the result of an aggregation.
The aggregation is specified as a Zed query.
For example,
```
zed index create IndexGroupEx agg "count() by field"
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

## Vacuum Support

While data objects currently can be deleted from a lake, the underlying data
is retained to support time travel.

The system must also support purging of old data so that retention policies
can be implemented.

This could be supported with the DANGER-ZONE command `zed vacuum`
(implementation tracked in [zed/2545](https://github.com/brimdata/zed/issues/2545)).
The commits still appear in the log but scans at any time-travel point
where the commit is present will fail to scan the deleted data.
In this case, perhaps we should emit a structured Zed error describing
the meta-data of the object that was unavailable.

Alternatively, old data can be removed from the system using a safer
command (but still in the DANGER-ZONE), `zed vacate` (also
[zed/2545](https://github.com/brimdata/zed/issues/2545)) which moves
the tail of the commit journal forward and removes any data no longer
accessible through the modified commit journal.
