---
sidebar_position: 2
sidebar_label: zed
---

# zed

> **TL;DR** `zed` is a command-line tool to manage and query Zed data lakes.
> You can import data from a variety of formats and `zed` will automatically
> commit the data in the Zed data model's [super-structured](../formats/README.md)
> format, providing full fidelity of the original format and the ability
> to reconstruct the original data without loss of information.
>
> Zed lakes provide an easy-to-use substrate for data discovery, preparation,
> and transformation as well as serving as a queryable and searchable store
> for super-structured data both for online and archive use cases.

<p id="status"></p>

:::tip Status
While [`zq`](zq.md) and the [Zed formats](../formats/README.md)
are production quality, the Zed lake is still fairly early in development
and alpha quality.
That said, Zed lakes can be utilized quite effectively at small scale,
or at larger scales when scripted automation
is deployed to manage the lake's data layout via the
[lake API](../lake/api.md).

Enhanced scalability with self-tuning configuration is under development.
:::

## The Lake Model

A Zed lake is a cloud-native arrangement of data, optimized for search,
analytics, ETL, data discovery, and data preparation
at scale based on data represented in accordance
with the [Zed data model](../formats/zed.md).

A lake is organized into a collection of data pools forming a single
administrative domain.  The current implementation supports
ACID append and delete semantics at the commit level while
we have plans to support CRUD updates at the primary-key level
in the near future.

