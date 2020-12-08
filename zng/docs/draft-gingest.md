Generic ingest 
===============

This page contains notes on a new generic ingest approach.


### Background: The existing "json types" system


Currently we have "json types", which works in two steps:
1. Classify an incoming json object (in the current implementation, based on a single field value, such a `_path`)
2. Create a zng record from the incoming json record, of the zng type that is defined for that class.

The classification rules and the target zng types for each class are defined in JSON. For example, the classifier rules for zeek conn and http logs are expressed as:

```json
{"rules": [
    {
      "descriptor": "conn_log",
      "name": "_path",
      "value": "conn"
    },
    {
      "descriptor": "http_log",
      "name": "_path",
      "value": "http"
    }
]}
```
and the JSON-representation of the conn target type is:

```json
{"conn_log":[
    {"name":"_path","type":"string"},
    {"name":"ts","type":"time"},
    {"name":"uid","type":"bstring"},
    {"name":"id","type":[
        {"name":"orig_h","type":"ip"},
        {"name":"orig_p","type":"port"},
        {"name":"resp_h","type":"ip"},
        {"name":"resp_p","type":"port"}
    ]},
    {"name":"proto","type":"zenum"},
    {"name":"service","type":"bstring"},
    {"name":"duration","type":"duration"},
    {"name":"orig_bytes","type":"uint64"},
    {"name":"resp_bytes","type":"uint64"},
    {"name":"conn_state","type":"bstring"},
    {"name":"local_orig","type":"bool"},
    {"name":"local_resp","type":"bool"},
    {"name":"missed_bytes","type":"uint64"},
    {"name":"history","type":"bstring"},
    {"name":"orig_pkts","type":"uint64"},
    {"name":"orig_ip_bytes","type":"uint64"},
    {"name":"resp_pkts","type":"uint64"},
    {"name":"resp_ip_bytes","type":"uint64"},
    {"name":"tunnel_parents","type":"set[bstring]"},
    {"name":"_write_ts","type":"time"}
]}
```

This all is a bit clunky, but works ok for Zeek, where we can generate the types automatically, and where there is (almost) a 1-1 correspondence between `_path` values and record types. 

It doesn't work so well in other settings or other ingest-related use cases. 

Most of all, it is hard-coded to this one narrow workflow, but there are many other processes that one might want to do as part of ingest but that can't be done:

