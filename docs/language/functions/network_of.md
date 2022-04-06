### Function

&emsp; **network_of** &mdash; the network of an IP

### Synopsis

```
network_of(val: ip [, mask: net|int|uint]) -> net
```
### Description

The _network_of_ function returns the network of the IP address given
by `val` as determined by the optional `mask`.  If `mask` is an integer rather
than a net, it is presumed to be a network prefix of the indicated length.
If `mask` is ommited, then a class A (8 bit), B (16 bit), or C (24 bit)
network is inferred from `val`, which in this case, must be an IPv4 address.

### Examples

Compute the network address of an IP given a network mask argument:
```mdtest-command
echo '10.1.2.129' | zq -z 'yield network_of(this, 255.255.255.128/25)' -
```
=>
```mdtest-output
10.1.2.128/25
```
Compute the network address of an IP given an integer prefix argument:
```mdtest-command
echo '10.1.2.129' | zq -z 'yield network_of(this, 25)' -
```
=>
```mdtest-output
10.1.2.128/25
```

Compute the network address implied by IP classful addressing:
```mdtest-command
echo '10.1.2.129' | zq -z 'yield network_of(this)' -
```
=>
```mdtest-output
10.0.0.0/8
```

The network of a value that is not an IP is an error:
```mdtest-command
echo 1 | zq -z 'yield network_of(this)' -
```
=>
```mdtest-output
error("network_of: not an IP")
```
