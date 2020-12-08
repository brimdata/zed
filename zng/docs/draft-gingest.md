Generic ingest outline
======================

We want a way to normalize, and "restore" json objects that are ingested into a zng lake.


- validate
  // xxx do we need to talk about validate? since we're probably not doing it here. 
  is field A present and is it numeric
  if field B is present, are fields C and D present?
- normalize
- restore 


Background: The existing "json types" system
--------------------------------------------

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

The proposal is to use ZQL to write ingest classification, transforms, and validation.  This idea was previously floated in sync (by Noah?), but at that time ZQL was far from being able to support ingest-related transforms.

Now, with the recent addition of type values to ZQL ("first class types"), and the forthcoming addition of complex literals (à la ZSON), we're a lot closer. We dont have all the bits yet, but now it's mostly just a matter of filling in a proc or two, and adding one or two flowgraph constructs to zql.

If we can express generic ingest stuff in ZQL, we get a unified language to express and reason both about ingest/ETL and analytics. Some advantages:

- Flexible: it should cover all the needs described above
- Extensible: we add the requisite proc/functions to ZQL (such as `reshape` below). Of course you can do that with a JSON DSL too, but it gets ugly quickly.
- Users only need to learn one thing, as opposed to having to learn and understand a different (typically ad hoc) DSL for ingest, such as logstash for Elastic or relabelling rules for Prometheus.
- Provides a reasonable starting point for query-time transformations. For example, if you've ingested 1PB and now realize you want to change a field name, you change your ZQL ingest config so that newly ingested data will have the new name, and for old data, we can run that same ZQL at query time to convert it. Some of it can also be optimized to avoid reading everything. For example, if you've renamed `_path` to `_zeek_path` and search for `_zeek_path=conn`, we can transform that search to `_path=conn` when looking at old logs. This is all of course not something we need to tackle anytime soon, and there is lots to design here, but doing everything in ZQL is a good basis.

- Of course, 



Zeek example
============

Define types

```
const zeek_id_t = {orig_h:ip, orig_p:port, resp_h:ip, resp_p:port}
const zeek_conn_t = {_path: string, id: zeek_conn_t, uid: string, proto: zenum, ...}
const zeek_http_t = {_path: string, id: zeek_conn_t, uid: string, method: bstring, ...}
```


Apply types

```
* | case {
  _path=conn | put . = reshape(zeek_conn_type) 
  _path=http | put . = reshape(http_conn_type)
  ..
}
```



More on apply
=============


