# Aggregate Functions

Comprehensive documentation for ZQL aggrgeate functions is still a work in
progress. In the meantime, here's a few examples to get started with.

#### Example #1:

To count how many events are in the sample data set:

```zq-command
zq -f table 'count()' *.log.gz
```

#### Output:
```zq-output
COUNT
1462078
```

#### Example #2:

To count how many events there are of each Zeek log type in the sample data
set:

```zq-command
zq -f table 'count() by _path' *.log.gz
```

#### Output:
```zq-output
_PATH        COUNT
pe           21
dns          53615
dpd          25
ftp          93
ntp          904
rdp          4122
rfb          3
ssh          22
ssl          35493
conn         1021952
http         144034
ntlm         422
smtp         1188
snmp         65
x509         10013
files        162986
stats        5
weird        24048
modbus       129
notice       64
syslog       2378
dce_rpc      78
kerberos     11
smb_files    12
smb_mapping  393
capture_loss 2
```

#### Example #3:

To count the time-sorted data set into 5-minute buckets:

```zq-command
zq -f table 'sort ts | every 5min count()' *.log.gz
```

#### Output:
```zq-output
TS                COUNT
1521911700.000000 441229
1521912000.000000 337264
1521912300.000000 310546
1521912600.000000 274284
1521912900.000000 98755
```

#### Example #4:

To calculate the total of `resp_bytes` values across all `conn` events and save
the result in a field called `download_traffic`:

```zq-command
zq -f table 'sum(resp_bytes) as download_traffic' conn.log.gz
```

#### Output:
```zq-output
DOWNLOAD_TRAFFIC
7017021819
```
