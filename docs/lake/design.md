# Zed Lake Design Doc

A Zed lake is a cloud-native arrangement of data,
optimized for search, analytics, ETL, and data discovery
at very large scale based on data represented in accordance
with the [Zed data model](../formats).

## Data Pools

A lake is comprised of _data pools_.  Each data pool is organized
according to its configured _pool key_.  Different data pools can have
different pool keys but all of the data in a pool must have the same
pool key.

The pool key is often a timestamp.  In this case, retention policies
and storage hierarchy decisions can be efficiently associated with
ranges of data over the pool key.

Data can be efficiently accessed via range scans comprised of a
range of values conforming to the pool key.

A lake has a configured sort order, either ascending or descending
and data is organized in the lake in accordance with this order.
Data scans may be either ascending or descending, and scans that
follow the configured order are generally more efficient than
scans that run in the opposing order.

Scans may also be range-limited but unordered.

If data loaded into a pool lacks the pool key, that data is still
imported but is not available to pool-key range scans.  Since it lacks
the pool key, such data is instead organized around its "." value.

> TBD: What is the interface for access non-keyed data?  Should this
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
we will illustrate examples of `zed lake` commands that are under development.
Note that while this CLI-first approach provides an ergonomic way to experiment with
and learn the Zed lake building blocks, all of this functionality is also
exposed through an API to a cloud-based service.  Most interactions between
a user and a Zed lake would be via an application like
[Brim](https://github.com/brimdata/brim) or a
programming environment like Python/Pandas rather than via direct interaction
with `zed lake`.

### New

A new pool is created with
```
zed lake new -p <name> -k <key>[,<key>...]
```
where `<name>` is the name of the pool within the implied lake instance and
`<key>` is the Zed language representation of the pool key, e.g.,
```
zed lake new -p logs -k ts
```
Note that there may be multiple pool keys, where subsequent keys act as the secondary,
tertiary, and so forth sort key.

In all these examples, the lake identity is implied by its path (e.g., an S3 URI
or a file system path) and may be specified by the ZED_LAKE_ROOT environment variable
when running `zed lake` commands on a local host.  In a cloud deployment
or running queries through an application, the lake path is determined by
an authenticated connection to the Zed lake service, which explicitly denotes
the lake name (analagous to how a GitHub user authenticates access to
a named GitHub organization).

### Load

Data is then loaded into a lake with the `load` command, .e.g.,
```
zed lake load -p logs sample.ndjson
```
where `sample.ndjson` contains logs in NDJSON format.  Any supported format
(i.e., CSV, NDJSON, Parquet, ZNG, and ZST) as well multiple files can be used
here, e.g.,
```
zed lake load -p logs sample1.csv sample2.zng sample3.zng
```
Parquet and ZST formats are not auto-detected so you must currently specify
`-i` with these formats, e.g.,
```
zed lake load -p logs -i parquet sample4.parquet
zed lake load -p logs -i zst sample5.zst
```

Note that there is no need to define a schema or insert data into
a "table" as all Zed data is _self describing_ and can be queried in a
schema-agnostic fashion.  Data of any _shape_ can be stored in any pool
and arbitrary data _shapes_ can coexist side by side.

### Scan

Data is read from a pool with the `scan` command.  By default, `scan`
generates sorted output of all of the pool data in the configured key order.
The order can be overridden with `-order asc` or `-order desc`.

A range can be specified with `-from` or `-to`.

The default output format is ZNG though it can be overridden with the various
supported output formats.

This example reads every record from the `logs` pool starting
from time `2020-1-1T12:00`, in ascending order,
sends the  results as ZNG to stdout, then pipes the output to `zq` to count the records:
```
zed lake scan -p logs -from 2020-1-1T12:00 -order asc | zq -z "count()" -
```

### Query

Of course, the example is an inefficient way to count records.  Instead of
reading the records out of the lake and processing them, a better approach
is to push the query into the lake.

The `query` command lets you do this, e.g.,
```
zed lake query -p logs -from 2020-1-1T12:00 -order asc "count()" -
```
Here, the query planner and optimizer can notice that the query is just
counting records and implement the query by mostly reading metadata from
the lake.

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
(commit <tag> printed to stdout)
zed lake commit -p logs <tag>
```
A commit `<tag>` refers to one or more data objects stored in the
data pool.  Genreally speaking, a list of commit tags is simply a shortcut
for the set of object tags that comprise the commits.  Both commit and object
tags are named using the same globally unique allocation of
[KSUIDs](https://github.com/segmentio/ksuid).

After an add operation, the commits are stored in a staging area and the
`zed lake stage` command can be used to inspect, squash, and/or delete
commits from staging before they are merged.

Likewise, you can stack multiple adds and commit them all at once, e.g., ,
```
zed lake add -p logs sample1.json
(<tag-1> printed to stdout)
zed lake add -p logs sample2.parquet
(<tag-2> printed to stdout)
zed lake add -p logs sample3.zng
(<tag-3> printed to stdout)
zed lake commit -p logs <tag-1> <tag-2> <tag-3>
```
The commit command also takes an optional title and message that is stored
in the commit journal for reference.  These messages are carried in
a description record attached to every journal entry, which has a Zed
type signature as follows:
```
{
    Author: string,
    Date: time,
    Description: string,
    Data: <any>
}
```
None of fields is used by the Zed lake system for any purpose
except to provide information about the journal commit to the end user
and/or end application.

#### Transactional Semantitcs

The `commit` operation is _transactional_.  This means that a reader scanning
a pool sees its entire data scan as a fixed "snapshot" with respect to the
commit history.

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

In an `add` operation, a commit is broken out into data units called _segments_
where a target segment size is configured into the pool,
typically 100-500MB.  The records of each segment are sorted by its pool key.
A segment of data is presumed to fit into the memory of an intake worker node
so that such a sort can be trivially accomplished.

Data added to a pool can arrive in any order with respect to the pool key.
While each segment is sorted before it is written,
the collection of segments is generally not sorted in its initial commit.

### Merge

To support _sorted scans_,
data from overlapping segments is read in parallel and merged in sorted order.

However, if many overlapping segments arise, merging the scan in this fashion
on every read can be inefficient.

This can arise when when
many random data `load` operations involving perhaps "late" data
(i.e., the pool key is `ts` and records with old `ts` values regularly
show up and need to be inserted into the past).  The data layout can become
fragmented and less efficient to scan, requiring a scan to merge data
from potentially a large number of different segments.

To solve this problem, the Zed lake design follows the
the [LSM](https://en.wikipedia.org/wiki/Log-structured_merge-tree) design pattern.
Since records in each segment are stored in sorted order, a total order over a collection
of segment (e.g., the collection coming from a specific set of commits)
can be produced by executing a sorted scan and rewriting the results back to the pool
in a new commit.  In addition, the segments comprising the total order
do not overlap.  This is just the basic LSM algorithm at work.

Continuing the `git` metaphor, the `merge` command
is like a "squash" and performs the LSM-like compaction function, e.g.,
```
zed lake merge -p logs <tag>
(merged commit <tag> printed to stdout)
```
After the merge, all of the segments comprising the new commit are sorted
and non-overlapping.
Here, the segments from the given commit tag are read and compacted into
a new commit as an `add` operation.  Again, until the data is actually committed,
no readers will see any change.

Additionally, multiple commits can be merged all at once to sort all of the
segments of all of the commits that comprise the group, e.g.,
```
zed lake merge -p logs <tag-1> <tag-2> <tag-3>
(merged commit <tag> printed to stdout)
```

### Squash

After the merge phase, we have a new commit that combines the old commits
across non-overlapping segments, but they are not yet committed.
To avoid consistency issues here, the old commits need
to be deleted while simultaneously adding the new commit.

This can be done automically
by performing a merge, staging the deletes, then
committing the merge and delete together:
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

Data can be deleted with the DANGER-ZONE command `zed lake purge`.
The commits still appear in the log but scans will fail.

Alternatively, old data can be removed from the system using a safer
command (but still in the DANGER-ZONE), `zed lake vacate`, which moves
the tail of the commit journal forward and removes any data no longer
accessible through the modified commit journal.

An orchestration layer outside of the Zed lake is responsible for defining
policy over
how data is ingested and committed and rolled up.  Depending on the
use case and workflow, we envision that some amount of overlapping segments
would persist at small scale and always be "mixed in" with other overlapping
segments during any key-range scan.

> Note: since this style of data organization follows the LSM pattern,
> how data is rolled up (or not) can control the degree of LSM write
> amplification that occurs for a given workload.  There is an explicit
> tradeoff here between overhead of merging overlapping segments on read
> and LSM write amplification to organize the data to avoid such overlaps.

> Note: we are showing here manual, CLI-driven steps to accomplish these tasks
> but a live data pipeline would automate all of this with orchestration that
> performs these functions via a service API, i.e., the same service API
> used by the CLI operators.

### Log

Like `git log`, the command `zed lake log` prints the journal of commit
operations.

> TBD: define format of this output.  The info record context should be
> displayed along with the adds/drops of each commit tag.
> For now, we output the ZNG form of the journal and can work out
> a human-readable form later.

## Cloud Object Naming

The Zed lake semantics defined above are achieved by mapping the
lake abstractions onto a key-value cloud object store.

Every data element in a Zed lake is either of two fundamental object types:
* a single-writer _immutable object_, or
* a multi-writer _mutable object_.

### Immutable Objects

All data in a data pool is comprised of immutable objects, which are organized
into data _segments_.  Each segment is composed of one or more immutable objects
all of which share a common, globally unique identifier,
which refer to below as `<tag>`.

These identifiers are [KSUIDs](https://github.com/segmentio/ksuid).
The KSUID allocation scheme
provides a decentralized solution for creating globally unique IDs.
KSUIDs have embedded timestamps so the creation time of
any object named in this way can be derived.  Also, a simple lexicographic
sort of the KSUIDs results in a creation-time ordering (though this ordering
is not relied on for causal relationships since clock skew can violate
such an assumption).

Segments are referred to by zero or more commits, where the commits
are maintained in a commit journal described below.

> While a Zed lake is defined in terms of a cloud object store, it may also
> be realized on top of a file system, which provides a convenient means for
> local, small-scale deployments or test/debug workflows.  Thus, for simple use cases,
> the complexity of running an object-store service may be avoided.

#### Segments

An immutable object is created by a single writer using a globally unique name
with an embedded KSUID.  
New objects are written in their entirety.  No updates, appends, or modifications
may be made once an object exists.  Given these semantics, any such object may be
trivially cached as its name or content never changes.

Since the object's name is globally unique and the
resulting object is immutable, there is no possible write concurrency to manage.

A segment is comprised of
* one or two data objects (for row and/or column layout),
* a metadata object,
* an optional seek index, and
* zero or more search indexes.

Data objects may be either in row form (i.e., ZNG) or column form (i.e., ZST),
or both forms may be present as a query optimizer may choose to use whatever
representation is more efficient.
When both row and column data objects are present, they must contains the same
underlying Zed data.

Immutable objects are named as follows:

|object type|name|
|-----------|----|
|column data|`<pool-tag>/data/<tag>.zst`|
|row data|`<pool-tag>/data/<tag>.zng`|
|row seek index|`<pool-tag>/<tag>-seek.zng`|
|search index|`<pool-tag>/index/<tag>.zng`|

`<tag>` is the KSUID of the segment.

The seek index maps pool key values to seek offset in the ZNG file thereby
allowing a scan to do a partial GET of the ZNG object when scanning only
a subset of data.

> Note the ZST format will have seekable checkpoints based on the sort key that
> are encoded into its metadata section so there is no need to have a separate
> seek index for the columnar object.

#### Search Indexes

To optimize pool scans, the lake design includes the well-known pruning
concept, where segments of data can be skipped when it can be determined
(either at "compile time" or "run time") that a segment of data is not
needed by a scan, e.g., because a filter predicate would otherwise filter
all of the data in that object.

In addition to standard techniques for pruning a cloud scan (e.g., summary stats
that can determine when a predicate would be false for every record, etc),
search indexes can be built and attached to any data segment using the
`zed lake index` command.

The indexing rules are defined at the lake level and assigned an integer
rule number.  Index objects for each segment are created at some
point of the segment's life cycle and the query system can use the information
in the indexes to prune eligible segments from a scan based on the predicates
present in the query.

While an individual search lookup involves latency to cloud storage to lookup
a key in each index, each lookup is cheap and involves a small amount of data
and the lookups can all be run in parallel, even from a single node, so
the scan schedule can be quickly computed in a small number of round-trips
(that navigate very wide B-trees) to cloud object storage.

### Mutable Objects

Mutable objects are built upon a journal of arbitrary set updates
to sets of one ore more key-value entities stored within a commit journal.

#### Commit Journal

The pool's journal is the definitive record of the evolution of data in
that pool.  At any point in the journal, a snapshot may be available
for efficient, pool-key range scans.

Each journal entry is identified with its `journal ID`,
a 64-bit, unsigned integer that begins at 0.
The journal may be updated concurrently by multiple writers so concurrency
controls are included (see below) to provide atomic updates.

A journal entry simply contains one or more ADD and DROP directives,
which refer to a commit tag and its key range, which in turn, implies
one or more data segments that comprise said commit.  A journal entry
also contains descriptive information as described earlier.

Because each reference commit includes its key range and because the
commit tag embeds a time stamp (indicating date/time of creation),
journal entries can be purged with _either_ key-based or time-based
retention policies (or both).

Each journal is a ZNG file numbered 1 to the end of journal (HEAD),
e.g., `1.zng`, `2.zng`, etc., each number corresponding to a journal ID.
The 0 value is reserved as the null journal ID.
The journal's TAIL begins at 1 and is increased as journal entries are purged.
Entries are added at the HEAD and removed from the TAIL.
Once created, a journal entry is never modified but it may be deleted and
never again allocated.

Each journal entry implies a snapshot of the data in a pool.  A snapshot
is computed by applying the ADD/DROP directives in sequence from entry TAIL to
the journal entry in question, up to HEAD.  This gives the set of commit tags
that comprise a snapshot.  A snapshot may then be scanned by scanning,
in key order, the segments that comprise all of the commits while merging
records from overlapping segments.  The snapshot is sorted by its pool key
range, where key-range values are sorted by the beginning key as the primary key
and the ending key is the secondary key.

For efficiency, a journal entry's snapshot may be stored as "cached snapshot"
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
based on the pool key can remove not just data segments from the main pool but
also data segments in the journal pool comprising commit segment data that falls
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

Second, strong read/writer ordering semantics (as exists in Amazon S3)
can be used to implement transactional journal updates as follows:
* _TBD: this is worked out but needs to be written up_

Finally, since the above algorithm requires many round trips to the storage
system and such round trips can be 10s of milliseconds, another approach
is to simply run a lock service as part of a cloud deployment that manages
a mutex lock for each pool's journal.

### Mutable Objects

Configuration state describing a lake or pool is stored in mutable objects.

Mutable objects can be modified by any writer and are stored in a directory
named after the configuration object.  Here, each entry represents the entire
mutable object as a ZNG file numbered from 0 upward, i.e., `0.zng`, `1.zng`,
and so forth.  The `HEAD` file points at the last and valid object in the sequence.
Once the HEAD has been atomically advanced, the previous mutable object can be deleted
(or the mutable objects can be preserved to keep a history of the configuration changes).

### Object Naming

```
<lake-path>/
  config/
    HEAD
    TAIL
    1.zng
    2.zng
    ...
  <pool-tag-1>/
    config/
      HEAD
      TAIL
      1.zng
      ...
    journal/
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
    data/
      <tag1>.{zng,zst}
      <tag2>.{zng,zst}
      ...
    <index-tag-1>/
      <tag2>.zng
      <tag23>.zng
      <tag101>.zng
    <index-tag-2>/
      <tag77>.zng
      <tag89>.zng
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

Commit history, meta data about segments, segment key spaces,
etc can all be queried and
returned as Zed data, which in turn, can be fed into Zed analytics.
This allows a very powerful approach to introspecting the structure of a
lake making it easy to measure, tune, and adjust lake parameters to
optimize layout for performance.

> TBD: define model for scanning metadata in this fashion.  It might be as
> easy as scanning virtual sub-pools that conform to the different types of
> metadata related to a pool, e.g., logs.$segments, logs.$commits, etc.

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
One approach could be to simply assign the "zero-value" as the pool key.
Or a configured default value could be used.  This would make key-based
retention policies more complicated.

Another approach would be to create a sub-pool on demand when the first
keyless data is encountered, e.g., `pool-name.$nokey` where the pool key
is configured to be ".".  This way, an app or user could query this pool
by this name to scan keyless data.

## Relational Model

Since a Zed lake can provide strong consistency, workflows that manipulate
data in a lake can utilize a model where updates are made to the data
in place.  Such updates involve creating new commits from the old data
where the new data is a modified form of the old data.  This provides
emulation of row-based updates and deletes.

If the pool-key is chosen to be "." for such a use case, then unique
rows can be maintained by trivially detected duplicates (because any
duplicate row will be adjacent when sorted by ".").
so that dups are trivially detected.

Efficient upserts can be accomplished because each segment is sorted by the
pool key.  Thus, an upsert can be sorted then merge-joined with each
overlapping segment.  Where segments produce changes and additions, they can
be forwarded to a streaming add operator and the list of modified segments
accumulated.  At the end of the operation, then new commit(s) along with
the to-be-deleted segments can be added to the journal in a single atomic
operation.  A write conflict occurs if there are any another deletes to
the list of to-be-deleted segments.  When this occurs, the transaction can
simply be restarted.  To avoid inefficiency of many restarts, an upsert can
be partitioned into smaller segments if the use case allows for it.

> TBD finish working through this use case, its requirements, and the
> mechanisms needed to implement it.  Write conflicts will need to be
> managed at a layer above the journal or the journal extended with the
> needed functionality.

## Thoughts on Version 2

Ref counted segments stored outside of a pool.
A global ref-count pool can record all the adjustments so we
transactionally know what data becomes unreachable.
This may or may not be better than mark-and-sweep garbage collection
(where mark and sweep can be done with scalable merge joins).

Copy-on-write semantics so massive pools can be instantaneously copied then
changed/updated under the new pool.

## Next steps...

To get going, we will simplify the above design then create issues
to incrementally add support for each needed area.  In this way, we can
leverage the current code base to get up and running quickly.

The initial prototype will be simplified as follows:
* no transactions, as we will presume a single writer
    * config/meta files stored as single ZNG files
* journal stored as sequence of complete snapshots with HEAD pointing to
most recent entry (HEAD written after update)
* no recursive journal pool
* no columnar support
* pool-key sorted scans only
* no keyless intake

As we work through the `zed lake` command set and API,
we should strive to make the Zed lake CLI-first commands one-to-one
with the cloud API end points when appropriate, i.e., any `zed lake`
command should automatically work with `zed api`, e.g.,
```
zed lake new foo
```
is analagous to
```
zed api lake new foo
```
etc.  That said, not all of the commands will be exposed through the API.
