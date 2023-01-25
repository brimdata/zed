---
sidebar_position: 1
sidebar_label: API
---

# Zed lake API

## _Status_

> This is a brief sketch of the functionality exposed in the
> Zed API. More detailed documentation of the API will be forthcoming.

## Endpoints

### Pools

#### Create pool

Create a new lake pool.

```
POST /pool
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| name | string | body | **Required.** Name of the pool. Must be unique to lake. |
| layout.order | string | body | Order of storage by primary key(s) in pool. Possible values: desc, asc. Default: asc. |
| layout.keys | [[string]] | body | Primary key(s) of pool. The element of each inner string array should reflect the hierarchical ordering of named fields within indexed records. Default: [[ts]]. |
| thresh | int | body | The size in bytes of each seek index. |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"name": "inventory", "layout": {"keys": [["product","serial_number"]]}}' \
     http://localhost:9867/pool
```

**Example Response**

```
{
  "pool": {
    "ts": "2022-07-13T21:23:05.323016Z",
    "name": "inventory",
    "id": "0x0f5ce9b9b6202f3883c9db8ff58d8721a075d1e4",
    "layout": {
      "order": "asc",
      "keys": [
        [
          "product",
          "serial_number"
        ]
      ]
    },
    "seek_stride": 65536,
    "threshold": 524288000
  },
  "branch": {
    "ts": "2022-07-13T21:23:05.367365Z",
    "name": "main",
    "commit": "0x0000000000000000000000000000000000000000"
  }
}
```

---

#### Rename pool

Change a pool's name.

```
PUT /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the requested pool. |
| name | string | body | **Required.** The desired new name of the pool. Must be unique to lake. |

**Example Request**

```
curl -X PUT \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"name": "catalog"}' \
     http://localhost:9867/pool/inventory
```

On success, HTTP 204 is returned with no response payload.

---

#### Delete pool

Permanently delete a pool.

```
DELETE /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the requested pool. |

**Example Request**

```
curl -X DELETE \
      http://localhost:9867/pool/inventory
```

On success, HTTP 204 is returned with no response payload.

---

### Branches

#### Load Data

Add data to a pool and return a reference commit ID.

```
POST /pool/{pool}/branch/{branch}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the pool. |
| branch | string | path | **Required.** Name of branch to which data will be loaded. |
|   | various | body | **Required.** Contents of the posted data. |
| Content-Type | string | header | MIME type of the posted content. If undefined, the service will attempt to introspect the data and determine type automatically. |
| csv.delim | string | query | Exactly one character specifing the field delimiter for CSV data. Defaults to ",". |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"product": {"serial_number": 12345, "name": "widget"}, "warehouse": "chicago"}
         {"product": {"serial_number": 12345, "name": "widget"}, "warehouse": "miami"}
         {"product": {"serial_number": 12346, "name": "gadget"}, "warehouse": "chicago"}' \
     http://localhost:9867/pool/inventory/branch/main
```

**Example Response**

```
{"commit":"0x0ed4f42da5763a9500ee71bc3fa5c69f306872de","warnings":[]}
```

---

#### Get Branch

Get information about a branch.

```
GET /pool/{pool}/branch/{branch}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the pool. |
| branch | string | path | **Required.** Name of branch. |

**Example Request**

```
curl -X GET \
     -H 'Accept: application/json' \
     http://localhost:9867/pool/inventory/branch/main
```

**Example Response**

```
{"commit":"0x0ed4fa21616ecd8fec9d6fd395ad876db98a5dae","warnings":null}
```

---

#### Delete Branch

Delete a branch.

```
DELETE /pool/{pool}/branch/{branch}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the pool. |
| branch | string | path | **Required.** Name of branch. |

**Example Request**

```
curl -X DELETE \
      http://localhost:9867/pool/inventory/branch/staging
```

On success, HTTP 204 is returned with no response payload.

---

#### Delete Data

Create a commit that reflects the deletion of some data in the branch. The data
to delete can be specified via a list of object IDs or
as a filter expression (see [limitations](../commands/zed.md#24-delete)).

```
POST /pool/{pool}/branch/{branch}/delete
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |
| object_ids | [string] | body | Object IDs to be deleted. |
| where | string | body | Filter expression (see [limitations](../commands/zed.md#24-delete)). |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"object_ids": ["274Eb1Kn8MTM6qxPyBpVTvYhLLa", "274EavbXt546VNelRLNXrzWShNh"]}' \
     http://localhost:9867/pool/inventory/branch/main/delete
```

**Example Response**

```
{"commit":"0x0ed4fee861e8fb61568783205a46a218182eba6c","warnings":null}
```

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"where": "product.serial_number > 12345"}' \
     http://localhost:9867/pool/inventory/branch/main/delete
```

**Example Response**

```
{"commit":"0x0f5ceaeaaec7b4c33cfdece9f2e8577ad89d21e2","warnings":null}
```

---

#### Merge Branches

Create a commit with the difference of the child branch added to the selected
branch.

```
POST /pool/{pool}/branch/{branch}/merge/{child}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch selected as merge destination. |
| child | string | path | **Required.** Name of child branch selected as source of merge. |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     http://localhost:9867/pool/inventory/branch/main/merge/staging
```

**Example Response**

```
{"commit":"0x0ed4ffc2566b423ee444c1c8e6bf964515290f4c","warnings":null}
```

---

#### Revert

Create a revert commit of the specified commit.

```
POST /pool/{pool}/branch/{branch}/revert/{commit}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch on which to revert commit. |
| commit | string | path | **Required.** ID of commit to be reverted. |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     http://localhost:9867/pool/inventory/branch/main/revert/27D22ifDw3Ms2NMzo8jXpDfpgjc
```

**Example Response**

```
{"commit":"0x0ed500ab6f80e5ac8a1b871bddd88c57fe963ab1","warnings":null}
```

---

#### Index Objects

Create an index of object(s) for the specified rule.

```
POST /pool/{pool}/branch/{branch}/index
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |
| rule_name | string | body | **Required.** Name of indexing rule. |
| tags | [string] | body | IDs of data objects to index. |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"rule_name": "MyRuleGroup", "tags": ["27DAbmqxukfABARaAHauARBJOXH", "27DAbeUBW7llN2mXAadYz00Zjpk"]}' \
     http://localhost:9867/pool/inventory/branch/main/index

