## v0.4.0
  
* zqd adds an endpoint to create a new empty space via POST
* zqd adds an endpoint to post packet captures that are indexed passed through Zeek

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
