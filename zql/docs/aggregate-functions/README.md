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
zq -f table 'count() by _path | sort' *.log.gz
```

#### Output:
```zq-output
_PATH        COUNT
capture_loss 2
rfb          3
stats        5
kerberos     11
smb_files    12
pe           21
ssh          22
dpd          25
notice       64
snmp         65
dce_rpc      78
ftp          93
modbus       129
smb_mapping  393
ntlm         422
ntp          904
smtp         1188
syslog       2378
rdp          4122
x509         10013
weird        24048
ssl          35493
dns          53615
http         144034
files        162986
conn         1021952
```

#### Example #3:

To count the data set into time-sorted 5-minute buckets:

```zq-command
zq -f table 'every 5min count() | sort ts' *.log.gz
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
zq -f table 'download_traffic=sum(resp_bytes)' conn.log.gz
```

#### Output:
```zq-output
DOWNLOAD_TRAFFIC
7017021819
```
