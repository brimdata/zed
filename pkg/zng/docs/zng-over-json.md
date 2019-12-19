# zson over http

TBD: this is just an example for now.  We'll clean this up later...

## Nested Example

This zson...

```
#1:record[id:record[orig_h:addr,orig_p:port,resp_h:addr,resp_p:port],etc:string]
1:[[192.168.1.1;80;192.168.2.22;6060;]foo;]
1:[[192.168.1.2;80;192.168.2.21;5050;]bar;]
1:[[192.168.1.3;80;192.168.2.20;4040;]hello;]
```

produces this zson over json...

```
{
  "id": 0,
  "type": [
    {
      "name": "id",
      "type": [
        {
          "name": "orig_h",
          "type": "addr"
        },
        {
          "name": "orig_p",
          "type": "port"
        },
        {
          "name": "resp_h",
          "type": "addr"
        },
        {
          "name": "resp_p",
          "type": "port"
        }
      ]
    },
    {
      "name": "etc",
      "type": "string"
    }
  ],
  "value": [
    [
      "192.168.1.1",
      "80",
      "192.168.2.22",
      "6060"
    ],
    "foo"
  ]
}
{
  "id": 0,
  "value": [
    [
      "192.168.1.2",
      "80",
      "192.168.2.21",
      "5050"
    ],
    "bar"
  ]
}
{
  "id": 0,
  "value": [
    [
      "192.168.1.3",
      "80",
      "192.168.2.20",
      "4040"
    ],
    "hello"
  ]
}
```


# Flat Example - conn, dns

## ZSON

```
#0:record[_path:string,ts:time,uid:string,id.orig_h:addr,id.orig_p:port,id.resp_h:addr,id.resp_p:port,proto:enum,service:string,duration:interval,orig_bytes:count,resp_bytes:count,conn_state:string,local_orig:bool,local_resp:bool,missed_bytes:count,history:string,orig_pkts:count,orig_ip_bytes:count,resp_pkts:count,resp_ip_bytes:count,tunnel_parents:set[string]]
0:[conn;1425565514.419939;CogZFI3py5JsFZGik;fe80::eef4:bbff:fe51:89ec;5353;ff02::fb;5353;udp;dns;15.007272;2148;0;S0;F;F;0;D;14;2820;0;0;[]]
#1:record[_path:string,ts:time,uid:string,id.orig_h:addr,id.orig_p:port,id.resp_h:addr,id.resp_p:port,proto:enum,trans_id:count,rtt:interval,query:string,qclass:count,qclass_name:string,qtype:count,qtype_name:string,rcode:count,rcode_name:string,AA:bool,TC:bool,RD:bool,RA:bool,Z:count,answers:vector[string],TTLs:vector[interval],rejected:bool]
1:[dns;1425565514.419939;CogZFI3py5JsFZGik;fe80::eef4:bbff:fe51:89ec;5353;ff02::fb;5353;udp;0;0.119484;_ipp._tcp.local;1;C_INTERNET;12;PTR;0;NOERROR;T;F;F;F;0;[_workstation._tcp.local;sniffer [ec:f4:bb:51:89:ec]._workstation._tcp.local;][4500.000000;4500.000000;]F;]
0:[conn;1425565545.440378;CoJHyyAGilEHkRZJf;fe80::eef4:bbff:fe51:89ec;5353;ff02::fb;5353;udp;dns;-;-;-;S0;F;F;0;D;1;93;0;0;[]]
```

## ZSON over JSON

