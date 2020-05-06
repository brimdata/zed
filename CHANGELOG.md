These entries focus on changes we think are relevant to users of Brim,
zq, or pcap.  For all changes to zqd, its API, or to other components in the
zq repo, check the git log.

## v0.13.1
* zq: Fix an issue with stream reset that was preventing the pcap button in Brim from activating (#725)
* zql: Allow multiple fields to be written from `put` processor (#697)

## v0.13.0
* zqd: Enable time indexing to provide faster query response in narrower time ranges (#647)
* zql: Make ipv4 subnet bases contain 4 octets to remove ambiguity between fractions & CIDR (#670)
* zq: Use an external sort for large inputs (removes the 10-million line `sort` limit) (#527)
* zq: Fix an issue where duplicate field names could be produced by aggregate functions & group-by (#676)
* zar: Introduce an experimental prototype for working with archived logs ([README](https://github.com/brimsec/zq/blob/master/cmd/zar/README.md)) (#700)
* zq: Support recursive record nesting in Zeek reader/writer (#715)
* zqd: Zeek log import support needed for Brim (#616, #517, #608, #592, #592, #582, #709)

## v0.12.0
* zql: Introduce `=~` and `!~` operators in filters for globs, regexps, and matching addresses against subnets (#604, #620)
* zq: When input auto-detect fails, include each attempted format's error (#616)
* zng: Binary format is now called "ZNG" and text format is called "TZNG" ("BZNG" has been retired) (#621, #630, #656)
* zql: `cut` now has a `-c` option to show all fields _not_ in the provided list (#639, #655)
* zq: Make `-f zng` (binary ZNG) the default `zq` output format, and introduce `-t` as shorthand for `-f tzng` (#654)

## v0.11.1
* zqd: Send HTTP status 200 for successful pcap search (#605)

## v0.11.0
* zql: Improve string search matching on field names (#570)
* pcap: Better handling of empty results (#572)
* zq: Introduce `-e` flag to allow for continued reads during input errors (#577)
* pcap: Allow reading of pcap files that have a capture length that exceeds the original length of the packet (#584)
* zqd: Fix an issue that was causing the histogram to draw incorrectly in Brim app (#602)

## v0.10.0

* zql: Let text searches match field names as well as values (#529)
* zql: Fix an issue where ZQL queries exceeding 255 chars caused a crash (#543)
* zql: Make searches case-insensitive by default (#536)
* Fix an issue where the Zeek reader failed to read whitespace from the rightmost column (#552)

## v0.9.0

* zql: Emit warnings from `put` processor (#477)
* zql: Add string functions (#475)
* zql: Narrow the use of `len()` to only sets/vectors, introduce new functions for string length (#485)
* zql: Add ternary conditional operator (#484)
* zqd: Add waterfall logger (#492)
* zqd: Make http shutdown more graceful (#500)
* zqd: Make space deletion cancel and await other operations (#451)

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
