### Function

&emsp; **cidr_match** &mdash; test if IP is in a network

### Synopsis

```
cidr_match(mask: net, val: any) -> bool
```
### Description

The _cidr_match_ function returns true if `val` contains an IP address that
falls within the network given by `mask`.  When `val` is a complex type, the
function traverses its nested structured to find any network values.
If `mask` is not type `net`, then an error is returned.

### Examples

Test whether values are IP addresses in a network:
```mdtest-command
echo '10.1.2.129 11.1.2.129 10 "foo"' | zq -z 'yield cidr_match(10.0.0.0/8, this)' -
```
=>
```mdtest-output
true
false
false
false
```
It also works for IPs in nested values:
echo '[10.1.2.129,11.1.2.129] {a:10.0.0.1} {a:11.0.0.1}' | zq -z 'yield cidr_match(10.0.0.0/8, this)' -
```
=>
```mdtest-output
true
true
false
```

The first argument must be a network:
```
echo '10.0.0.1' | zq -z 'yield cidr_match([1,2,3], this)' -
```
=>
```
error("cidr_match: not a net: [1,2,3]")
```
