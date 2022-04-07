---
sidebar_position: 1
sidebar_label: API
---

# Zed lake API

---

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
     -H 'Content-Type: application/json' \
     -d '{"name": "inventory", "layout": {"keys": [["product","serial_number"],["warehouse"]]}}' \
     http://localhost:9867/pool
```

**Example Response**

```
{
  "pool": {
    "ts": "2022-04-01T18:18:50.54718Z",
    "name": "inventory",
    "id": "0x0ed4f40a9ab28531c25ebc860fac69fe52fe6eb7",
    "layout": {
      "order": "asc",
      "keys": [
        [
          "product",
          "serial_number"
        ],
        [
          "warehouse"
        ]
      ]
    },
    "seek_stride": 65536,
    "threshold": 524288000
  },
  "branch": {
    "ts": "2022-04-01T18:18:50.547752Z",
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
      -H 'Content-Type: application/json' \
      http://localhost:9867/pool/inventory \
      -d '{"name": "catalog"}'
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

**Example Request**

```
curl -X POST \
      http://localhost:9867/pool/inventory/branch/main \
      -d '{"product": {"serial_number": 12345, "name": "widget"}, "warehouse": "chicago"}
          {"product": {"serial_number": 12345, "name": "widget"}, "warehouse": "miami"}
          {"product": {"serial_number": 12346, "name": "gadget"}, "warehouse": "chicago"}'
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

Take a list of commit IDs or object IDs in a branch and create a deletion
commit of all referenced objects.

```
POST /pool/{pool}/branch/{branch}/delete
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |
| object_ids | [string] | body | Commit IDs or object IDs to be deleted. |

**Example Request**

```
curl -X POST \
      -H 'Content-Type: application/json' \
      http://localhost:9867/pool/inventory/branch/main/delete \
      -d '{"object_ids": ["274Eb1Kn8MTM6qxPyBpVTvYhLLa", "274EavbXt546VNelRLNXrzWShNh"]}'
```

**Example Response**

```
{"commit":"0x0ed4fee861e8fb61568783205a46a218182eba6c","warnings":null}
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
      -H 'Content-Type: application/json' \
      http://localhost:9867/pool/inventory/branch/main/index \
      -d '{"rule_name": "MyRuleGroup", "tags": ["27DAbmqxukfABARaAHauARBJOXH", "27DAbeUBW7llN2mXAadYz00Zjpk"]}'
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
      -H 'Content-Type: application/json' \
      http://localhost:9867/pool/inventory/branch/main/index/update \
      -d '{"rule_names": ["MyRuleGroup", "AnotherRuleGroup"]}'
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
| head.branch | string | body | Branch to query against Defaults to "main". |

**Example Request**

```
curl -X POST \
     -H 'Content-Type: application/json' \
     http://localhost:9867/query -d '{"query":"from inventory@main | count() by warehouse"}'
```

**Example Response**

```
{warehouse:"chicago",count:2(uint64)}
{warehouse:"miami",count:1(uint64)}
```

---

### Events

Subscribe to an events feed, which returns an event stream in the format of
[server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html).

```
GET /events
```

**Params**

None

**Example Request**

```
curl -X GET \
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

If the Accept header is not specified, the service will return JSON as the
default response format for the endpoints described above, with the exception
of the [query](#query) endpoint that returns ZSON by default.

The supported MIME types are as follows:

| Format | MIME Type |
| ------ | --------- |
| JSON | application/json |
| NDJSON | application/x-ndjson |
| ZJSON | application/x-zjson |
| ZSON | application/x-zson |
| ZNG | application/x-zng |
