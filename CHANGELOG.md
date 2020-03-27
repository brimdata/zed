## v0.8.0

* zql: add the `put` processor that adds or updates fields using a computed
  expression. (#437)
* zql: add functions for use with put, like `Math.min`, `Math.max`, and others.
  (#453, #459, #461, #472)
* zq: support reading ndjson with user supplied type information. (#441)
* Fix an issue reading pcaps with snaplen=0. (#462)

## v0.7.0

* Address ingest issues for packet captures in legacy pcap format.
* Calculate and respond with packet capture time range at the start of ingest,
  so that Brim can immediately display the space's time range.

## v0.6.0

* zq now displays warnings by default; the "-W" flag is removed, replaced by
  the "-q" for quieting warnings.
* Update license to reflect new corporate name.
* Address ingest issues for some pcapng packet captures.
* Address ingest issues for file or path names that required uri encoding.

## v0.5.0

* Support search queries during pcap ingestion.
* Improved error reporting in zqd, especially during pcap ingestion.
* Improved performance of space info api.
* zqd supports ingesting pcapng formatted packet capture files.

## v0.4.0
  
* zqd adds an endpoint to create a new empty space via post
* zqd adds an endpoint to post packet captures that are indexed and turned into Zeek logs

## v0.3.0

* zqd adds -datadir flag for space root directory.
* zqd adds -version flag.
* Add pcap command to interact with packet capture files.

## v0.2.0

* Per-platform binaries will be available as Github release assets.
* zql examples under zql/docs are now verified via `make test-heavy`.
* Negative integers and floats are accepted in zql expressions.
* Internal integer types now match the ZNG specification.
* Fixed comparisons of aliased types.

## v0.1.0

* zq moves from github.com/mccanne/zq to github.com/brimsec/zq.
* Parser and AST moved to zq repo from github.com/looky-cloud/lookytalk.
* Query language name changed to ZQL.
* ZNG specification added.

## v0.0.1

* Initial release of zq.
