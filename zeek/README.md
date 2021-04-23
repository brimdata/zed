# JSON Type Definitions

- [Summary](#summary)
- [Contact us!](#contact-us)
- [Sample data](#sample-data)
- [Usage](#usage)
- [Why is this even necessary?](#why-is-this-even-necessary)
- [Type definition structure & importance of `_path`](#type-definition-structure--importance-of-_path)
- [Customizing type definitions](#customizing-type-definitions)
  * [Sample data](#sample-data-1)
  * [Handling exceptions](#handling-exceptions)
- [Older versions of Zeek/Bro](#older-versions-of-zeekbro)
- [Need help? Have feedback?](#need-help-have-feedback)

# Summary

The file `types.json` in this directory contains configuration
that can be used to restore Zeek-compatible rich data types to events that
have been output in JSON format by Zeek. The definitions in the current
revision of `types.json` reflect the default set of logs output by
[Zeek v3.1.2](https://github.com/zeek/zeek/releases/tag/v3.1.2) and are
likely to be compatible with default logs from older Zeek/Bro releases as
well.

If your Zeek environment is customized, you will need to adjust the type
definition to match the differences in your JSON logs. The sections below
describe the customization process.

# Contact us!

If you're using JSON type definitions with `zq` or [Brim](https://github.com/brimdata/brim),
we'd like to hear from you! We know the process of customizing the definitions
can be tricky. We have ideas for ways we might improve it, but we'll have a
better sense of priority and how to go about it if we've heard from those
who have tried it. Whether you've used this feature and it "just worked" or if
you hit challenges and need help, please join our
[public Slack](https://www.brimsecurity.com/join-slack/)
and tell us about it, or
[open an issue](https://github.com/brimdata/zed/issues/new/choose). Thanks!

# Sample data

To see a working example before working with your own data, clone the
[zed-sample-data](https://github.com/brimdata/zed-sample-data) repository.

```
# git clone --depth=1 https://github.com/brimdata/zed-sample-data ~/zed-sample-data
```

A set of JSON-format Zeek logs will now be present in `~/zed-sample-data/zeek-ndjson`.

# Usage

Assuming you have the [`zed`](https://github.com/brimdata/zed) repository locally
cloned to `~/zed` and the `zq` binary is in your `$PATH`, here's an example of
reading in all Zeek events while applying the JSON type definition:

```
# zq -f table -j ~/zed/zeek/types.json "count()" ~/zed-sample-data/zeek-ndjson/*
COUNT
1462078
```

Since we saw no errors from `zq`, that means the definitions in `types.json`
fully described all named fields in all the log files. If there had been
errors, `zq` would have stopped reading when the first error was encountered
and would not have returned a `count()` result. The sections below describe
what to do if we'd seen such errors here.

# Why is this even necessary?

Consider this Zeek HTTP event as output by the
[JSON Streaming Logs](https://github.com/corelight/json-streaming-logs)
package:

```
# gzcat ~/zed-sample-data/zeek-ndjson/http.ndjson.gz | head -n 1 | jq -S .
{
  "_path": "http",
  "_write_ts": "2018-03-24T17:15:20.610930Z",
  "host": "10.47.3.200",
  "id.orig_h": "10.164.94.120",
  "id.orig_p": 36729,
  "id.resp_h": "10.47.3.200",
  "id.resp_p": 80,
  "method": "GET",
  "request_body_len": 0,
  "resp_fuids": [
    "FnHkIl1kylqZ3O9xhg"
  ],
  "resp_mime_types": [
    "text/html"
  ],
  "response_body_len": 56,
  "status_code": 301,
  "status_msg": "Moved Permanently",
  "tags": [],
  "trans_depth": 1,
  "ts": "2018-03-24T17:15:20.609736Z",
  "uid": "CpQfkTi8xytq87HW2",
  "uri": "/chassis/config/GeneralChassisConfig.html",
  "user_agent": "Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0)",
  "version": "1.1"
}
```

To appreciate the importance of correct data typing, it helps to compare this
event with how it's described when output in Zeek's default TSV log format.
Here the schema is defined in a pair of headers at the top of the log.

```
# gzcat ~/zed-sample-data/zeek-default/http.log.gz | egrep "#fields"\|"#types"
#fields	ts	uid	id.orig_h	id.orig_p	id.resp_h	id.resp_p	trans_depth	method	host	uri	referrer	version	user_agent	origin	request_body_len	response_body_len	status_code	status_msg	info_code	info_msg	tags	username	password	proxied	orig_fuids	orig_filenames	orig_mime_types	resp_fuids	resp_filenames	resp_mime_types
#types	time	string	addr	port	addr	port	count	string	stringstring	string	string	string	string	count	count	count	string	count	string	set[enum]	string	string	set[string]	vector[string]	vector[string]	vector[string]	vector[string]	vector[string]	vector[string]
```

To the naked eye the JSON event appears descriptive. However, upon closer
examination, we see important differences compared to the schema as it
appeared in the TSV header:

* The `ts` field is of Zeek type `time`, but JSON must represent it as a string.
* Fields such as `id.orig_h` are of Zeek type `addr`, but JSON must represent these as strings.
* Fields of the Zeek `set` type must be represented as JSON arrays, indicating a significance of ordering among the elements that was not present in the original Zeek data.

Therefore, an operation such as a CIDR match would not work as expected
if the JSON event were read _without_ using `-j` to specify the data type
definition. In the following ZQL pipeline, the
[`cut` processor](../docs/language/processors#cut)
emits a warning because no events were returned from the attempted CIDR
match.

```
# zq -f table "id.orig_h =~ 10.47.0.0/16 | cut ts,id.orig_h | head 1" ~/zed-sample-data/zeek-ndjson/http.ndjson.gz
Cut fields ts,id.orig_h not present together in input
```

Because we didn't apply a typing definition (via the `-j` flag), here `zq` was performing _type_
_inference_, assigning data types that match JSON's limited types. We can see
this more clearly by having `zq` print an event back out in Zeek format after
inferring data types:

```
# zq -f zeek "cut ts,id.orig_h | head 1" ~/zed-sample-data/zeek-ndjson/http.ndjson.gz
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	ts	id.orig_h
#types	string	string
2018-03-24T17:15:20.609736Z	10.164.94.120
```

However, once we apply the the type definition, `zq` now knows to treat
`id.orig_h` as an IP address.

```
# zq -f zeek -j ~/zed/zeek/types.json "cut ts,id.orig_h | head 1" ~/zed-sample-data/zeek-ndjson/http.ndjson.gz
#separator \x09
#set_separator	,
#empty_field	(empty)
#unset_field	-
#fields	ts	id.orig_h
#types	time	addr
1521911720.609736	10.164.94.120
```

Revisiting our original query, now the CIDR match succeeds, and we see the
expected result.

```
# zq -f table -j ~/zed/zeek/types.json "id.orig_h =~ 10.47.0.0/16 | cut ts,id.orig_h | head 1" ~/zed-sample-data/zeek-ndjson/http.ndjson.gz
TS                ID.ORIG_H
1521911722.314494 10.47.5.155
```

Much the same is true if you import such JSON events into the Brim desktop
application. Zeek workflows in Brim are highly dependent on the time-ordered
nature of Zeek events, so ensuring the fields of type `time` (such as `ts`)
are correctly recognized is a prerequisite for working with the data.

For these reasons, it's important to invest a little time to ensure you have
the correct type defintions for your data.

# Type definition structure & importance of `_path`

If you examine it in a JSON browser, you'll see that the type definition in
`types.json` is structured into two parts:

1. A section of `rules` that identify each expected Zeek event. Each rule
specifies a `_path` of a unique Zeek event type, then names a corresponding
`descriptor` configuration.
2. A section of `descriptors` that define the expected the name and
[ZNG](../docs/formats/zng.md)
data type for each field in a Zeek event that was identified by a rule.

Zeek's `_path` field plays an important role in this definition. `zq` will
typically learn the `_path` for events in one of two ways:

* **Explicit** - It will be included in each JSON event, such as if the events were output by the [JSON Streaming Logs](https://github.com/corelight/json-streaming-logs)
package. This was the case for the zed-sample-data events shown above.
* **Implicit** - `zq` will deduce it from filenames, such as if the Zeek logs were output
by the built-in [ASCII logger](https://docs.zeek.org/en/current/scripts/base/frameworks/logging/writers/ascii.zeek.html)
with `redef LogAscii::use_json = T;` configured. If the files have been subject
to rotation by Zeek, `zq` will recognize the `_path` as being the leading
portion of filenames to the left of the timestamp range. So for the filenames
shown below, `zq` would infer `_path` values of `conn`, `dns`, and `stats`,
respectively:

```
# ls -l
total 24
-rw-r--r--  1 phil  wheel   739 May  2 13:18 conn.16:58:12-16:58:37.log.gz
-rw-r--r--  1 phil  wheel  1514 May  2 13:18 dns.16:58:11-16:58:37.log.gz
-rw-r--r--  1 phil  wheel   238 May  2 13:18 stats.16:58:07-16:58:37.log.gz
```

# Customizing type definitions

When reading a set of Zeek JSON logs, the exceptions that may occur typically
fall into one of two categories:

1. `descriptor not found` - An event is read that has no `_path` defined in
the `rules` portion of the type defintion
2. `incomplete descriptor` - An event is read that contains addtional fields
beyond those in the matching `descriptors` entry

Next we'll walk through an example where we handle each of these exceptions.

## Sample data

First we'll regenerate our Zeek JSON logs from the same subset of
[wrccdc 2018 pcaps](https://wrccdc.org/) described in
the [zed-sample-data README](https://github.com/brimdata/zed-sample-data/blob/main/README.md),
but with a customized Zeek v3.1.2 that has the following additional packages
installed:

* https://github.com/soelkongen/json-streaming-logs
* https://github.com/salesforce/hassh
* https://github.com/sethhall/unknown-mime-type-discovery

In addition to installing these via [Zeek Package Manager](https://docs.zeek.org/projects/package-manager/en/stable/),
the necessary configuration is present in our `local.zeek` to invoke them:

```
@load ./json-streaming-logs
@load ./hassh
@load ./unknown-mime-type-discovery
```

We merge the original pcaps together and generate logs from them using our
customized Zeek.

```
# mergecap -w wrccdc.pcap wrccdc.2018-03-24.10*.pcap
# zeek -C -r wrccdc.pcap local
```

This produces the following logs:

```
# ls json_streaming_*
json_streaming_capture_loss.1.log
json_streaming_capture_loss.log
json_streaming_conn.1.log
json_streaming_conn.2.log
json_streaming_conn.log
json_streaming_dce_rpc.1.log
json_streaming_dce_rpc.2.log
json_streaming_dns.1.log
json_streaming_dns.2.log
...
```

## Handling exceptions

Now we attempt to read the new events with `zq`, using the same type
definition as before.

```
# zq -f table -j ~/zed/zeek/types.json "count()" json_streaming_*.log
json_streaming_ssh.2.log: line 1: incomplete descriptor
```

`zq` exited as soon as it encountered a problem. We can instead use the
`-e=false` option, which will prevent `zq` from exiting. Now `zq` will read
events from each file for as long as it can successfully apply the type
definition, then for any file where there's a failure, it will exit with an
error message that references the line number where it encountered the first
problem.

```
# zq -f table -e=false -j ~/zed/zeek/types.json "count()" json_streaming_*.log
json_streaming_ssh.2.log: line 1: incomplete descriptor
json_streaming_unknown_mime_type_discovery.1.log: line 1: descriptor not found
json_streaming_unknown_mime_type_discovery.2.log: line 1: descriptor not found
json_streaming_ssh.1.log: line 4: incomplete descriptor
COUNT
1463285
```

Knowing what we do about HASSH, we understand the `incomplete descriptor` error
is because additional fields were present in the event that are not present in
the `ssh_log` entry of the `descriptors` portion of the type definition. This
was because these fields were not present in the default Zeek v3.1.2 output
that the type definition is based on.

```
# head -1 json_streaming_ssh.2.log | jq -S .
{
  "_path": "ssh",
  ...
  "hassh": "06046964c022c6407d15a27b12a6a4fb",
  "hasshAlgorithms": "curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group-exchange-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,diffie-hellman-group-exchange-sha1,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1,ext-info-c;chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com;umac-64-etm@openssh.com,umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,hmac-sha1-etm@openssh.com,umac-64@openssh.com,umac-128@openssh.com,hmac-sha2-256,hmac-sha2-512,hmac-sha1;none,zlib@openssh.com,zlib",
  "hasshServer": "b12d2871a1189eff20364cf5333619ee",
  "hasshServerAlgorithms": "curve25519-sha256,curve25519-sha256@libssh.org,ecdh-sha2-nistp256,ecdh-sha2-nistp384,ecdh-sha2-nistp521,diffie-hellman-group-exchange-sha256,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512,diffie-hellman-group14-sha256,diffie-hellman-group14-sha1;chacha20-poly1305@openssh.com,aes128-ctr,aes192-ctr,aes256-ctr,aes128-gcm@openssh.com,aes256-gcm@openssh.com;umac-64-etm@openssh.com,umac-128-etm@openssh.com,hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,hmac-sha1-etm@openssh.com,umac-64@openssh.com,umac-128@openssh.com,hmac-sha2-256,hmac-sha2-512,hmac-sha1;none,zlib@openssh.com",
  "hasshVersion": "1.1",
  ...
}
```

Likewise, there's no `unknown_mime_type_discovery` entry in the `rules`
portion of the type defintion. This is because this event type is wholly
absent from the default Zeek v3.1.2 output that the type definition is based
on.

Now that we understand the problem, we could manually add the missing
configuration to our `types.json`. But this would be tedious and error-prone.
To make this easier, a Zeek script
[`print-types.zeek`](https://github.com/brimdata/zeek/blob/master/brim/print-types.zeek)
exists that will output a complete type definition for each event type that
may be generated by Zeek. By running it in our customized Zeek installation,
we'll get a customized type definition that's unique for our environment.

Following the comments in the script, we run it while ensuring to invoke the
`local` configuration so that all customizations are included:
```
# ZEEK_ALLOW_INIT_ERRORS=1 zeek print-types.zeek local | grep descriptors | jq -S . > types-custom.json
warning in /usr/local/zeek-3.1.2/share/zeek/site/local.zeek, line 106: Loading script '__load__.bro' with legacy extension, support for '.bro' will be removed in Zeek v4.1
Skipping openflow log as it has records nested within records
```

(Note: The "Skipping openflow..." message is due to known issue [#15](https://github.com/brimdata/zeek/issues/15)
It does not affect this example. If you have logs with multiple levels of nesting, please let us know and/or comment on the issue.)

We can compare it to our original type definition to confirm we see the
expected modifications. The top set of differences describes the additional
fields that HASSH added to the `ssh` events, and the bottom two sets
describe the newly-defined `unknown_mime_type_discovery` event.

```
# diff ~/zed/zeek/types.json types-custom.json
3075a3076,3103
>         "name": "hasshVersion",
>         "type": "bstring"
>       },
>       {
>         "name": "hassh",
>         "type": "bstring"
>       },
>       {
>         "name": "hasshServer",
>         "type": "bstring"
>       },
>       {
>         "name": "cshka",
>         "type": "bstring"
>       },
>       {
>         "name": "hasshAlgorithms",
>         "type": "bstring"
>       },
>       {
>         "name": "sshka",
>         "type": "bstring"
>       },
>       {
>         "name": "hasshServerAlgorithms",
>         "type": "bstring"
>       },
>       {
3394a3423,3444
>     "unknown_mime_type_discovery_log": [
>       {
>         "name": "_path",
>         "type": "string"
>       },
>       {
>         "name": "ts",
>         "type": "time"
>       },
>       {
>         "name": "fid",
>         "type": "bstring"
>       },
>       {
>         "name": "bof",
>         "type": "bstring"
>       },
>       {
>         "name": "_write_ts",
>         "type": "time"
>       }
>     ],
3797a3848,3852
>       "descriptor": "unknown_mime_type_discovery_log",
>       "name": "_path",
>       "value": "unknown_mime_type_discovery"
>     },
>     {
```

Re-reading our data with the new custom definitions, we no longer see errors.
The higher event count reflects events that are no longer rejected by `zq`
due to the type definitions that were previously absent.

```
# zq -f table -e=false -j types-custom.json "count()" json_streaming_*.log
COUNT
1463414
```

# Older versions of Zeek/Bro

Since the `types.json` provided here is based on Zeek v3.1.2, you may wonder
if it's appropriate to use it if you're running an older version of Zeek.
While we've not tested with every version, we expect this should generally
work fine. In terms of the default outputs from Zeek, what we've observed is
that typically _more_ log types and additional fields appear in newer
versions, but they're rarely taken away or have the data types of existing
fields changed. Therefore, fields named in the `descriptors` entries that
never appear in your older Zeek/Bro logs will always have `null` values when
read into `zq`/Brim, but this is harmless. If such persistently `null` values
concern you, you could customize the `types.json` by trimming them from the
appropriate `descriptors` section. However, you may want to simply continue
using the newer `types.json` because now you're "future-proofed" for when you
eventually do upgrade to a newer Zeek version and your logs start to include
the additional fields and logs included in the type definition.

# Need help? Have feedback?

Once again, please do join our [public Slack](https://www.brimsecurity.com/join-slack/)
and let us know your experience (good or bad) so we can improve it. Thanks!
