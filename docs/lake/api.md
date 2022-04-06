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
| layout.order | string | body | Order of value storage in pool. Possible values: desc, asc. Default: asc. |
| layout.keys | [string] | body | Primary key(s) of pool. Default: ts. |
| thresh | int | body | The size in bytes of each seek index. |

#### Rename pool

Changes a pool's name.

```
PUT /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |
| name | string | body | **Required.** The desired new name of the pool. Must be unique. |

#### Delete pool

Permanently deletes a pool.

```
DELETE /pool/{pool}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |

### Branches

#### Load Data

Adds data to a pool's staging and returns a reference commit ID.

```
POST /pool/{pool}/branch/{branch}
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| Content-Type | string | header | Mime type of the posted content. If undefined, the service will attempt to introspect the data and determine type automatically. |
|   | various | body | Contents of the posted data. |

#### Get Branch

Get information about a branch.

```
POST /pool/{pool}/branch/{branch}
```

### Delete Branch

Delete a branch.

```
DELETE /pool/{pool}/branch/{branch}
```

#### Delete Data

Takes a list of commit IDs or object IDs in a branch and creates a deletion
commit of all referenced objects.

```
POST /pool/{pool}/branch/{branch}/delete
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| object_ids | [string] | body | Commit IDs or object IDs to be deleted. |

#### Merge Branches

Creates a commit with the difference of the child branch added to the selected
branch.

```
POST /pool/{pool}/branch/{branch}/merge/{child}
```

#### Revert

Creates a revert commit of the specified commit.

```
POST /pool/{pool}/branch/{branch}/revert/{commit}
```

#### Index Object

Creates an index of an object for the specified rule.

```
POST /pool/{pool}/branch/{branch}/index
```

#### Update Index

Applies all or a range of index rules for all objects that are not indexed.

```
POST /pool/{pool}/branch/{branch}/index/update
```

### Query

Executes a Zed query against data in a data lake.

```
POST /query
```

**Params**

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| query | string | body | Zed query to execute. |
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

Creates an index rule for a specified field.

```
POST /index
```

#### Delete Index Rule

Deletes the specified index rule. Any created object indexes will persist.

```
DELETE /index
```

## Media Types

For responses content types, the service can handle a variety of formats. To
receive responses in the desired format, include the mime type of the format in
the requests ACCEPT HTTP header.

If the ACCEPT header is not specified, the service will return json as default.

The supported mime types are as follows:

| Format | Mime Type |
| ------ | --------- |
| json | application/json |
| ndjson | application/ndjson |
| zjson | application/x-zjson |
| zson | application/x-zson |
| zng | application/x-zng |
