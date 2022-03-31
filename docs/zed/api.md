# Zed lake service API

> Note: This file contains a brief sketch of the functionality exposed in the
> Zed API. More fine-grained documentation will be forthcoming.

## Contents

* [Endpoints](#endpoints)
  + [Pools](#pools)
    - [Create Pool](#create-pool)
    - [Rename Pool](#rename-pool)
    - [Delete Pool](#delete-pool)
  + [Branches](#branches)
    - [Load Data](#load-data)
    - [Get Branch](#get-branch)
    - [Delete Branch](#delete-branch)
    - [Delete Data](#delete-data)
    - [Merge Branches](#merge-branches)
    - [Revert](#revert)
    - [Index Object](#index-object)
    - [Update Index](#update-index)
  + [Query](#query)
  + [Events](#events)
  + [Index Rules](#index-rules)
    - [Create Index Rule](#create-index-rule)
    - [Delete Index Rule](#delete-index-rule)
* [Media Types](#media-types)
* [Example](#example)


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
| layout.keys | [string] | body | Primary key(s) of pool. Default: ts. |
| layout.keys | [string] | body | Primary key(s) of pool. Default: ts. |
| thresh | int | body | The size in bytes of each seek index. |

#### Rename pool

Change a pool's name.

```
PUT /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the requested pool. |
| name | string | body | **Required.** The desired new name of the pool. Must be unique. |

#### Delete pool

Permanently delete a pool.

```
DELETE /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the requested pool. |

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

#### Get Branch

Get information about a branch.

```
POST /pool/{pool}/branch/{branch}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID or name of the pool. |
| branch | string | path | **Required.** Name of branch. |

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
| pool | string | path | ID of the pool. |
| object_ids | [string] | body | Commit IDs or object IDs to be deleted. |

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
| child | string | path | **Required.** Name of child branch. |

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

#### Index Object

Create an index of an object for the specified rule.

```
POST /pool/{pool}/branch/{branch}/index
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |
| rule_name | string | body | **Required.** Name of indexing rule. |
| tags | array&lt;string> | body | IDs of data objects to index. |

#### Update Index

Apply all rules or a range of index rules for all objects that are not indexed.

```
POST /pool/{pool}/branch/{branch}/index/update
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | **Required.** ID of the pool. |
| branch | string | path | **Required.** Name of branch. |

### Query

Execute a Zed query against data in a data lake.

```
POST /query
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| query | string | body | Zed query to execute. (All data is returned if not specified.) ||
| head.pool | string | body | Pool to query against (Not required if pool is specified in query). |
| head.branch | string | body | Branch to query against (Defaults to main). |

### Events

Subscribe to an events feed.

```
GET /events
```

#### Response

An event-stream in the format of [Server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html).

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

### Index Rules

#### Create Index Rule

Create an index rule for a specified field.

```
POST /index
```

#### Delete Index Rule

Delete the specified index rule. Any created object indexes will persist.

```
DELETE /index
```

## Media Types

For response content types, the service can handle a variety of formats. To
receive responses in the desired format, include the MIME type of the format in
the request's Accept HTTP header.

If the Accept header is not specified, the service will return JSON as the default.

The supported MIME types are as follows:

| Format | MIME Type |
| ------ | --------- |
| json | application/json |
| ndjson | application/ndjson |
| zjson | application/x-zjson |
| zson | application/x-zson |
| zng | application/x-zng |

## Example

Here we [create a pool](#create-pool) called "inventory" with a primary key
field "product_name", requesting the response data in ZSON.

Request:

```
 curl -X POST \
      -H 'Content-Type: application/json' \
      -H "Accept: application/x-zson" \
      -d '{"name": "inventory", "layout": {"keys": [["product_name"]]}}' \
      http://localhost:9867/pool
```

Response:

```
{pool:{ts:2022-03-28T15:29:05.177632Z,name:"inventory",id:0x0ecf86419ae7e6c288b448da9f18e3c28adce8d0(=ksuid.KSUID),layout:{order:"asc"(=order.Which),keys:[["product_name"](=field.Path)](=field.List)}(=order.Layout),seek_stride:65536,threshold:524288000}(=pools.Config),branch:{ts:2022-03-28T15:29:05.178407Z,name:"main",commit:0x0000000000000000000000000000000000000000(ksuid.KSUID)}(=branches.Config)}(=lake.BranchMeta)
```
