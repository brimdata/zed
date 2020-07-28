# `pcap`

The pcap command indexes and slices pcap files.  Use pcap to create a time index for a large pcap, then derive
smaller pcaps by efficiently extracting subsets of packets from the large pcap using time range and flow filter
arguments.  The pcap command was inspired by Vern Paxson's tcpslice program written in the early 1990's.  However,
tcpslice does not work with the more sophisticated pcap-ng file format and does not properly handle pcaps with
out-of-order timestamps.

For all `pcap` options, see the built-in help by running:

```
pcap help
```
