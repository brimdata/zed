# Reading Zeek Log Formats

- [Summary](#summary)
- [Zeek TSV](#zeek-tsv)
- [Zeek NDJSON](#zeek-ndjson)
- [The Role of `_path`](#the-role-of-_path)

# Summary

Zed is capable of reading both common Zeek log formats. This document
provides guidance for what to expect when reading logs of these formats using
the Zed tools such as [`zq`](../cmd/zed/README.md#zq).

# Zeek TSV

[Zeek TSV](https://docs.zeek.org/en/master/log-formats.html#zeek-tsv-format-logs)
is Zeek's default output format for logs. This format can be read automatically
(i.e., no `-i` command line flag is necessary to indicate the input format)
with the Zed tools such as `zq`.

The following example shows a TSV `conn.log` being read via `zq` and
output as [ZSON](../docs/data-model/zson.md).

#### conn.log:

```mdtest-input conn.log
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#path	conn
#open	2019-11-08-11-44-16
#fields	ts	uid	id.orig_h	id.orig_p	id.resp_h	id.resp_p	proto	service	duration	orig_bytes	resp_bytes	conn_state	local_orig	local_resp	missed_bytes	history	orig_pkts	orig_ip_bytes	resp_pkts	resp_ip_bytes	tunnel_parents
#types	time	string	addr	port	addr	port	enum	string	interval	count	count	string	bool	bool	count	string	count	count	count	count	set[string]
1521911721.255387	C8Tful1TvM3Zf5x8fl	10.164.94.120	39681	10.47.3.155	3389	tcp	-	0.004266	97	19	RSTR	-	-	0	ShADTdtr	10	730	6	342	-
```

#### Example:

```mdtest-command
zq -Z 'head 1' conn.log
```

#### Output:
```mdtest-output
{
    _path: "conn",
    ts: 2018-03-24T17:15:21.255387Z,
    uid: "C8Tful1TvM3Zf5x8fl" (bstring),
    id: {
        orig_h: 10.164.94.120,
        orig_p: 39681 (port=(uint16)),
        resp_h: 10.47.3.155,
        resp_p: 3389 (port)
    },
    proto: "tcp" (=zenum),
    service: null (bstring),
    duration: 4.266ms,
    orig_bytes: 97 (uint64),
    resp_bytes: 19 (uint64),
    conn_state: "RSTR" (bstring),
    local_orig: null (bool),
    local_resp: null (bool),
    missed_bytes: 0 (uint64),
    history: "ShADTdtr" (bstring),
    orig_pkts: 10 (uint64),
    orig_ip_bytes: 730 (uint64),
    resp_pkts: 6 (uint64),
    resp_ip_bytes: 342 (uint64),
    tunnel_parents: null (|[bstring]|)
}
```

Other than Zed, Zeek provides one of the richest data typing systems available
and therefore such records typically need no adjustment to their data types
once they've been read in as is. The
[Zed/Zeek Data Type Compatibility](Data-Type-Compatibility.md) document
provides further detail on how the rich data types in Zeek TSV map to the
equivalent [rich types in Zed](../docs/data-model/zson.md#33-primitive-values).

# Zeek NDJSON

As an alternative to the default TSV format, there are two common ways that
Zeek may instead generate logs in [NDJSON](http://ndjson.org/) format.

1. Using the [JSON Streaming Logs](https://github.com/corelight/json-streaming-logs)
   package (recommended for use with Zed)
2. Using the built-in [ASCII logger](https://docs.zeek.org/en/current/scripts/base/frameworks/logging/writers/ascii.zeek.html)
   configured with `redef LogAscii::use_json = T;`

In both cases, Zed tools such as `zq` can read these NDJSON logs automatically
as is, but with caveats.

Let's revisit the same `conn` record we just examined from the Zeek TSV
log, but now as NDJSON generated using the JSON Streaming Logs package.

#### conn.ndjson:

```mdtest-input conn.ndjson
{"_path":"conn","_write_ts":"2018-03-24T17:15:21.400275Z","ts":"2018-03-24T17:15:21.255387Z","uid":"C8Tful1TvM3Zf5x8fl","id.orig_h":"10.164.94.120","id.orig_p":39681,"id.resp_h":"10.47.3.155","id.resp_p":3389,"proto":"tcp","duration":0.004266023635864258,"orig_bytes":97,"resp_bytes":19,"conn_state":"RSTR","missed_bytes":0,"history":"ShADTdtr","orig_pkts":10,"orig_ip_bytes":730,"resp_pkts":6,"resp_ip_bytes":342}
```

#### Example:

```mdtest-command
zq -Z 'head 1' conn.ndjson
```

#### Output:
```mdtest-output
{
    _path: "conn",
    _write_ts: "2018-03-24T17:15:21.400275Z",
    ts: "2018-03-24T17:15:21.255387Z",
    uid: "C8Tful1TvM3Zf5x8fl",
    "id.orig_h": "10.164.94.120",
    "id.orig_p": 39681,
    "id.resp_h": "10.47.3.155",
    "id.resp_p": 3389,
    proto: "tcp",
    duration: 0.004266023635864258,
    orig_bytes: 97,
    resp_bytes: 19,
    conn_state: "RSTR",
    missed_bytes: 0,
    history: "ShADTdtr",
    orig_pkts: 10,
    orig_ip_bytes: 730,
    resp_pkts: 6,
    resp_ip_bytes: 342
}
```

When we compare this to the TSV example, we notice a few things right away that
all follow from the records having been previously output as JSON.

1. The timestamps like `_write_ts` and `ts` are printed as strings rather than
   the ZSON `time` type.
2. The IP addresses such as `id.orig_h` and `id.resp_h` are printed as strings
   rather than the ZSON `ip` type.
3. The connection `duration` is printed as a floating point number rather than
   the ZSON `duration` type.
4. The keys for the null-valued fields in the record read from
   TSV are not present in the record read from NDJSON.

If you're familiar with the limitations of the JSON data types, it makes sense
that Zeek chose to output these values in NDJSON as it did. Furthermore, if
you were just seeking to do quick searches on the string values or simple math
on the numbers, these limitations may be acceptable. However, if you intended
to perform operations like
[aggregations with time-based grouping](../docs/language/grouping.md#time-grouping---every)
or [CIDR matches](../docs/language/search-syntax.md#example-14)
on IP addresses, you would likely want to restore the rich Zed data types as
the records are being read. The document on [Shaping Zeek NDJSON](Shaping-Zeek-NDJSON.md)
provides details on how this can be done.

# The Role of `_path`

Zeek's `_path` field plays an important role in differentiating between its
different [log types](https://docs.zeek.org/en/master/script-reference/log-files.html)
(`conn`, `dns`, etc.) For instance,
[shaping Zeek NDJSON](Shaping-Zeek-NDJSON.md) relies on the value of
the `_path` field to know which Zed type to apply to an input NDJSON
record.

If reading Zeek TSV logs or logs generated by the JSON Streaming Logs
package, this `_path` value is provided within the Zeek logs. However, if the
log was generated by Zeek's built-in ASCII logger when using the
`redef LogAscii::use_json = T;` configuration, the value that would be used for
`_path` is present in the log _file name_ but is not in the NDJSON log
records. In this case you could adjust your Zeek configuration by following the
[Log Extension Fields example](https://docs.zeek.org/en/master/frameworks/logging.html#log-extension-fields)
from the Zeek docs. If you enter `path` in the locations where the example
shows `stream`, you will see the field named `_path` populated just like was
shown for the JSON Streaming Logs output.
