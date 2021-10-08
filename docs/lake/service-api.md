# Zed lake service API

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

## Endpoints

### Pool managment endpoints:
  - [List Pools](#list-pools)
  - [Create Pool](#create-pool)
  - [Get Pool](#get-pool)
  - [Rename Pool](#rename-pool)
  - [Drop Pool](#drop-pool)

### Query endpoints:
  - [Query](#query)

### Add data endpoints:
  - [Add](#add)
  - [Add By Path](#add-by-path)

### View/edit pool commit history:
  - [Squash](#squash)
  - [Commit](#commit)
  - [Delete](#delete)
  - [List Staging](#list-staging)
  - [List Segments](#list-segments)
  - [List Log](#list-log)

### Events subscription:
  - [Events](#events)

<!-- XXX: Index revamp -->
<!-- - [Create Index](#create-index) -->
<!-- - [List Indexes](#list-indexes) -->
<!-- - [Drop Index](#drop-index) -->
<!-- - [Index Data](#index-data) -->


### List pools

List all the pools in the lake.

```
GET /pool
```

#### Response

A list of [pool configs](#pool-config).

```
[
  {
    "kind": "PoolConfig",
    "value": {
      "id": "1sOfdIOF6UvYxbbOrKvokilZKvT",
      "layout": {
        "keys": [
          [
            "ts"
          ]
        ],
        "order": "desc"
      },
      "name": "test",
      "threshold": 524288000,
      "version": 0
    }
  }
]
```

### Create pool

Create a new pool in a lake.

```
POST /pool
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| name | string | body | **Required.** Name of the pool. Must be unique to lake. |
| layout.order | string | body | Order of value storage in pool. Possible values: desc, asc. Default: asc. |
| layout.keys | array<string> | body | Primary key(s) of pool. Default: ts. |
| thresh | int | body | The size in bytes of each seek index. | 

#### Response

A [pool config](#pool-config).

```
{
  "kind": "PoolConfig",
  "value": {
    "id": "1sOfdIOF6UvYxbbOrKvokilZKvT",
    "layout": {
      "keys": [
        [
          "ts"
        ]
      ],
      "order": "desc"
    },
    "name": "test",
    "threshold": 524288000,
    "version": 0
  }
}
```

### Get Pool

Get a pool's configuration.

```
GET /pool/{pool}
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |
| stats | bool | query | Include pool stats in response. |

#### Response

A [pool config](#pool-config).

```
{
  "kind": "PoolConfig",
  "value": {
    "id": "1sOfdIOF6UvYxbbOrKvokilZKvT",
    "layout": {
      "keys": [
        [
          "ts"
        ]
      ],
      "order": "desc"
    },
    "name": "test",
    "threshold": 524288000,
    "version": 0
  }
}
```

### Get Pool Info

```
GET /pool/{pool}/info
```

Get a pool's configuration as well as stats about the pool.

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |
| at | string | query | Commit or journal ID for time travel. |

#### Response

A [pool info object](#pool-config).

```
{
  "kind": "PoolConfig",
  "value": {
    "config": {
      "id": "1sOfdIOF6UvYxbbOrKvokilZKvT",
      "layout": {
        "keys": [
          [
            "ts"
          ]
        ],
        "order": "desc"
      },
      "name": "test",
      "threshold": 524288000,
      "version": 0
	},
	"size": "1024", // in bytes
	"span": {} // XXX
  }
}
```
### Rename pool

Changes a pool's name.

```
PUT /pool/{pool}
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |
| name | string | body | **Required.** The desired new name of the pool. Must be unique. |

#### Response

A list of [Pool Configs](#pool-config).

```
[
  {
    "kind": "PoolConfig",
    "value": {
      "id": "1sOfdIOF6UvYxbbOrKvokilZKvT",
      "layout": {
        "keys": [
          [
            "ts"
          ]
        ],
        "order": "desc"
      },
      "name": "test",
      "threshold": 524288000,
      "version": 0
    }
  }
]
```

### Drop pool

Permanently deletes a pool from a lake.

```
DELETE /pool/{pool}
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID or name of the requested pool. |

#### Response

```
Status: 204 No Content
```

### Query

Executes a Zed query against data in a data lake.

```
POST /query
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| query | array<string> | body | Zed query to execute. |
| includes | array<string> | body | XXX |

#### Response

A stream of ZNG records.

```
curl \
  -X POST \
  -X "Accept: application/x-zson" \
  http://localhost:9867/query \
  -d '{"zed": "from test | _path = http"}'
```

```
{
    _path: "http",
    ts: 2018-03-25T01:08:40.752884Z,
    uid: "Cox5bO350nHiWJ1mzf" (bstring),
    id: {
        orig_h: 10.47.42.200,
        orig_p: 49967 (port=(uint16)),
        resp_h: 198.189.255.222,
        resp_p: 80 (port)
    } (=0),
    method: "GET" (bstring),
    host: "gadgets.live.com",
    uri: "/config.xml",
} (=4)
{
    _path: "http",
    ts: 2018-03-25T01:08:40.638527Z,
    uid: "CHJ8jb2uExGAW2t4uh",
    id: {
        orig_h: 10.47.44.68,
        orig_p: 49674,
        resp_h: 198.189.255.222,
        resp_p: 80
    },
    method: "GET",
    host: "gadgets.live.com",
    uri: "/config.xml",
} (4)
```

### Add

Adds data to a pool's staging and returns a reference commit ID. Data will
not be queryable until the data is committed.

```
POST /pool/{pool}/add
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| Content-Type | string | header | Mime type of the posted content. If undefined, the service will attempt to introspect the data and determine type automatically. |
|   | various | body | Contents of the posted data. |

#### Response

```
{
  "kind": "AddResponse",
  "value": {
    "warnings": [<string>],
    "commit": {
      "kind": "StagedCommit",
      "value": {
        "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz"
      }
    }
  }
}
```

### Add by path

Adds data via a list of URIs. URIs must be accessible by the lake service
which will open them and write the data directly to staging. Like the add
endpoint returns a reference commit ID. Data will not be queryable until the
data is committed.

```
POST /pool/{pool}/add/path
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| paths | array<string> | body | A list of uri's accessible to the lake service. Can be a local file path or the uri of an s3 object. |

#### Response

A stream of AddStatus|AddWarning messages ending with an AddComplete message.
Accept header for application/json not supported, defaults to
application/ndjson.

```
{
  "kind": "AddWarning",
  "value": {
    "warning": "a warning message"
  }
}
{
  "kind": "AddStatus",
  "value": {
    "read_size": 1024,
    "total_size": 1024,
  }
}
{
  "kind": "AddComplete",
  "value": {
    "error": null,
    "commit": {
      "kind": "StagedCommit",
      "value": {
        "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz"
      }
    }
  }
}
```

### Squash

Takes multiple pending commits in a pool and combines them into a single pending
commit, returning the new tag of the squashed commits. The order of the tags is
significant as the pending commits are assembled into a snapshot reflecting the
indicated order of any underlying add/delete operations.  If a delete operation
encounters a tag that is not present in the implied commit, the squash will
fail.

```
POST /pool/{pool}/squash
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| commits | array<string> | body | An in order array of pending commit tags. |

#### Response

A [staged commit](#staged-commit).

```
{
  "kind": "StagedCommit",
  "value": {
    "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz"
  }
}
```

### Commit

Transactionally add a pending commit to a pool.

```
POST /pool/{pool}/commit
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| commit | string | body | Hash of the staging commit. |
| user | string | body | Name of user doing the commit (XXX At some point this will taken from user authentication) |
| message | string | body | Commit message. |

#### Response

```
Status: 204 No Content
```

### Delete

Takes a list of commit tags or data segment tags in the specified pool and
stages a deletion commit for each object listed and each object in the listed
commits.

```
POST /pool/{pool}/delete
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| tags | array<string> | body | Commit tags or segments tag to be deleted. XXX Would this not work for index objects as well? |

#### Response

A [staged commit](#staged-commit).

```
{
  "kind": "StagedCommit",
  "value": {
    "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz"
  }
}
```


### List Staging

List information about commits in staging.

```
GET /pool/{pool}/staging
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| tag | string | query | Specific tag to show from staging. Can be specified multiple times for multiple objects. |

#### Response

A list of [actions](#actions).

```
{
  [
    {
      "kind": "Add",
      "value": {
        "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz",
        "segment": {
          "id": "1sMDRRqpCGgWXm7DYCvXhfE3VGv",
          "meta": {
            "count": 1280803,
            "first": "2018-03-24T19:59:15.584818Z",
            "last": "2018-03-24T06:54:43.122816Z",
            "row_size": 93109436,
            "size": 341316030
          }
        }
      }
    },
    {
      "kind": "StagedCommit",
      "value": {
        "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz"
      }
    }
  ]
}
```

### List Segments

List segments in a pool.

```
GET /pool/{pool}/segment
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| at | string | query | Commit or journal ID for time travel. |
| partition | string | query | Display partitions as determined by scan logic. Accepts "T" for true and "F" for false. (Default: false) |

#### Response

A list of [segment references](#segment-reference).

```
[
  {
    "kind": "Reference",
    "value": {
      "id": "1sMDRRqpCGgWXm7DYCvXhfE3VGv",
      "meta": {
        "count": 2397065,
        "first": "2018-03-23T22:17:54.187963Z",
        "last": "2018-03-23T21:52:39.571939Z",
        "row_size": 112812133,
        "size": 524288037
      }
    }
  }
]
```

### List Log

List a pool's commit log.

```
GET /pool/{pool}/log
```

#### Params

| Name | Type | In | Description |
| ---- | ---- | -- | ----------- |
| pool | string | path | ID of the pool. |
| at | string | query | Commit or journal ID for time travel. XXX Not supported in go api, should it? |

#### Response

A list of [actions](#actions).

```
[
  {
    "kind": "Add",
    "value": {
      "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz",
      "segment": {
        "id": "1sMDRRqpCGgWXm7DYCvXhfE3VGv",
        "meta": {
          "count": 1280803,
          "first": "2018-03-24T19:59:15.584818Z",
          "last": "2018-03-24T06:54:43.122816Z",
          "row_size": 93109436,
          "size": 341316030
        }
      }
    }
  },
  {
    "kind": "CommitMessage",
    "value": {
      "author": "nibs@Matthews-MacBook-Air.local",
      "commit": "1sMDXpVwqxm36Rc2vfrmgizc3jz",
      "date": "2021-05-10T19:25:19.298139Z",
      "message": "Add some test data."
    }
  }
]
```

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

## Object Reference

### Actions

XXX

### Segment Reference

XXX

### Commit

XXX

### Pool Config

XXX