```

**Example Response**

```
{"commit":"0x0ed510f4648da9742e8e9c35e3439d5b708843e1","warnings":null}
```

---

#### Update Index

Apply all rules or a range of index rules for all objects that are not indexed
in a branch.

```
POST /pool/{pool}/branch/{branch}/index/update
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |
| rule_names | [string] | body | Name(s) of index rule(s) to apply. If undefined, all rules will be applied. |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/json' \
     -H 'Content-Type: application/json' \
     -d '{"rule_names": ["MyRuleGroup", "AnotherRuleGroup"]}' \
     http://localhost:9867/pool/inventory/branch/main/index/update
```

**Example Response**

```
{"commit":"0x0ed51322b7d69bd0bddad10e31e3211408e34a88","warnings":null}
```

### Query

Execute a Zed query against data in a data lake.

```
POST /query
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| query | string | body | Zed query to execute. All data is returned if not specified. ||
| head.pool | string | body | Pool to query against Not required if pool is specified in query. |
| head.branch | string | body | Branch to query against. Defaults to "main". |
| ctrl | string | query | Set to "T" to include control messages in ZNG or ZJSON responses. Defaults to "F". |

**Example Request**

```
curl -X POST \
     -H 'Accept: application/x-zson' \
     -H 'Content-Type: application/json' \
     http://localhost:9867/query -d '{"query":"from inventory@main | count() by warehouse"}'
```

**Example Response**

```
{warehouse:"chicago",count:2(uint64)}
{warehouse:"miami",count:1(uint64)}
```

**Example Request**

```
curl -X POST \
     -H 'Accept: application/x-zjson' \
     -H 'Content-Type: application/json' \
     http://localhost:9867/query?ctrl=T -d '{"query":"from inventory@main | count() by warehouse"}'
```

**Example Response**

```
{"type":"QueryChannelSet","value":{"channel_id":0}}
{"type":{"kind":"record","id":30,"fields":[{"name":"warehouse","type":{"kind":"primitive","name":"string"}},{"name":"count","type":{"kind":"primitive","name":"uint64"}}]},"value":["miami","1"]}
{"type":{"kind":"ref","id":30},"value":["chicago","2"]}
{"type":"QueryChannelEnd","value":{"channel_id":0}}
{"type":"QueryStats","value":{"start_time":{"sec":1658193276,"ns":964207000},"update_time":{"sec":1658193276,"ns":964592000},"bytes_read":55,"bytes_matched":55,"records_read":3,"records_matched":3}}
```

---

### Events

Subscribe to an events feed, which returns an event stream in the format of
[server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html).
The MIME type specified in the request's Accept HTTP header determines the format
of `data` field values in the event stream.

```
GET /events
```

**Params**

None

**Example Request**

```
curl -X GET \
     -H 'Accept: application/json' \
     http://localhost:9867/events
```

**Example Response**

```
event: pool-new
data: {"pool_id": "1sMDXpVwqxm36Rc2vfrmgizc3jz"}

event: pool-update
data: {"pool_id": "1sMDXpVwqxm36Rc2vfrmgizc3jz"}

event: pool-commit
data: {"pool_id": "1sMDXpVwqxm36Rc2vfrmgizc3jz", "commit_id": "1tisISpHoWI7MAZdFBiMERXeA2X"}

event: pool-delete
data: {"pool_id": "1sMDXpVwqxm36Rc2vfrmgizc3jz"}
```

---

## Media Types

For response content types, the service can produce a variety of formats. To
receive responses in the desired format, include the MIME type of the format in
the request's Accept HTTP header.

If the Accept header is not specified, the service will return ZSON as the
default response format for the endpoints described above.

The supported MIME types are as follows:

| Format | MIME Type |
| ------ | --------- |
| Arrow IPC Stream | application/vnd.apache.arrow.stream |
| CSV | text/csv |
| JSON | application/json |
| NDJSON | application/x-ndjson |
| Parquet | application/x-parquet |
| ZJSON | application/x-zjson |
| ZSON | application/x-zson |
| ZNG | application/x-zng |
