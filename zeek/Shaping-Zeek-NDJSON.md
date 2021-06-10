# Shaping Zeek NDJSON

- [Summary](#summary)
- [Reference Shaper Contents](#reference-shaper-contents)
  * [Leading Type Definitions](#leading-type-definitions)
  * [Type Definitions Per Zeek Log `_path`](#type-definitions-per-zeek-log-_path)
  * [Mapping From `_path` Values to Types](#mapping-from-_path-values-to-types)
  * [Zed Pipeline Config](#zed-pipeline-config)
- [Invoking the Shaper From `zq`](#invoking-the-shaper-from-zq)
- [Zeek Version/Configuration](#zeek-versionconfiguration)
- [Importing Shaped Data Into Brim](#importing-shaped-data-into-brim)
- [Contact us!](#contact-us)

> **Note:** This document describes functionality that's available in Zed
> `v0.30.0` and newer (hence also [Brim](https://github.com/brimdata/brim)
> `v0.25.0` and newer). If you're looking for docs regarding the legacy
>`types.json` approach that was used in Zed [`v0.29.0`](https://github.com/brimdata/zed/releases/tag/v0.29.0)
> (or Brim [`0.24.0`](https://github.com/brimdata/brim/releases/tag/v0.24.0))
> and older you can find it [here](https://github.com/brimdata/zed/blob/v0.29.0/zeek/README.md).

# Summary

As described in [Reading Zeek Log Formats](Reading-Zeek-Log-Formats.md),
logs output by Zeek in NDJSON format lose much of their rich data typing that
was originally present inside Zeek. This detail can be restored using a Zed
shaper configuration, such as the reference example [`shaper.zed`](shaper.zed)
that can be found in this directory of the repository.

A full description of all that's possible with shaping configurations is beyond
the scope of this doc. However, this example configuration is quite simple and
is described below.

# Reference Shaper Contents

The example `shaper.zed` may seem large, but ultimately its config follows a
fairly simple pattern that repeats across the many [Zeek log types](https://docs.zeek.org/en/master/script-reference/log-files.html).

## Leading Type Definitions

The top three lines define types that are referenced further below in the main
portion of the config.

```
type port=uint16;
type zenum=string;
type conn_id={orig_h:ip,orig_p:port,resp_h:ip,resp_p:port};
```
The `port` and `zenum` types are described further in the [Zed/Zeek Data Type Compatibility](Data-Type-Compatibility.md)
doc. The `conn_id` type will just save us from having to repeat these fields
individually in the many Zeek record types that contain an embedded `id`
record.

## Type Definitions Per Zeek Log `_path`

The bulk of the configuration consists of detailed per-field data type
definitions for each record in the default set of NDJSON logs output by Zeek.
These type definitions reference the types we defined above, such as `port`
and `conn_id`. The syntax for defining primitive and complex types follows the
relevant sections of the [ZSON Format](../docs/formats/zson.md#3-the-zson-format)
specification.

```
...
type conn={_path:string,ts:time,uid:bstring,id:conn_id,proto:zenum,service:bstring,duration:duration,orig_bytes:uint64,resp_bytes:uint64,conn_state:bstring,local_orig:bool,local_resp:bool,missed_bytes:uint64,history:bstring,orig_pkts:uint64,orig_ip_bytes:uint64,resp_pkts:uint64,resp_ip_bytes:uint64,tunnel_parents:|[bstring]|,_write_ts:time};
type dce_rpc={_path:string,ts:time,uid:bstring,id:conn_id,rtt:duration,named_pipe:bstring,endpoint:bstring,operation:bstring,_write_ts:time};
...
```

> **Note:** See [the role of `_path`](Reading-Zeek-Log-Formats.md#the-role-of-_path)
> doc for important details if you're using Zeek's built-in [ASCII logger](https://docs.zeek.org/en/current/scripts/base/frameworks/logging/writers/ascii.zeek.html)
> to generate NDJSON rather than the [JSON Streaming Logs](https://github.com/corelight/json-streaming-logs) package.

## Mapping From `_path` Values to Types

The next section is just simple mapping from the string values typically found
in the Zeek `_path` field to the name of one of the types we defined above.

```
const schemas = |{
  "broker": broker,
  "capture_loss": capture_loss,
  "cluster": cluster,
  "config": config,
  "conn": conn,
  "dce_rpc": dce_rpc,
...
```

## Zed Pipeline Config

The config ends with a pipeline that stitches together everything we've defined
so far.

```
put this := unflatten(this) | put this := shape(schemas[_path])
```

Picking this apart, it transforms reach record as it's being read, in two
steps:

1. `unflatten()` reverses the Zeek NDJSON logger's "flattening" of nested
   records, e.g., how it populates a field named `id.orig_h` rather than
   creating a field `id` with sub-field `orig_h` inside it. Restoring the
   original nesting now gives us the option to reference the record named `id`
   in the Zed language and access the entire 4-tuple of values, but still
   access the individual values using the same dotted syntax like `id.orig_h`
   when needed.

2. `shape()` applies the type definition based on the value found in the
   incoming record's `_path` field. This includes:

   * For any fields referenced in the type definition that aren't present in
     the input record, the field is added with a `null` value. (Note: This
     could be performed separately via the `fill()` function.)

   * The data type of each field in the type definition is applied to the
     field of that name in the input record. (Note: This could be performed
     separately via the `cast()` function.)

   * The fields in the input record are ordered to match the order in which
     they appear in the type definition. (Note: This could be performed
     separately via the `order()` function.)

   Any fields that appear in the input record that are not present in the
   type definition are kept and assigned an inferred data type. If you would
   prefer to have such additional fields dropped (i.e., to maintain strict
   adherence to the shape), append a call to the `crop()` function to the
   Zed pipeline, e.g.:

      ```
      put this := unflatten(this) | put this := shape(schemas[_path]) | put this := crop(schemas[_path])
      ```

   Open issues [zed/2585](https://github.com/brimdata/zed/issues/2585) and
   [zed/2776](https://github.com/brimdata/zed/issues/2776) both track planned
   future improvements to this part of the configuration.

# Invoking the Shaper From `zq`

A shaper is typically invoked via the `-I` option of `zq`.

For example, if working in a directory containing many NDJSON logs, the
reference shaper config can be applied to all the records they contain and
output them all in a single binary [ZNG](../docs/formats/zng.md) file as
follows:

```
zq -I shaper.zed *.log > /tmp/all.zng
```

If you wish to apply the shaping configuration and then perform additional
operations on the richly-typed records, the Zed query on the command line
should begin with a `|`, as this appends it to the pipeline at the bottom of
the shaper from the included file.

For example, to count Zeek `conn` records into CIDR-based buckets based on
originating IP address:

```
zq -I shaper.zed -f table '| count() by network_of(id.orig_h) | sort -r' conn.log
```

[zed/2584](https://github.com/brimdata/zed/issues/2584) tracks a planned
improvement for this use of `zq -I`.

If you intend to frequently shape the same NDJSON data, you may want to create
an [alias](https://tldp.org/LDP/abs/html/aliases.html) in your
shell to always invoke `zq` with the necessary `-I` flag pointing to the path
of your finalized shaper. [zed/1059](https://github.com/brimdata/zed/issues/1059)
tracks a planned enhancement to persist such settings within Zed itself rather
than relying on external mechanisms such as shell aliases.

# Zeek Version/Configuration

The fields and data types in the reference `shaper.zed` reflect the default
NDJSON-format logs output by the Zeek version referenced in the comments at the
top of that file. This configuration has been revisited periodically as new
Zeek versions have been released and in that time we've only seen a couple new
fields appear and none have been removed. Because of this, we expect the
shaper configuration should be usable as-is for Zeek releases older than the
one most recently tested, since fields in the shaper not present in your
environment would just be filled in with `null` values. All attempts will be
made to update it in a timely manner as new Zeek versions are released.

However, if you have modified your Zeek installation with [packages](https://packages.zeek.org/)
or other customizations, or if you are using a [Corelight Sensor](https://corelight.com/products/appliance-sensors/)
that produces Zeek logs with many fields and logs beyond those found in open
source Zeek, the reference shaper will not cover all the fields in your logs.
[As described above](#zed-pipeline-config), the reference shaper will assign
inferred types to such additional fields. By exploring your data, you can then
iteratively enhance your shaper to match your environment. If you need
assistance, please speak up on our [public Slack](https://www.brimsecurity.com/join-slack/).

# Importing Shaped Data Into Brim

If you wish to browse your shaped data with [Brim](https://github.com/brimdata/brim),
the best way to accomplish this at the moment would be to use `zq` to convert
it to ZNG [as shown above](#invoking-the-shaper-from-zq), then drag the ZNG
into Brim as you would any other log. An enhancement [zed/2695](https://github.com/brimdata/zed/issues/2695)
is planned that will soon make it possible to attach your shaper config to a
Pool. This will allow you to drag the original NDJSON logs directly into the
Pool in Brim and have the shaping applied as the records are being committed to
the Pool.

# Contact us!

If you're having difficulty, interested in shaping other data sources, or
just have feedback, please join our [public Slack](https://www.brimsecurity.com/join-slack/)
and speak up or [open an issue](https://github.com/brimdata/zed/issues/new/choose).
Thanks!
