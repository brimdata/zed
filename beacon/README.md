# Beacon Analysis on the Zng Data Lake

The zng data lake (ZDL) provides a flexible and powerful foundation for security
analytics.  In addition to the zeek- and suricata-specific UX in the brim app,
ZDL can be used to analyze security aspects of network and endpoint data along with
cloud telemetry data to provide insights into beaconing, dns infiltration, and
other related security concepts.

> This directory contains working notes on how ZDL solves beaconing and related
> analytics challenges.  We will migrate the concepts worked out here into
> reusable building blocks integrated into zq, zdl, the brim app, etc.

## RITA

The mechanisms here are inspired by the techniques used in
[RITA](https://github.com/activecm/rita).

RITA is a go program structured as a command-line tool that parses zeek logs,
performs initial analyses on the parsed logs, and places these results in mongodb.
It then provides
a means to query the mongodb to look for various security-related features present
in the original zeek logs, e.g., dns exfiltration, peculiar relations between
top-level DNS names and high-volume queries within the domain, peculiar connections,
beaconing patterns, and so forth.


## The ZDL Approach

The imperative nature of the RITA framework (i.e., the algorithm is realized
as a sequence of actions taken on tables in mongo) leads to a fairly complex
implementation mixing together parsing of JSON and Zeek TSV log lines with
analytics some of which live in native Go code and some of which lives in MongoDB.

The ZDL approach, on the other hand, can be described
in declarative terms and none of the underlying plumbing needs to be addressed
as it's all taken care of by zq.
This simplifies the description of the analytics.
Additionally, as you will see, the fact zng streams are based on heterogeneous
streams of records and need not conform to a single relational schema (much as
mongo is based on semistructured documents instead of pre-defined schemas)
further simplifies the declarative approach.

The idea here is to re-implement the concepts
from RITA in a completely different way using the declarative approach
of zql.

In this approach, the zeek log data is converted to zng and imported into a
zng data lake.  Then, various declarative queries operate over windowed
passes over the raw data creating intermediate analytics that are, in turn,
stored in the lake.  These results can then be further queried to create
the beacon tables or queried directly in the threat hunting workflow,
just as RITA's mongo tables can be queried in various ways in security workflows.

We will call these partial results "summaries".
The data here is simply stored as zng (or zst columnar) files or cloud objects,
in accordance with the zdl data naming conventions.

> There are different ways using zq and zdl to organize and store summaries,
> for example, you can take the output of a "zdl map" command that builds summary
> and pipe it a big zng file and put it wherever you like or import it to another
> lake using 'zdl import'.
>
> Note 'zdl import' and the 'zdl map' scanning model currently presumes that the
> `ts` field is the primary key for partitioning data in a lake so care must be
> taken to preserve the time into and across group-by results.  Since these
> summaries represent analysis across time, it would make sense to store the
> time spans in the output records and have a way to automatically append these
> spans to group-by result.

Taking this approach, at a high level, the model here is as follows:
* a _connection summary_ is created from a zql group-by keyed on both id.orig_h and
id.resp_h;
* a _host summary_ is created from a zql query merging a group-by keyed on `id.orig_h` with
another group-by keyed on `id.resp_h`;
* a _domain summary_ is created from a zql group-by over the all the query strings in DNS logs plus
all the domain names obtained from unwinding the subdomains of each query string,
* a _hostname summary_ is created from a group-by also keyed on DNS host names (without
the subdomains) that aggregates additional information;
* a _user agent summary_ dresults from a zql group-by keyed by the user-agent
string in http logs and the ja3 hash in ssl logs;
* a _cert summary results from a zql group-by keyed on server IP of ssl logs
with cerr validation issues aggregating information about that server; and,
* a _beacon summary_ results from a group-by keyed by edge ID that aggregates
various summary metrics performed over a sequence one or more other
connection summaries.

Okay, this all makes sense but the devil's in the details.  There are lots of
things going on in the RITA analyses so let's see how all these details maybe
onto specific zdl mechanism and what how we might need to extend the system
to provide a comprehensive solution for this use case.

### The Connection Summary

The connection summary involves a table indexed by host pairs (i.e., an edge
in the network graph), where the edges are directed (i.e, `<a,b>` and `<b,a>`
are  different) and the edges have a connection count associated with each of them.

This turn out to be really easy.  We just say:
```
count() by id.resp_h,id.orig_h
```

For example, assuming a chunk of relevant logs in a file called
`logs.zng`, this query will show the first five lines of the connection
summary in tabular form:
```
zq -f table "count() by id.resp_h,id.orig_h | head 5" logs.zng
```
which might look like
```
ID.ORIG_H       ID.RESP_H      COUNT
71.217.167.178  192.168.0.54   1
105.237.220.218 192.168.0.54   1
192.168.0.53    145.151.34.146 1
192.168.0.51    166.78.45.63   2
67.71.27.157    192.168.0.54   1
```
However, in a typical large network there are going to be too many unique rows
in this table to fit into the memory of the machine running this zql query.

In zq, when the group-by processor hits a memory limit, it simply spills
the partial results sorted-by-key to disk just like hadoop and spark,
and merges the results at the end of the input scan.  Nothing fancy or new here,
but it gets the job done.

Since the spilled files are just zng streams and since the internal representation
of the zq analytics is also just zng, the implementation is quite straightforward
and simple.  You can even run zq on a spilled temp file, runnging on-the-fly
zql searches and analytics on its content.

> We're working on a distributed approach to group-by where we continuously
> shuffle rows in an asynchronous and adaptive fashion without the need
> of a central job scheduler typically used in a data warehouse, in spark,
> and so forth.

But hold on here, we're brushing under the rug is that these group-by
queries could be processing all of the underlying zeek logs that have been
ingested into the time region being scanned for analysis, i.e., the query
really going on was
```
* | count(),... by id.resp_h,id.orig_h
```
and we're relying on the fact that zq will ignore any records that do not
have all of the group-by keys.  And it just so happens that the zeek conn and
ssl logs both have a field called `id`, so this works beautifully because
the connection summary depends only on these two log sources.

Given this, we can speed things
up a bit for processing only those logs like this:
```
_path=conn OR _path=ssl | count() by id.resp_h,id.orig_h
```
Now you might notice a problem.  Since both ssl logs and conn logs pass through
the same group-by arrangement and since there is a conn record that corresponds
to each ssl connection, we are double counting ssl connections in the count()
aggregator.

Turns out there's fix: just count the conn logs:
```
_path=conn OR _path=ssl | count() where _path=conn,<other-aggregations> by id.resp_h,id.orig_h
```
Now we can have aggregations that work either on conn logs or ssl logs
and do not mess up our connection count.  The beauty of the zng data model here
is that aggregations on values that do not exist in a given input record simply
ignore that value and continue along.  In SQL, you would have to project columns
from separate tables, join them, then do the group-by aggregation.

Anyway, back the connection summary.
Remember that each edge in the connection summary represents one _or more_
connections that occurred for that directed pair.  It is this little time
series of connection start times and sizes that will be analyzed to create
the beacon score.

We already have the connection count, the remaining fields are:
* `bytes` a list of integers representing the amount of bytes transferred in
each separate connection from the originator to the
responder (i.e., typically the data transferred out of
a compromised host to an external c2 server),
* `ts` is a list of 64-bit timestamps of each separate connection start time
* `tuples` a list of unique strings of the form "port:proto:service"
where these values come from the zeek conn log fields of the same name,
* `icerts` is a boolean indicating whether there were any SSL validation
issues between the host pair during the scanned time period according to the zeek
ssl logs,
* `maxdur` is the time span of the longest-lived connection,
* `tbytes` is the sum of all transport-payload bytes transmitted in both
directions according to zeek conn logs, and
* `tdur` is the sum of the durations of all the connections between the pair.

Let's now look at how we would enhance the zql query for the edge graph
from above to capture all of these fields.

#### _bytes_

The `bytes` field is interesting because it consists of a _list_ of byte counts
from each connection that occurred between the pair.  We can't just use a
sum aggregator.  Instead we use the zql 'collect' aggregator, which aggregates
each value into an array, e.g.,
```
... | count(),bytes=collect(orig_bytes) by id.resp_h,id.orig_h
```

#### _ts_

The _ts_ field is similar to _bytes_ but is the connection start time instead of
the origin bytes for each connection and only unique timestamps are stored.
This sounds like a "set" type, which zng happens to have, as sets store just
one value of each item in the set.   This set can be computed with the zql
`union` aggregator, e.g.,
```
... | count(),bytes=collect(orig_bytes),ts_set=union(ts) by id.resp_h,id.orig_h
```
Not we call this aggregation `ts_set` so as to not confuse it with the `ts`
field of the underlying logs:

#### _tuples_

Like `ts, the `tuples` field is also a list of unique values, i.e., a set,
where each value is the three-tuple formed from the zeek conn log fields
`id.resp_p`, `proto`, and `service`.

It is worth elaborating on the types of these values here.
`id.resp_p` is a zeek "port" type, `proto` is a zeek "enum" type, and `service`
is a string.  Because we use network data often with zng, it's nice to know
when something is a port as compared to a regular integer.  And because zng
supports type aliases (roughly analogous to logical types in avro and parquet),
zq creates an alias for "port" and maps zeek port values to this aliased type
when ingesting zeek data into a zng data lake.

Likewise, for zeek enums, while zng supports a bona fide enum type, the zeek
log format does not describe all of the elements in its TSV `#types` header
so it's not possible to create
an enum typedef without scanning all of the logs to determine the set of
enum elements.  Instead, zq creates a type alias called "zenum" for zeek enums
that maps  to "string" and treats zeek enum values as a string equivalent to
its enum element identifier.

> Note that because we employ these mappings, any time we extract and export
> zeek data in the zeek log format, we can restore the original zeek types.

Okay, back to the `tuples` field.  With the union operator,
we simply need a way to treat the input to union as a value tuple comprised
of the aforementioned three elements.  Zng however does not have a tuple
data type (they are not columnar friendly), but zql does have a way to
cut fields from a record to form another record value so we can compose the
tuple into a record on-the-fly and have the union aggregator apply to
the resulting sequence of record values.

Armed with the `union` aggregator and the `cut` funtion,
we can now extend our connection-summary query to handle the `tuples` field:
```
count(),...,tuples=union(cut(id.orig_p,proto,service)) by id.resp_h,id.orig_h
```
where we have elided the other aggregations to save space here.

#### _icerts_

The `icerts` field is simply a boolean that indicates whether there was an
issue validating an SSL certificate during a handshake on _any_ connection
for each edge pair (i.e., an invalid cert).
RITA computes the boolean by noting when the zeek
field `validation_status` from the ssl log [is not one of "ok", null,
empty string, or space](https://github.com/activecm/rita/blob/c4ae2f7d010b2b4477713affda405dd63d208db3/parser/fsimporter.go#L680)
(these variations in "ok" status seem strange but, of course, zeek faithfully
reports these odd variations based on what went over the wire
from real-world SSL implementations).

So, we just need a way to compute the boolean and "OR" together all these
boolean values across the connection of a given pair.  Yet again, we have
another aggregation, and it looks this:
```
... | ...,icert=or(!(validation_status="ok" OR validation_status=" " OR
                     validation_status!="" OR validation_status != null)) by id.resp_h,id.orig_h
```
Note that when encountering a conn log, this `or` aggregator would simply
ignore the entire record since the `validation_status` field is not present.

#### _maxdur_

The `maxdur` field is the time span of the longest-lived connection between the
hosts in the edge pair, i.e.,
```
... | ...,maxdur=max(duration) by id.resp_h,id.orig_h
```

#### _tbytes_

The `tbytes` field is the sum of all transport-payload bytes transmitted in both
directions according to zeek conn logs, i.e.,
```
... | ...,tbytes=sum(orig_bytes+resp_bytes) by id.resp_h,id.orig_h
```

#### _tdur_

The `tdur` field is the sum of the durations of each connection, i.e.,
```
... | ...,tdur=sum(duration) by id.resp_h,id.orig_h
```

#### Putting it all together

Combining all of the above concepts,
here the final zql expression to construct a connection summary:
```
_path=conn OR _path=ssl |
   count(_path=conn),
   bytes=collect(orig_bytes),
   ts_set=union(ts) where _path=conn,
   tuples=union(cut(id.orig_p,proto,service)),
   icert=or(!(validation_status="ok" OR validation_status=" " OR
              validation_status="" OR validation_status=null)),
   maxdur=max(duration),
   tbytes=sum(orig_bytes+resp_bytes),
   tdur=sum(duration)
     by id.resp_h,id.orig_h
```

> TBD: compute strobe and filter out ts/bytes for strobes to reduce overhead.
> Could use a new proc that is conditional action, e.g., so you do a conditional
> put rather than fork into two flows and merge.

## The Host Summary Revisted

Now that we have the connection summary, it's straightforward to construct
the host summary by processing the connection summary data, which would
typically be stored adjacent to the underlying log data.  We also need to
process the DNS logs here.

The host summary is keyed by host IP and has the following fields:
* `count_src` the number of instances in which the host IP appears as an originator
in the connection graph,
* `count_dst` the number of instances in which the host IP appears as a responder
in the connection graph,
* `txt_query_count` the number of times this host issued a DNS TXT query, and
* `upps_count` the number of times this host participated as an originator
in an SSL handshake that resulted in an invalid cert response.

First the counts.  All the counts are already stored in the count field
of the the connection summary.  But for each record, we need to add the count
into two counters: one for the `count_src` keyed on `id.orig_h` and one for
`count_dst` keyed on `id.resp_h`.  However, groupby can't do both of these
things at the same time.  So we build two group-by tablea and merge them.
In practice there is little overlap between originators (on the inside of the secure domain)
and responders (on the outside), so little inefficiency arises fromo having
seprate tables.

Here is parallel query that accomplishes this:
```
<from-connection-summary> | (
	count_src=sum(count) by host=id.orig_h | sort host ;
	count_dst=sum(count) by host=id.resp_h | sort host
      ) | join host
```
> TBD: we need zql syntax for join that creates an orderedmerge join
> <bikeshed>
> let's change the name of orderedmerge in the code
> </bikeshed>

For the `txt_query_count`, we need to scan the raw DNS logs but the query
is easy:
```
qtype_name="TXT" | count() by id.orig_h
```
This summary table could be stored adjacent to the counts summary, or
it could be mixed in like this:
```
( <from-connection-summary> | (
	count_src=sum(count) by host=id.orig_h | sort host ;
	count_dst=sum(count) by host=id.resp_h | sort host
      ) | join host ;
   _path=dns qtype_name="TXT" | count() by host=id.orig_hp | sort host
) | join host
```
> BTW, noting here that the groupby spill code will already have sorted the
> data so no use sorting again, and when you spill groupby you'll have to
> spill sort.  We should fix this.  Maybe the easiest thing to do is make sorting
> an option to groupby (not always-on as we discussed in the past), then
> we could turn on the option when compiling the flowgraph if we see the
> output entering an order-dependent downstream element like join (i.e., mergejoin),
> and/or also make it a user-settable flag on groupby.

`upps_count` is easy to compute
```
upps_count=count(icert=true) by id.orig_h
```
and can be folded into the first leg of the first parallel flow graph
as follows:
```
( <from-connection-summary> | (
	count_src=sum(count),upps_count=count(icert=true) by host=id.orig_h | sort host ;
	count_dst=sum(count) by host=id.resp_h | sort host
      ) | join host ;
   _path=dns qtype_name="TXT" | count() by host=id.orig_hp | sort host
) | join host
```

### The Domain Summary

The domain summary is easy to compute from the query strings:
```
_path=dns | count() by query
```
However, we also want to summarize the subdomains of each domain so
we need a way to split the query string by "." and update the table
with each subdomain prefix, but not the top-level domain.  

> TBD: This logic is a little bit complex for data flow so let's create some
> helper functions.  `split()`` will split a string into an array so we
> can convert "a.b.c.com" into ["a", "b", "c", "com"].  From, we want to build
> ["a.b.c.com", "b.c.com", "c.com", ".com" ] then take the first three.
> So, let's create a splay() string function that does the string join
> operator and a have a slice

Armed with split, we can say
```
_path=dns | split -sep "." query | count() by query
```

### The Host Name Summary

> TBD

### The User Agent Summary

> TBD

### The Cert Summary

> TBD

### The Beacon Summary

> TBD

## Streaming model

> TBD

Streaming model... lots of joins can be done in parallel using `zar map`
and ... no need to loop through tables and do mongo lookups on each
search key.  Instead we think in terms of dataflow and map-reduce style
aggregations and analytics.
