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