The semantics of a Zed lake loosely follows the nomenclature and
design patterns of [`git`](https://git-scm.com/).  In this approach,
* a _lake_ is like a GitHub organization,
* a _pool_ is like a `git` repository,
* a _branch_ of a _pool_ is like a `git` branch,
* the _use_  command is like a `git checkout`, and
* the _load_ command is like a `git add/commit/push`.

A core theme of the Zed lake design is _ergonomics_.  Given the Git metaphor,
our goal here is that the Zed lake tooling be as easy and familiar as Git is
to a technical user.

Since Zed lakes are built around the Zed data model,
getting different kinds of data into and out of a lake is easy.
There is no need to define schemas or tables and then fit
semi-structured data into schemas before loading data into a lake.
And because Zed supports a large family of formats and the load endpoint
automatically detects most formats, it's easy to just load data into a lake
without thinking about how to convert it into the right format.

### CLI-First Approach

The Zed project has taken a _CLI-first approach_ to designing and implementing
the system.  Any time a new piece of functionality is added to the lake,
it is first implemented as a `zed` command.  This is particularly convenient
for testing and continuous integration as well as providing intuitive,
bite-sized chunks for learning how the system works and how the different
components come together.

While the CLI-first approach provides these benefits,
all of the functionality is also exposed through [an API](../lake/api.md) to
a Zed service.  Many use cases involve an application like
[Zui](https://zui.brimdata.io/) or a
programming environment like Python/Pandas interacting
with the service API in place of direct use with the `zed` command.

### Storage Layer

The Zed lake storage model is designed to leverage modern cloud object stores
and separates compute from storage.

A lake is entirely defined by a collection of cloud objects stored
at a configured object-key prefix.  This prefix is called the _storage path_.
All of the meta-data describing the data pools, branches, commit history,
and so forth is stored as cloud objects inside of the lake.  There is no need
to set up and manage an auxiliary metadata store.

Data is arranged in a lake as a set of pools, which are comprised of one
or more branches, which consist of a sequence of data commit objects
that point to cloud data objects.

Cloud objects and commits are immutable and named with globally unique IDs,
based on the [KSUIDs](https://github.com/segmentio/ksuid), and many
commands may reference various lake entities by their ID, e.g.,
* _Pool ID_ - the KSUID of a pool
* _Commit object ID_ - the KSUID of a commit object
* _Data object ID_ - the KSUID of a committed data object

Data is added and deleted from the lake only with new commits that
are implemented in a transactionally consistent fashion.  Thus, each
commit object (identified by its globally-unique ID) provides a completely
consistent view of an arbitrarily large amount of committed data
at a specific point in time.

While this commit model may sound heavyweight, excellent live ingest performance
can be achieved by micro-batching commits.

Because the Zed lake represents all state transitions with immutable objects,
the caching of any cloud object (or byte ranges of cloud objects)
is easy and effective since a cached object is never invalid.
This design makes backup/restore, data migration, archive, and
replication easy to support and deploy.

The cloud objects that comprise a lake, e.g., data objects,
commit history, transaction journals, partial aggregations, etc.,
are stored as Zed data, i.e., either as [row-based ZNG](../formats/zng.md)
or [columnar VNG](../formats/vng.md).
This makes introspection of the lake structure straightforward as many key
lake data structures can be queried with metadata queries and presented
to a client as Zed data for further processing by downstream tooling.

Zed's implementation also includes a storage abstraction that maps the cloud object
model onto a file system so that Zed lakes can also be deployed on standard file systems.

### Zed Command Personalities

The `zed` command provides a single command-line interface to Zed lakes, but
different personalities are taken on by `zed` depending on the particular
sub-command executed and the [lake location](#locating-the-lake).

To this end, `zed` can take on one of three personalities:

* _Direct Access_ - When the lake is a storage path (`file` or `s3` URI),
then the `zed` commands (except for `serve`) all operate directly on the
lake located at that path.
* _Client Personality_ - When the lake is an HTTP or HTTPS URL, then the
lake is presumed to be a Zed lake service endpoint and the client
commands are directed to the service managing the lake.
* _Server Personality_ - When the [`zed serve`](#serve) command is executed, then
the personality is always the server personality and the lake must be
a storage path.  This command initiates a continuous server process
that serves client requests for the lake at the configured storage path.

Note that a storage path on the file system may be specified either as
a fully qualified file URI of the form `file://` or be a standard
file system path, relative or absolute, e.g., `/lakes/test`.

Concurrent access to any Zed lake storage, of course, preserves
data consistency.  You can run multiple `zed serve` processes while also
running any `zed` lake command all pointing at the same storage endpoint
and the lake's data footprint will always remain consistent as the endpoints
all adhere to the consistency semantics of the Zed lake.

> One caveat here: data consistency is not fully implemented yet for
> the S3 endpoint so only single-node access to S3 is available right now,
> though support for multi-node access is forthcoming.
> For a shared file system, the close-to-open cache consistency
> semantics of NFS should provide the necessary consistency guarantees needed by
> a Zed lake though this has not been tested.  Multi-process, single-node
> access to a local file system has been thoroughly tested and should be
> deemed reliable, i.e., you can run a direct-access instance of `zed` alongside
> a server instance of `zed` on the same file system and data consistency will
> be maintained.

### Locating the Lake

At times you may want the Zed CLI tools to access the same lake storage
used by other tools such as [Zui](https://zui.brimdata.io/). To help
enable this by default while allowing for separate lake storage when desired,
`zed` checks each of the following in order to attempt to locate an existing
lake.

1. The contents of the `-lake` option (if specified)
2. The contents of the `ZED_LAKE` environment variable (if defined)
3. A Zed lake service running locally at `http://localhost:9867` (if a socket
   is listening at that port)
4. A `zed` subdirectory below a path in the
   [`XDG_DATA_HOME`](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html)
   environment variable (if defined)
5. A default file system location based on detected OS platform:
   - `%LOCALAPPDATA%\zed` on Windows
   - `$HOME/.local/share/zed` on Linux and macOS

### Data Pools

A lake is made up of _data pools_, which are like "collections" in NoSQL
document stores.  Pools may have one or more branches and every pool always
has a branch called `main`.

A pool is created with the [create command](#create)
and a branch of a pool is created with the [branch command](#branch).

A pool name can be any valid UTF-8 string and is allocated a unique ID
when created.  The pool can be referred to by its name or by its ID.
A pool may be renamed but the unique ID is always fixed.

#### Commit Objects

Data is added into a pool in atomic units called _commit objects_.

Each commit object is assigned a global ID.
Similar to Git, Zed commit objects are arranged into a tree and
represent the entire commit history of the lake.

> Technically speaking, Git can merge from multiple parents and thus
Git commits form a directed acyclic graph instead of a tree;
Zed does not currently support multiple parents in the commit object history.

A branch is simply a named pointer to a commit object in the Zed lake
and like a pool, a branch name can be any valid UTF-8 string.
Consistent updates to a branch are made by writing a new commit object that
points to the previous tip of the branch and updating the branch to point at
the new commit object.  This update may be made with a transaction constraint
(e.g., requiring that the previous branch tip is the same as the
commit object's parent); if the constraint is violated, then the transaction
is aborted.

The _working branch_ of a pool may be selected on any command with the `-use` option
or may be persisted across commands with the [use command](#use) so that
`-use` does not have to be specified on each command-line.  For interactive
workflows, the `use` command is convenient but for automated workflows
in scripts, it is good practice to explicitly specify the branch in each
command invocation with the `-use` option.

#### Commitish

Many `zed` commands operate with respect to a commit object.
While commit objects are always referenceable by their commit ID, it is also convenient
to refer to the commit object at the tip of a branch.

The entity that represents either a commit ID or a branch is called a _commitish_.
A commitish is always relative to the pool and has the form:
* `<pool>@<id>` or
* `<pool>@<branch>`

where `<pool>` is a pool name or pool ID, `<id>` is a commit object ID,
and `<branch>` is a branch name.

In particular, the working branch set by the [use command](#use) is a commitish.

A commitish may be abbreviated in several ways where the missing detail is
obtained from the working-branch commitish, e.g.,
* `<pool>` - When just a pool name is given, then the commitish is assumed to be
`<pool>@main`.
* `@<id>` or `<id>`- When an ID is given (optionally with the `@` prefix), then the commitish is assumed to be `<pool>@<id>` where `<pool>` is obtained from the working-branch commitish.
* `@<branch>` - When a branch name is given with the `@` prefix, then the commitish is assumed to be `<pool>@<id>` where `<pool>` is obtained from the working-branch commitish.

An argument to a command that takes a commit object is called a _commitish_
since it can be expressed as a branch or as a commit ID.

#### Pool Key

Each data pool is organized according to its configured _pool key_,
which is the sort key for all data stored in the lake.  Different data pools
can have different pool keys but all of the data in a pool must have the same
pool key.

As pool data is often comprised of Zed records (analogous to JSON objects),
the pool key is typically a field of the stored records.
When pool data is not structured as records/objects (e.g., scalar or arrays or other
non-record types), then the pool key would typically be configured
as the [special value `this`](../language/dataflow-model.md#the-special-value-this).

Data can be efficiently scanned if a query has a filter operating on the pool
key.  For example, on a pool with pool key `ts`, the query `ts == 100`
will be optimized to scan only the data objects where the value `100` could be
present.

> The pool key will also serve as the primary key for the forthcoming
> CRUD semantics.

A pool also has a configured sort order, either ascending or descending
and data is organized in the pool in accordance with this order.
Data scans may be either ascending or descending, and scans that
follow the configured order are generally more efficient than
scans that run in the opposing order.

Scans may also be range-limited but unordered.

Any data loaded into a pool that lacks the pool key is presumed
to have a null value with regard to range scans.  If large amounts
of such "keyless data" are loaded into a pool, the ability to
optimize scans over such data is impaired.

### Time Travel

Because commits are transactional and immutable, a query
sees its entire data scan as a fixed "snapshot" with respect to the
commit history.  In fact, Zed's [from operator](../language/operators/from.md)
allows a commit object to be specified with the `@` suffix to a
pool reference, e.g.,
```
zed query 'from logs@1tRxi7zjT7oKxCBwwZ0rbaiLRxb | ...'
```
In this way, a query can time-travel through the commit history.  As long as the
underlying data has not been deleted, arbitrarily old snapshots of the Zed
lake can be easily queried.

If a writer commits data after and while a reader is scanning, then the reader
does not see the new data since it's scanning the snapshot that existed
before these new writes occurred.

Also, arbitrary metadata can be committed to the log as described below,
e.g., to associate derived analytics to a specific
journal commit point potentially across different data pools in
a transactionally consistent fashion.

While time travel through commit history provides one means to explore
past snapshots of the commit history, another means is to use a timestamp.
Because the entire history of branch updates is stored in a transaction journal
and each entry contains a timestamp, branch references can be easily
navigated by time.  For example, a list of branches of a pool's past
can be created by scanning the internal "pools log" and stopping at the largest
timestamp less than or equal to the desired timestamp.  Then using that
historical snapshot of the pools, a branch can be located within the pool
using that pool's "branches log" in a similar fashion, then its corresponding
commit object can be used to construct the data of that branch at that
past point in time.

 > Note that time travel using timestamps is a forthcoming feature.

## Zed Commands

The `zed` command is structured as a primary command
consisting of a large number of interrelated sub-commands, similar to the
[`docker`](https://docs.docker.com/engine/reference/commandline/cli/)
or [`kubectl`](https://kubernetes.io/docs/reference/generated/kubectl/kubectl-commands)
commands.

The following sections describe each of the available commands and highlight
some key options.  Built-in help shows the commands and their options:

* `zed -h` with no args displays a list of `zed` commands.
* `zed command -h`, where `command` is a sub-command, displays help
for that sub-command.
* `zed command sub-command -h` displays help for a sub-command of a
sub-command and so forth.

### Auth
```
zed auth login|logout|method|verify
```
Access to a Zed lake can be secured with [Auth0 authentication](https://auth0.com/).
Please reach out to us on our [community Slack](https://www.brimdata.io/join-slack/)
if you'd like help setting this up and trying it out.

### Branch
```
zed branch [options] [name]
```
The `branch` command creates a branch with the name `name` that points
to the tip of the working branch or, if the `name` argument is not provided,
lists the existing branches of the selected pool.

For example, this branch command
```
zed branch -use logs@main staging
```
creates a new branch called "staging" in pool "logs", which points to
the same commit object as the "main" branch.  Once created, commits
to the "staging" branch will be added to the commit history without
affecting the "main" branch and each branch can be queried independently
at any time.

Supposing the `main` branch of `logs` was already the working branch,
then you could create the new branch called "staging" by simply saying
```
zed branch staging
```
Likewise, you can delete a branch with `-d`:
```
zed branch -d staging
```
and list the branches as follows:
```
zed branch
```

### Create
```
zed create [-orderby key[,key...][:asc|:desc]] <name>
```
The `create` command creates a new data pool with the given name,
which may be any valid UTF-8 string.

The `-orderby` option indicates the pool key that is used to sort
the data in lake, which may be in ascending or descending order.

If a pool key is not specified, then it defaults to
the [special value `this`](../language/dataflow-model.md#the-special-value-this).

A newly created pool is initialized with a branch called `main`.

> Zed lakes can be used without thinking about branches.  When referencing a pool without
> a branch, the tooling presumes the "main" branch as the default, and everything
> can be done on main without having to think about branching.

### Delete
```
zed delete [options] <id> [<id>...]
zed delete [options] -where <filter>
```
The `delete` command removes one or more data objects indicated by their ID from a pool.
This command
simply removes the data from the branch without actually deleting the
underlying data objects thereby allowing time travel to work in the face
of deletes.  Permanent deletion of underlying data objects is handled by the
separate [`vacuum`](#vacuum) command.

If the `-where` flag is specified, delete will remove all values for which the
provided filter expression is true.  The value provided to `-where` must be a
single filter expression, e.g.:

```
zed delete -where 'ts > 2022-10-05T17:20:00Z and ts < 2022-10-05T17:21:00Z'
```

### Drop
```
zed drop [options] <name>|<id>
```
The `drop` command deletes a pool and all of its constituent data.
As this is a DANGER ZONE command, you must confirm that you want to delete
the pool to proceed.  The `-f` option can be used to force the deletion
without confirmation.

### Init
```
zed init [path]
```
A new lake is initialized with the `init` command.  The `path` argument
is a [storage path](#storage-layer) and is optional.  If not present, the path
is [determined automatically](#locating-the-lake).

If the lake already exists, `init` reports an error and does nothing.

Otherwise, the `init` command writes the initial cloud objects to the
storage path to create a new, empty lake at the specified path.

### Load
```
zed load [options] input [input ...]
```
The `load` command commits new data to a branch of a pool.

Run `zed load -h` for a list of command-line options.

Note that there is no need to define a schema or insert data into
a "table" as all Zed data is _self describing_ and can be queried in a
schema-agnostic fashion.  Data of any _shape_ can be stored in any pool
and arbitrary data _shapes_ can coexist side by side.

As with `zq`,
the [input arguments](zq.md#usage) can be in
any [supported format](zq.md#input-formats) and
the input format is auto-detected if `-i` is not provided.  Likewise,
the inputs may be URLs, in which case, the `load` command streams
the data from a Web server or [S3](../integrations/amazon-s3.md) and into the lake.

When data is loaded, it is broken up into objects of a target size determined
by the pool's `threshold` parameter (which defaults 500MiB but can be configured
when the pool is created).  Each object is sorted by the pool key but
a sequence of objects is not guaranteed to be globally sorted.  When lots
of small or unsorted commits occur, data can be fragmented.  The performance
impact of fragmentation can be eliminated by regularly [compacting](#manage)
pools.

For example, this command
```
zed load sample1.json sample2.zng sample3.zson
```
loads files of varying formats in a single commit to the working branch.

An alternative branch may be specified with a branch reference with the
`-use` option, i.e., `<pool>@<branch>`.  Supposing a branch
called `live` existed, data can be committed into this branch as follows:
```
zed load -use logs@live sample.zng
```
Or, as mentioned above, you can set the default branch for the load command
via `use`:
```
zed use logs@live
zed load sample.zng
```
During a `load` operation, a commit is broken out into units called _data objects_
where a target object size is configured into the pool,
typically 100MB-1GB.  The records within each object are sorted by the pool key.
A data object is presumed by the implementation
to fit into the memory of an intake worker node
so that such a sort can be trivially accomplished.

Data added to a pool can arrive in any order with respect to the pool key.
While each object is sorted before it is written,
the collection of objects is generally not sorted.

Each load operation creates a single commit object, which includes:
* an author and message string,
* a timestamp computed by the server, and
* an optional metadata field of any Zed type expressed as a ZSON value.
This data has the Zed type signature:
```
{
    author: string,
    date: time,
    message: string,
    meta: <any>
}
```
where `<any>` is the type of any optionally attached metadata .
For example, this command sets the `author` and `message` fields:
```
zed load -user user@example.com -message "new version of prod dataset" ...
```
If these fields are not specified, then the Zed system will fill them in
with the user obtained from the session and a message that is descriptive
of the action.

The `date` field here is used by the Zed lake system to do time travel
through the branch and pool history, allowing you to see the state of
branches at any time in their commit history.

Arbitrary metadata expressed as any [ZSON value](../formats/zson.md)
may be attached to a commit via the `-meta` flag.  This allows an application
or user to transactionally commit metadata alongside committed data for any
purpose.  This approach allows external applications to implement arbitrary
data provenance and audit capabilities by embedding custom metadata in the
commit history.

Since commit objects are stored as Zed, the metadata can easily be
queried by running the `log -f zng` to retrieve the log in ZNG format,
for example, and using [`zq`](zq.md) to pull the metadata out
as in:
```
zed log -f zng | zq 'has(meta) | yield {id,meta}' -
```

### Log
```
zed log [options] [commitish]
```
The `log` command, like `git log`, displays a history of the commit objects
starting from any commit, expressed as a [commitish](#commitish).  If no argument is
given, the tip of the working branch is used.

Run `zed log -h` for a list of command-line options.

To understand the log contents, the `load` operation is actually
decomposed into two steps under the covers:
an "add" step stores one or more
new immutable data objects in the lake and a "commit" step
materializes the objects into a branch with an ACID transaction.
This updates the branch pointer to point at a new commit object
referencing the data objects where the new commit object's parent
points at the branch's previous commit object, thus forming a path
through the object tree.

The `log` command prints the commit ID of each commit object in that path
from the current pointer back through history to the first commit object.

A commit object includes
an optional author and message, along with a required timestamp,
that is stored in the commit journal for reference.  These values may
be specified as options to the `load` command, and are also available in the
API for automation.

> Note that the branchlog meta-query source is not yet implemented.

### Manage
```
zed manage [options]
```
The `manage` command performs maintenance tasks on a lake.

Currently the only supported task is _compaction_, which reduces fragmentation
by reading data objects in a pool and writing their contents back to large,
non-overlapping objects.

If the `-monitor` option is specified and the lake is [located](#locating-the-lake)
via network connection, `zed manage` will run continuously and perform updates
as needed.  By default a check is performed once per minute to determine if
updates are necessary.  The `-interval` option may be used to specify an
alternate check frequency in [duration format](../formats/zson.md#23-primitive-values).

If `-monitor` is not specified, a single maintenance pass is performed on the
lake.

The output from `manage` provides a per-pool summary of the maintenance
performed, including a count of `objects_compacted`.

### Merge

Data is merged from one branch into another with the `merge` command, e.g.,
```
zed merge -use logs@updates main
```
where the `updates` branch is being merged into the `main` branch
within the `logs` pool.

A merge operation finds a common ancestor in the commit history then
computes the set of changes needed for the target branch to reflect the
data additions and deletions in the source branch.
While the merge operation is performed, data can still be written concurrently
to both branches and queries performed and everything remains transactionally
consistent.  Newly written data remains in the
branch while all of the data present at merge initiation is merged into the
parent.

This Git-like behavior for a data lake provides a clean solution to
the live ingest problem.
For example, data can be continuously ingested into a branch of main called `live`
and orchestration logic can periodically merge updates from branch `live` to
branch `main`, possibly [compacting](#manage) data after the merge
according to configured policies and logic.

### Query
```
zed query [options] <query>
```
The `query` command runs a Zed program with data from a lake as input.
A query typically begins with a [from operator](../language/operators/from.md)
indicating the pool and branch to use as input.  If `from` is not present, then the
query reads from the working branch.

The pool/branch names are specified with `from` at the beginning of the Zed
query.

As with `zq`, the default output format is ZSON for
terminals and ZNG otherwise, though this can be overridden with
`-f` to specify one of the various supported output formats.

If a pool name is provided to `from` without a branch name, then branch
"main" is assumed.

This example reads every record from the full key range of the `logs` pool
and sends the results to stdout.

```
zed query 'from logs'
```

We can narrow the span of the query by specifying a filter on the pool key:
```
zed query 'from logs | ts >= 2018-03-24T17:36:30.090766Z and ts <= 2018-03-24T17:36:30.090758Z'
```
Filters on pool keys are efficiently implemented as the data is laid out
according to the pool key and seek indexes keyed by the pool key
are computed for each data object.

Lake queries also can refer to HEAD (i.e., the branch context set in the most
recent `use` command) either implicitly by omitting the `from` operator:
```
zed query '*'
```
or by referencing `HEAD`:
```
zed query 'from HEAD'
```

When querying data to the ZNG output format,
output from a pool can be easily piped to other commands like `zq`, e.g.,
```
zed query -f zng 'from logs' | zq -f table 'count() by field' -
```
Of course, it's even more efficient to run the query inside of the pool traversal
like this:
```
zed query -f table 'from logs | count() by field'
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
* `from pool[@<branch>]<:meta>` - branch level

`<meta>` is the name of the metadata being queried. The available metadata
sources vary based on level.

For example, a list of pools with configuration data can be obtained
in the ZSON format as follows:
```
zed query -Z "from :pools"
```
This meta-query produces a list of branches in a pool called `logs`:
```
zed query -Z "from logs:branches"
```
Since this is all just Zed, you can filter the results just like any query,
e.g., to look for particular branch:
```
zed query -Z "from logs:branches | branch.name=='main'"
```

This meta-query produces a list of the data objects in the `live` branch
of pool `logs`:
```
zed query -Z "from logs@live:objects"
```

You can also pretty-print in human-readable form most of the metadata Zed records
using the "lake" format, e.g.,
```
zed query -f lake "from logs@live:objects"
```

The `main` branch is queried by default if an explicit branch is not specified,
e.g.,

```
zed query -f lake "from logs:objects"
```

### Rename
```
zed rename <existing> <new-name>
```
The `rename` command assigns a new name `<new-name>` to an existing
pool `<existing>`, which may be referenced by its ID or its previous name.

### Serve
```
zed serve [options]
```
The `serve` command implements Zed's server personality to service requests
from instances of Zed's client [personality](#zed-command-personalities).
It listens for Zed lake API requests on the interface and port
specified by the `-l` option, executes the requests, and returns results.

The `-log.level` option controls log verbosity.  Available levels, ordered
from most to least verbose, are `debug`, `info` (the default), `warn`,
`error`, `dpanic`, `panic`, and `fatal`.  If the volume of logging output at
the default `info` level seems too excessive for production use, `warn` level
is recommended.

### Use
```
zed use [<commitish>]
```
The `use` command sets the working branch to the indicated commitish.
When run with no argument, it displays the working branch and [lake](#locating-the-lake).

For example,
```
zed use logs
```
provides a "pool-only" commitish that sets the working branch to `logs@main`.

If a `@branch` or commit ID are given without a pool prefix, then the pool of
the commitish previously in use is presumed.  For example, if you are on
`logs@main` then run this command:
```
zed use @test
```
then the working branch is set to `logs@test`.

To specify a branch in another pool, simply prepend
the pool name to the desired branch:
```
zed use otherpool@otherbranch
```
This command stores the working branch in `$HOME/.zed_head`.

### Vacuum
```
zed vacuum [options]
```

The `vacuum` command permanently removes underlying data objects that have
previously been subject to a [`delete`](#delete) operation.  As this is a
DANGER ZONE command, you must confirm that you want to remove
the objects to proceed.  The `-f` option can be used to force removal
without confirmation.  The `-dryrun` option may also be used to see a summary
of how many objects would be removed by a `vacuum` but without removing them.