The output below was produced by running
```
zq -f zjson input.zson | jq
```
This is pretty-printed with jq, but would otherwise be newline-delimited JSON:
```
{
  "id": 0,
  "type": [
    {
      "name": "_path",
      "type": "string"
    },
    {
      "name": "ts",
      "type": "time"
    },
    {
      "name": "uid",
      "type": "string"
    },
    {
      "name": "id.orig_h",
      "type": "addr"
    },
    {
      "name": "id.orig_p",
      "type": "port"
    },
    {
      "name": "id.resp_h",
      "type": "addr"
    },
    {
      "name": "id.resp_p",
      "type": "port"
    },
    {
      "name": "proto",
      "type": "enum"
    },
    {
      "name": "service",
      "type": "string"
    },
    {
      "name": "duration",
      "type": "interval"
    },
    {
      "name": "orig_bytes",
      "type": "count"
    },
    {
      "name": "resp_bytes",
      "type": "count"
    },
    {
      "name": "conn_state",
      "type": "string"
    },
    {
      "name": "local_orig",
      "type": "bool"
    },
    {
      "name": "local_resp",
      "type": "bool"
    },
    {
      "name": "missed_bytes",
      "type": "count"
    },
    {
      "name": "history",
      "type": "string"
    },
    {
      "name": "orig_pkts",
      "type": "count"
    },
    {
      "name": "orig_ip_bytes",
      "type": "count"
    },
    {
      "name": "resp_pkts",
      "type": "count"
    },
    {
      "name": "resp_ip_bytes",
      "type": "count"
    },
    {
      "name": "tunnel_parents",
      "type": "set[string]"
    }
  ],
  "value": [
    "conn",
    "1425565514.419939",
    "CogZFI3py5JsFZGik",
    "fe80::eef4:bbff:fe51:89ec",
    "5353",
    "ff02::fb",
    "5353",
    "udp",
    "dns",
    "15.007272",
    "2148",
    "0",
    "S0",
    "F",
    "F",
    "0",
    "D",
    "14",
    "2820",
    "0",
    "0",
    null
  ]
}
{
  "id": 1,
  "type": [
    {
      "name": "_path",
      "type": "string"
    },
    {
      "name": "ts",
      "type": "time"
    },
    {
      "name": "uid",
      "type": "string"
    },
    {
      "name": "id.orig_h",
      "type": "addr"
    },
    {
      "name": "id.orig_p",
      "type": "port"
    },
    {
      "name": "id.resp_h",
      "type": "addr"
    },
    {
      "name": "id.resp_p",
      "type": "port"
    },
    {
      "name": "proto",
      "type": "enum"
    },
    {
      "name": "trans_id",
      "type": "count"
    },
    {
      "name": "rtt",
      "type": "interval"
    },
    {
      "name": "query",
      "type": "string"
    },
    {
      "name": "qclass",
      "type": "count"
    },
    {
      "name": "qclass_name",
      "type": "string"
    },
    {
      "name": "qtype",
      "type": "count"
    },
    {
      "name": "qtype_name",
      "type": "string"
    },
    {
      "name": "rcode",
      "type": "count"
    },
    {
      "name": "rcode_name",
      "type": "string"
    },
    {
      "name": "AA",
      "type": "bool"
    },
    {
      "name": "TC",
      "type": "bool"
    },
    {
      "name": "RD",
      "type": "bool"
    },
    {
      "name": "RA",
      "type": "bool"
    },
    {
      "name": "Z",
      "type": "count"
    },
    {
      "name": "answers",
      "type": "vector[string]"
    },
    {
      "name": "TTLs",
      "type": "vector[interval]"
    },
    {
      "name": "rejected",
      "type": "bool"
    }
  ],
  "value": [
    "dns",
    "1425565514.419939",
    "CogZFI3py5JsFZGik",
    "fe80::eef4:bbff:fe51:89ec",
    "5353",
    "ff02::fb",
    "5353",
    "udp",
    "0",
    "0.119484",
    "_ipp._tcp.local",
    "1",
    "C_INTERNET",
    "12",
    "PTR",
    "0",
    "NOERROR",
    "T",
    "F",
    "F",
    "F",
    "0",
    [
      "_workstation._tcp.local",
      "sniffer [ec:f4:bb:51:89:ec]._workstation._tcp.local"
    ],
    [
      "4500.000000",
      "4500.000000"
    ],
    "F"
  ]
}
{
  "id": 0,
  "value": [
    "conn",
    "1425565545.440378",
    "CoJHyyAGilEHkRZJf",
    "fe80::eef4:bbff:fe51:89ec",
    "5353",
    "ff02::fb",
    "5353",
    "udp",
    "dns",
    "",
    "",
    "",
    "S0",
    "F",
    "F",
    "0",
    "D",
    "1",
    "93",
    "0",
    "0",
    null
  ]
}
```