- Enriching incoming records (for example, add `_sourcetype=zeek` )
- Renaming fields (for example, rename `timestamp` to `ts`)
- Validation 
- Hierarchical classification (for example, `suricata.category` then `suricata.alert`)
- Compose schemas (for example, don't repeat `id` in each zeek record type)

to be our general purpose ingest system.


Proposal
--------

The proposal is to use ZQL to write ingest classification, transforms, and validation. This idea was previously floated in sync (by Noah?), but at that time ZQL was far from being able to support ingest-related transforms.

Now, with the recent addition of type values to ZQL ("first class types"), and the forthcoming addition of complex literals (à la ZSON), we're a lot closer. We dont have all the bits yet, but now it's mostly just a matter of filling in a proc or two, and adding one or two flowgraph constructs to zql.

If we can express generic ingest stuff in ZQL, we get a unified language to express and reason both about ingest/ETL and analytics. Similar to ZQL for analytics, ZQL for ingest should strike a nice UX balance between a general-purpose programming language and a hard-coded (JSON) DSL. 

With the changes outlined below, we have the flexibility to cover all the need described above, and if there are missing pieces in the future the extensibility route is to add any requisite proc/functions to ZQL (as we're doing with `reshape` below), where they will also be useful for CLI exploration and post-ingest Brim uses.

Probably the biggest advantage is that users only need to learn one thing, as opposed to having to learn and understand a different (typically ad hoc) DSL for ingest, such as logstash for Elastic or relabelling rules for Prometheus.

Finally, ZQL-based ingest provides a reasonable starting point for query-time transformations. For example, if you've ingested 1PB and now realize you want to change a field name, you change your ZQL ingest config so that newly ingested data will have the new name, and for old data, we can run that same ZQL at query time to convert it. Some of it can also be optimized to avoid reading everything. For example, if you've renamed `_path` to `_zeek_path` and search for `_zeek_path=conn`, we can transform that search to `_path=conn` when looking at old logs. This is all of course not something we need to tackle anytime soon, and there is lots to design here, but doing everything in ZQL is a good basis.


Zeek as driving example
-----------------------

### Define types

```
const zeek_id_t = {orig_h:ip, orig_p:port, resp_h:ip, resp_p:port}
const zeek_conn_t = {_path: string, id: zeek_id_t, uid: string, proto: zenum, ...}
const zeek_http_t = {_path: string, id: zeek_id_t, uid: string, method: bstring, ...}
```


### A new `reshape` proc 

The `reshape` proc (placeholder name) takes a record type value as parameter, and "reshapes" input records to that type.

```
... | reshape(zeek_conn_type) | ...

// (or maybe it should be) 
... | put . = reshape(zeek_conn_type) | ...
```

In order to support input records that may have originated from data languages with unordered fields, `reshape` must be order agnostic. 

For example, `reshape({host:ip, port:port})` should handle both `{host: 1.2.3.4, port: 90}` and `{host: 90, port: 1.2.3.4}`, in the latter case "reordering" the record columns  so that the result matches the type `{host:ip, port:port}`.

### Switched parallel flowgraphs


``` 
... | switch (
    <boolean expression> => ... | ...;
    <boolean expression> => ... | ...;
    ...
    ) | ...
```    

An incoming record is evaluated against the boolean expressions until one matches, and is then "pushed" into the match's RHS flowgraph. 


### Putting both together:

```
const zeek_id_t = {orig_h:ip, orig_p:port, resp_h:ip, resp_p:port}
const zeek_conn_t = {_path: string, id: zeek_conn_t, uid: string, proto: zenum, ...}
const zeek_http_t = {_path: string, id: zeek_conn_t, uid: string, method: bstring, ...}

* | switch {
  _path=conn | reshape(zeek_conn_type) 
  _path=http | reshape(http_conn_type)
  ..
} | ...
```

We've reached parity with "json types" ingest. 

### Variations on reshape:

1. What to do with extra fields: (ja3 example)
  - Include fields & infer their types 
  - Discard records.

2. What to do about missing fields
  - create "tight" descriptor that is subset of reshape descriptor
  - fill in with nulls
  - discard record

3. What to do about leaf type mismatches
   - an int field is "" (Phil's example with netflow)
   - a field can't parse according to it's type

We'll probably want to support most/all of these behaviors, e.g. `reshape(t, "inferextra", "dropmissing")` or some better syntax.
Maybe all of leaf handling should be moved into a different proc? (`reshape` `releaf`?)


### Doing other ingest-related processing in ZQL
Renames, annotation, validation, can all be done in ZQL.

- **rename**:  `rename ts=timestamp, src_ip=id.orig_h, dst_ip=id.dest_h, src_port=id.orig_p, dst_port=id.dest_p`
- **validation**: ZQL expressions
- **annotation**: `put source_type=zeek`



### Side note: building a ZNG schema registry (one day)

```
// map of string->type
const schemas = |{
   { "zeek_conn_log", {_path: string, id: zeek_conn_t, uid: string, proto: zenum, ...}},
     "zeek_http_t", {_path: string, id: zeek_conn_t, uid: string, method: bstring, ...}}
}|
```

A simple schema registry could be obtained by putting a simple HTTP API over this data structure. 

With a bit more structure, you could imagine this being tied into the ingest, and having the ingest ZQL flowgraph derived from the info in the registry. Documentation for fields could also be added here.


Other tooling that would be useful
==================================

- Shape finder: Tool that takes ZSON type values as input, and provides and output report describing the common fields (easy), how field sets are associated with field values (hard). The input to this tool can be obtained by running `* | by typeof(.)` over a sample dataset.

- Leaf finder: Tool that tries to infer the type of leaf values by parsing leaves according to different ZNG types (e.g. figure out a string is an IP)

- Fuse reducer: `fuse (typeof(.)) by alert.signature, alert.category` to get a per-alert type uberschema. 

- Comparison operators for record types: `contains(zeek_id_t)` true iff record has zeek `id` fields.


Plumbing notes
==============

- Need to immediately transform incoming json into zng via good old inference (or read it as zson, which could be the same thing...)
- How do we associate an ingest POST with a ZQL to run on it?





