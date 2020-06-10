# Search Syntax

  * [Search all events](#search-all-events)
  * [Value Match](#value-match)
    + [Bare Word](#bare-word)
    + [Quoted Word](#quoted-word)
    + [Glob Wildcards](#glob-wildcards)
    + [Regular Expressions](#regular-expressions)
  * [Field/Value Match](#fieldvalue-match)
    + [Role of Data Types](#role-of-data-types)
    + [Exact, Glob, and Regexp Matches](#exact-glob-and-regexp-matches)
    + [Comparisons](#comparisons)
    + [Wildcard Field Names](#wildcard-field-names)
    + [Other Examples](#other-examples)
  * [Boolean Operators](#boolean-operators)
    + [`and`](#and)
    + [`or`](#or)
    + [`not`](#not)
    + [Parentheses & Order of Evaluation](#parentheses--order-of-evaluation)

## Search all events

The simplest possible ZQL search is a match against all events. This search is expressed in `zq` with the wildcard `*`. The response will be a ZNG-formatted dump of all events. The default `zq` format is binary ZNG, a compact format that's ideal for working in pipelines. However, in these docs we'll sometimes make use of the `-t` option to output the text-based TZNG format, which is readable at the command line.

#### Example:
```zq-command
zq -t '*' conn.log.gz
```

#### Output:
```zq-output head:6
#zenum=string
#0:record[_path:string,ts:time,uid:bstring,id:record[orig_h:ip,orig_p:port,resp_h:ip,resp_p:port],proto:zenum,service:bstring,duration:duration,orig_bytes:uint64,resp_bytes:uint64,conn_state:bstring,local_orig:bool,local_resp:bool,missed_bytes:uint64,history:bstring,orig_pkts:uint64,orig_ip_bytes:uint64,resp_pkts:uint64,resp_ip_bytes:uint64,tunnel_parents:set[bstring]]
0:[conn;1521911721.255387;C8Tful1TvM3Zf5x8fl;[10.164.94.120;39681;10.47.3.155;3389;]tcp;-;0.004266;97;19;RSTR;-;-;0;ShADTdtr;10;730;6;342;-;]
0:[conn;1521911721.411148;CXWfTK3LRdiuQxBbM6;[10.47.25.80;50817;10.128.0.218;23189;]tcp;-;0.000486;0;0;REJ;-;-;0;Sr;2;104;2;80;-;]
0:[conn;1521911721.926018;CM59GGQhNEoKONb5i;[10.47.25.80;50817;10.128.0.218;23189;]tcp;-;0.000538;0;0;REJ;-;-;0;Sr;2;104;2;80;-;]
0:[conn;1521911722.690601;CuKFds250kxFgkhh8f;[10.47.25.80;50813;10.128.0.218;27765;]tcp;-;0.000546;0;0;REJ;-;-;0;Sr;2;104;2;80;-;]
...
```

If the ZQL argument is left out entirely, this wildcard is the default search. The following shorthand command line would produce the same output shown above.

```
zq conn.log.gz
```

To start a ZQL pipeline with this default search, you can similarly leave out the leading `* |` before invoking your first [processor](#../processors/README.md) or [aggregate function](#../aggregate-functions/README.md).

#### Example #1:
```zq-command
zq -t 'cut server_tree_name' ntlm.log.gz # Shorthand for: zq '* | cut server_tree_name' ntlm.log.gz
```

#### Output:
```zq-output head:4
#0:record[server_tree_name:bstring]
0:[factory.oompa.loompa;]
0:[factory.oompa.loompa;]
0:[jerry.land;]
...
```

#### Example #2:
```zq-command
zq -t 'count() by _path' *.log.gz  # Shorthand for: zq '* | count() by _path' *.log.gz
```

#### Output:
```zq-output head:4
#0:record[_path:string,count:uint64]
0:[pe;21;]
0:[dns;53615;]
0:[dpd;25;]
...
```

## Value Match

The search result can be narrowed to include only events that contain certain values in any field(s).

### Bare Word

The simplest form of such a search is a "bare" word (not wrapped in quotes), which will match against any field value that contains the word as a substring.

For example, searching across all our logs for `10.150.0.85` matches against events that contain `addr`-type fields containing this precise value (fields such as `tx_hosts` and `id.resp_h` in our sample data) and also where it appears within `string`-type fields (such as the field `certificate.subject` in `x509` events.)

* **Note**: In this and many following examples, we'll use the `zq -f table` output format for human readability. Due to the width of the Zeek events used as sample data, you may need to "scroll right" in the output to see some matching field values.

#### Example:
```zq-command
zq -f table '10.150.0.85' *.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P PROTO SERVICE DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY   ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521911722.187980 CFis4J1xm9BOgtib34 10.47.8.10   56800     10.150.0.85 443       tcp   -       1.000534 31         77         SF         -          -          0            ^dtAfDTFr 8         382           10        554           -
conn  1521911725.527535 CnvVUp1zg3fnDKrlFk 10.47.27.186 58665     10.150.0.85 443       tcp   -       1.000958 31         77         SF         -          -          0            ^dtAfDFTr 8         478           10        626           -
conn  1521911727.167552 CsSFJyH4ucFtpmhqa  10.10.18.2   57331     10.150.0.85 443       tcp   -       1.000978 31         77         SF         -          -          0            ^dtAfDFTr 8         478           10        626           -
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P VERSION CIPHER                                CURVE  SERVER_NAME RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS   CLIENT_CERT_CHAIN_FUIDS SUBJECT                                                      ISSUER                                                       CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521911732.513518 Ckwqsn2ZSiVGtyiFO5 10.47.24.186 55782     10.150.0.85 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 x25519 -           F       -          h2            T           FZW30y2Nwc9i0qmdvg (empty)                 CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU -              -             self signed certificate
_PATH TS                FUID               TX_HOSTS    RX_HOSTS     CONN_UIDS          SOURCE DEPTH ANALYZERS     MIME_TYPE                    FILENAME DURATION LOCAL_ORIG IS_ORIG SEEN_BYTES TOTAL_BYTES MISSING_BYTES OVERFLOW_BYTES TIMEDOUT PARENT_FUID MD5                              SHA1                                     SHA256 EXTRACTED EXTRACTED_CUTOFF EXTRACTED_SIZE
files 1521911732.519299 FZW30y2Nwc9i0qmdvg 10.150.0.85 10.47.24.186 Ckwqsn2ZSiVGtyiFO5 SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL CERTIFICATE.SUBJECT                                          CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911732.519299 FZW30y2Nwc9i0qmdvg 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
...
```

By comparison, the section below on [Field/Value Match](#fieldvalue-match) describes ways to perform searches against only fields of a specific [data type](../data-types/README.md).

### Quoted Word

Sometimes you may need to search for sequences of multiple words or words that contain special characters. To achieve this, wrap your search term in quotes.

Let's say we want to isolate the events containing the text `O=Internet Widgits` that we saw in the response to the previous example search. If typed "bare" as our ZQL, we'd experience two problems:

   1. The leading `O=` would be interpreted as the start of an attempted [field/value match](#fieldvalue-match) for a field named `O`.
   2. The space character would cause the input to be interpreted as two separate words and hence the search would not be as strict.

However, wrapping in quotes gives the desired result.

#### Example:
```zq-command
zq -f table '"O=Internet Widgits"' *.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P VERSION CIPHER                                CURVE  SERVER_NAME RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS   CLIENT_CERT_CHAIN_FUIDS SUBJECT                                                      ISSUER                                                       CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521911732.513518 Ckwqsn2ZSiVGtyiFO5 10.47.24.186 55782     10.150.0.85 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 x25519 -           F       -          h2            T           FZW30y2Nwc9i0qmdvg (empty)                 CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU -              -             self signed certificate
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL CERTIFICATE.SUBJECT                                          CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911732.519299 FZW30y2Nwc9i0qmdvg 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
_PATH  TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P FUID               FILE_MIME_TYPE FILE_DESC PROTO NOTE                     MSG                                                              SUB                                                          SRC          DST         P   N PEER_DESCR ACTIONS            SUPPRESS_FOR REMOTE_LOCATION.COUNTRY_CODE REMOTE_LOCATION.REGION REMOTE_LOCATION.CITY REMOTE_LOCATION.LATITUDE REMOTE_LOCATION.LONGITUDE
notice 1521911732.521729 Ckwqsn2ZSiVGtyiFO5 10.47.24.186 55782     10.150.0.85 443       FZW30y2Nwc9i0qmdvg -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (self signed certificate) CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 10.47.24.186 10.150.0.85 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
_PATH TS                UID                ID.ORIG_H  ID.ORIG_P ID.RESP_H   ID.RESP_P VERSION CIPHER                                CURVE  SERVER_NAME RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS  CLIENT_CERT_CHAIN_FUIDS SUBJECT                                                      ISSUER                                                       CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521911742.629228 CqwJmZ2Lzd42fuvg4k 10.47.8.10 56802     10.150.0.85 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 x25519 -           F       -          h2            T           Fo9ltu1O8DGE0KAgC (empty)                 CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU -              -             self signed certificate
_PATH TS                ID                CERTIFICATE.VERSION CERTIFICATE.SERIAL CERTIFICATE.SUBJECT                                          CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911742.635094 Fo9ltu1O8DGE0KAgC 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
...
```

### Glob Wildcards

To find values that may contain arbitrary substrings between or alongside the desired word(s), one or more [glob](https://en.wikipedia.org/wiki/Glob_(programming))-style wildcards can be used.

For example, the following search finds events that contain web server hostnames that include the letters `cdn` in the middle of them, such as `www.cdn.amazon.com` or `www.herokucdn.com`.

#### Example:
```zq-command
zq -f table 'www.*cdn*.com' *.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY              QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                                                                                                                                                                                                                                                                                      TTLS                        REJECTED
dns   1521911784.038839 ChS4MN2D9iPNzSwAw4 10.47.2.154 59353     10.0.0.100 53        udp   11089    0.000785 www.amazon.com     1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.cdn.amazon.com,d3ag4hukkh62yn.cloudfront.net,54.192.139.227                                                                                                                                                                                                                                                                              578,57,57                   F
dns   1521911784.038843 ChS4MN2D9iPNzSwAw4 10.47.2.154 59353     10.0.0.100 53        udp   11089    0.000784 www.amazon.com     1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.cdn.amazon.com,d3ag4hukkh62yn.cloudfront.net,54.192.139.227                                                                                                                                                                                                                                                                              578,57,57                   F
dns   1521911784.038845 ChS4MN2D9iPNzSwAw4 10.47.2.154 59353     10.0.0.100 53        udp   15749    0.001037 www.amazon.com     1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  T  0 www.cdn.amazon.com,d3ag4hukkh62yn.cloudfront.net                                                                                                                                                                                                                                                                                             578,57                      F
dns   1521911784.038847 ChS4MN2D9iPNzSwAw4 10.47.2.154 59353     10.0.0.100 53        udp   15749    0.001039 www.amazon.com     1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  T  0 www.cdn.amazon.com,d3ag4hukkh62yn.cloudfront.net                                                                                                                                                                                                                                                                                             578,57                      F
dns   1521911829.930694 Cfah1k4TTqKPt2tUNa 10.47.1.10  54657     10.0.0.100 53        udp   50394    0.001135 www.cdn.amazon.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 d3ag4hukkh62yn.cloudfront.net,54.192.139.227                                                                                                                                                                                                                                                                                                 12,12                       F
dns   1521911829.930698 Cfah1k4TTqKPt2tUNa 10.47.1.10  54657     10.0.0.100 53        udp   50394    0.001133 www.cdn.amazon.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 d3ag4hukkh62yn.cloudfront.net,54.192.139.227                                                                                                                                                                                                                                                                                                 12,12                       F
dns   1521912177.049941 CiCGyr4RPOcBLVyh33 10.47.2.100 39482     10.0.0.100 53        udp   27845    0.014268 www.herokucdn.com  1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 d3v17f49c4gdd3.cloudfront.net,52.85.83.228,52.85.83.238,52.85.83.247,52.85.83.110,52.85.83.12,52.85.83.97,52.85.83.135,52.85.83.215                                                                                                                                                                                                          300,60,60,60,60,60,60,60,60 F
dns   1521912177.049944 CiCGyr4RPOcBLVyh33 10.47.2.100 39482     10.0.0.100 53        udp   27845    0.014269 www.herokucdn.com  1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 d3v17f49c4gdd3.cloudfront.net,52.85.83.228,52.85.83.238,52.85.83.247,52.85.83.110,52.85.83.12,52.85.83.97,52.85.83.135,52.85.83.215                                                                                                                                                                                                          300,60,60,60,60,60,60,60,60 F
dns   1521912177.049945 CiCGyr4RPOcBLVyh33 10.47.2.100 39482     10.0.0.100 53        udp   13966    0.017272 www.herokucdn.com  1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  T  0 d3v17f49c4gdd3.cloudfront.net,2600:9000:201d:8a00:15:5f5a:e9c0:93a1,2600:9000:201d:3600:15:5f5a:e9c0:93a1,2600:9000:201d:b400:15:5f5a:e9c0:93a1,2600:9000:201d:2400:15:5f5a:e9c0:93a1,2600:9000:201d:a00:15:5f5a:e9c0:93a1,2600:9000:201d:ba00:15:5f5a:e9c0:93a1,2600:9000:201d:f200:15:5f5a:e9c0:93a1,2600:9000:201d:1800:15:5f5a:e9c0:93a1 300,60,60,60,60,60,60,60,60 F
...
```

   * **Note:** Our use of `*` to [search all events](#search-all-events) as shown previously is the simplest example of using a glob wildcard.

Glob wildcards only have effect when used with [bare word](#bare-word) searches. An asterisk in a [quoted word](#quoted-word) search will match explicitly against an asterisk character. For example, the following search matches events that contain the substring `CN=*` as is often found in the start of certificate subjects.

#### Example:
```zq-command
zq -f table '"CN=*"' *.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL               CERTIFICATE.SUBJECT                                                                                  CERTIFICATE.ISSUER                                    CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS                                     SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911723.174330 FQ290u35UG0B05Zky9 3                   017E45A31AA50BC35053BC50F9B69BAD CN=*.services.mozilla.com,OU=Cloud Services,O=Mozilla Corporation,L=Mountain View,ST=California,C=US CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US 1507014000.000000            1578513600.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.services.mozilla.com,services.mozilla.com -       -         -      F                    -
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H      ID.RESP_P VERSION CIPHER                                  CURVE     SERVER_NAME                RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS                      CLIENT_CERT_CHAIN_FUIDS SUBJECT                                                                                              ISSUER                                                       CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521911723.363645 Ck6KyHTvFSs6ilQ43  10.47.26.160 49161     216.58.193.195 443       TLSv12  TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 x25519    fonts.gstatic.com          F       -          h2            T           FPxVI11Qp4XsZx8MIf,F287wP3LNxC1jQJZsb (empty)                 CN=*.google.com,O=Google Inc,L=Mountain View,ST=California,C=US                                      CN=Google Internet Authority G3,O=Google Trust Services,C=US -              -             ok
ssl   1521911723.363999 CdREh1wNA3vUhNI1f  10.47.26.160 49162     216.58.193.195 443       TLSv12  TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 x25519    fonts.gstatic.com          F       -          h2            T           FWz7sY1pnCwl9NvQe,FJ469V1AfRW24KDwBc  (empty)                 CN=*.google.com,O=Google Inc,L=Mountain View,ST=California,C=US                                      CN=Google Internet Authority G3,O=Google Trust Services,C=US -              -             ok
ssl   1521911723.375960 CYVobu3DR0JyyP1m3g 10.47.26.160 49163     216.58.193.195 443       TLSv12  TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 x25519    ssl.gstatic.com            F       -          h2            T           F8iNVI29EYGgwvRa6j,FADPVCnp9r9OThjk9  (empty)                 CN=*.google.com,O=Google Inc,L=Mountain View,ST=California,C=US                                      CN=Google Internet Authority G3,O=Google Trust Services,C=US -              -             ok
ssl   1521911723.139892 CmkwsI9pQSw1nyclk  10.47.1.208  50083     52.40.133.43   443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256   secp256r1 tiles.services.mozilla.com F       -          -             T           FQ290u35UG0B05Zky9,Fx8Cg11p5utkG9q2G7 (empty)                 CN=*.services.mozilla.com,OU=Cloud Services,O=Mozilla Corporation,L=Mountain View,ST=California,C=US CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US        -              -             ok
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL               CERTIFICATE.SUBJECT                                                           CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911723.393858 FWz7sY1pnCwl9NvQe  3                   7AEE77D0AA874D3A                 CN=*.google.com,O=Google Inc,L=Mountain View,ST=California,C=US               CN=Google Internet Authority G3,O=Google Trust Services,C=US 1520481815.000000            1527731580.000000           id-ecPublicKey      sha256WithRSAEncryption ecdsa                256                    -                    prime256v1        *.google.com,*.android.com,*.appengine.google.com,*.cloud.google.com,*.db833953.google.cn,*.g.co,*.gcp.gvt2.com,*.google-analytics.com,*.google.ca,*.google.cl,*.google.co.in,*.google.co.jp,*.google.co.uk,*.google.com.ar,*.google.com.au,*.google.com.br,*.google.com.co,*.google.com.mx,*.google.com.tr,*.google.com.vn,*.google.de,*.google.es,*.google.fr,*.google.hu,*.google.it,*.google.nl,*.google.pl,*.google.pt,*.googleadapis.com,*.googleapis.cn,*.googlecommerce.com,*.googlevideo.com,*.gstatic.cn,*.gstatic.com,*.gvt1.com,*.gvt2.com,*.metric.gstatic.com,*.urchin.com,*.url.google.com,*.youtube-nocookie.com,*.youtube.com,*.youtubeeducation.com,*.yt.be,*.ytimg.com,android.clients.google.com,android.com,developer.android.google.cn,developers.android.google.cn,g.co,goo.gl,google-analytics.com,google.com,googlecommerce.com,source.android.google.cn,urchin.com,www.goo.gl,youtu.be,youtube.com,youtubeeducation.com,yt.be -       -         -      F                    -
x509  1521911723.394013 FPxVI11Qp4XsZx8MIf 3                   7AEE77D0AA874D3A                 CN=*.google.com,O=Google Inc,L=Mountain View,ST=California,C=US               CN=Google Internet Authority G3,O=Google Trust Services,C=US 1520481815.000000            1527731580.000000           id-ecPublicKey      sha256WithRSAEncryption ecdsa                256                    -                    prime256v1        *.google.com,*.android.com,*.appengine.google.com,*.cloud.google.com,*.db833953.google.cn,*.g.co,*.gcp.gvt2.com,*.google-analytics.com,*.google.ca,*.google.cl,*.google.co.in,*.google.co.jp,*.google.co.uk,*.google.com.ar,*.google.com.au,*.google.com.br,*.google.com.co,*.google.com.mx,*.google.com.tr,*.google.com.vn,*.google.de,*.google.es,*.google.fr,*.google.hu,*.google.it,*.google.nl,*.google.pl,*.google.pt,*.googleadapis.com,*.googleapis.cn,*.googlecommerce.com,*.googlevideo.com,*.gstatic.cn,*.gstatic.com,*.gvt1.com,*.gvt2.com,*.metric.gstatic.com,*.urchin.com,*.url.google.com,*.youtube-nocookie.com,*.youtube.com,*.youtubeeducation.com,*.yt.be,*.ytimg.com,android.clients.google.com,android.com,developer.android.google.cn,developers.android.google.cn,g.co,goo.gl,google-analytics.com,google.com,googlecommerce.com,source.android.google.cn,urchin.com,www.goo.gl,youtu.be,youtube.com,youtubeeducation.com,yt.be -       -         -      F                    -
...
```

### Regular Expressions

For matching that requires more precision than can be achieved with [glob wildcards](#glob-wildcards), regular expressions (regexps) are also available. To use them, simply place a `/` character before and after the regexp.

For example, let's say you'd already done a [glob wildcard](#glob-wildcard) search for `www.*google*.com` and found events that reference the following hostnames:

```
www.google-analytics.com
www.google.com
www.googleadservices.com
www.googleapis.com
www.googlecommerce.com
www.googletagmanager.com
www.googletagservices.com
```

But if you're only interested in events having to do with "ad" or "tag" services, the following regexp search can accomplish this.

#### Example:
```zq-command
zq -f table '/www.google(ad|tag)services.com/' *.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                     QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                             TTLS     REJECTED
dns   1521911746.074840 CYjLXM1Yp1ZuuVJQV1 10.47.6.154 12478     10.10.6.1  53        udp   49089    0.001342 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0        F
dns   1521911746.074842 CYjLXM1Yp1ZuuVJQV1 10.47.6.154 12478     10.10.6.1  53        udp   49089    0.001375 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0        F
dns   1521911746.078050 Cn1BpA2bKVzWn7IvVe 10.47.6.154 38992     10.10.6.1  53        udp   14171    0.000262 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0        F
dns   1521911746.078051 Cn1BpA2bKVzWn7IvVe 10.47.6.154 38992     10.10.6.1  53        udp   14171    0.000265 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0        F
dns   1521911746.078071 CtUHnV2nyFWejoYQ23 10.47.6.154 48071     10.10.6.1  53        udp   64736    0.009286 www.googletagservices.com 1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  F  0 pagead46.l.doubleclick.net,2607:f8b0:4007:804::2002 44266,53 F
dns   1521911746.078072 CtUHnV2nyFWejoYQ23 10.47.6.154 48071     10.10.6.1  53        udp   64736    0.009287 www.googletagservices.com 1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  F  0 pagead46.l.doubleclick.net,2607:f8b0:4007:804::2002 44266,53 F
dns   1521911769.132486 CUsIaD4CHJDv2dMpp  10.47.7.10  51674     10.0.0.100 53        udp   12049    0.00132  www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0        F
dns   1521911769.132491 CUsIaD4CHJDv2dMpp  10.47.7.10  51674     10.0.0.100 53        udp   12049    0.001316 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0        F
dns   1521911777.181981 CfofM11rhswW1NDNS  10.47.7.10  52373     10.0.0.100 53        udp   61544    0.000881 www.googleadservices.com  1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0        F
...
```

Regexps are a detailed topic all their own. For details, reference the [documentation for re2](https://github.com/google/re2/wiki/Syntax), which is the library that `zq` uses to provide regexp support.

## Field/Value Match

The search result can be narrowed to include only events that contain a certain value in a particular named field. For example, the following search will only match events containing the field called `uid` where it is set to the precise value `ChhAfsfyuz4n2hFMe`.

#### Example:
```zq-command
zq -f table 'uid=ChhAfsfyuz4n2hFMe' *.log.gz
```

#### Output:

```zq-output
_PATH TS                UID               ID.ORIG_H    ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO SERVICE DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521912990.158539 ChhAfsfyuz4n2hFMe 10.239.34.35 56602     10.47.6.51 873       tcp   -       0.000004 0          0          S0         -          -          0            S       2         88            0         0             -
```

### Role of Data Types

When working with named fields, the data type of the field comes becomes significant in two ways.

1. To match successfully, the value entered must be comparable to the data type of the named field. For instance, since `id.resp_h` is typically an `addr`-type field, an attempted field/value match `id.resp_h=10.150.0.999` will return an error, since this is not valid IP address syntax.

2.  The correct operator must be chosen based on whether the field type is primitive or complex.  For example, `id.resp_h=10.150.0.85` will match in our sample data because `id.resp_h` is a primitive type, `addr`. However, to check if the same IP had been a transmitting host in a `files` event, the syntax `10.150.0.85 in tx_hosts` would be used because `tx_hosts` is a complex type, `set[addr]`.

See the [Data Types](./data-types/README.md) page for more details on types and the operators for working with them.

### Exact, Glob, and Regexp Matches

An important distinction is that a "bare" field/value match is treated as an _exact_ match. If we take one of the results from our [bare word value match](#bare-word) example and attempt to look for `Widgits`, but only on a field named `certificate.subject`, there will be no matches. This is because `Widgits` only happens to appear as a _substring_ of `certificate.subject` fields in our sample data.

#### Example:
```zq-command
zq -f table 'certificate.subject=Widgits' *.log.gz         # Produces no output
```
```zq-output
```

To achieve this with a field/value match, we can use [glob wildcards](#glob-wildcards). Because this is not testing for strict equality, here we use the pattern matching operator (`=~`) between the field name and value.

#### Example:
```zq-command
zq -f table 'certificate.subject=~*Widgits*' *.log.gz
```

#### Output:

```zq-output head:5
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL CERTIFICATE.SUBJECT                                          CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521911732.519299 FZW30y2Nwc9i0qmdvg 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  1521911742.635094 Fo9ltu1O8DGE0KAgC  3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  1521911746.548292 F7oQQK1qo9HfmlN048 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  1521911747.493786 FdBWBA3eODh6nHFt82 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 1521728557.000000            2385642000.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
...
```

[Regular expressions](#regular-expressions) can also be used with the `=~` operator in field/value matches.

### Comparisons

In addition to testing for equality and pattern matching via `=` and `=~`, other common comparison operators `!=`, `<`, `>`, `<=`, and `=>` are also available.

For example, the following search finds connections that have transferred many bytes.

#### Example:
```zq-command
zq -f table 'orig_bytes > 1000000' *.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521912315.208232 CVimRo24ubbKqFvNu7 172.30.255.1 11        10.128.0.207 0         icmp  -       100.721937  1647088    0          OTH        -          -          0            -                44136     2882896       0         0             -
conn  1521911720.630818 CO0MhB2NCc08xWaly8 10.47.1.154  49814     134.71.3.17  443       tcp   -       1269.512465 1618740    12880888   OTH        -          -          0            ^dtADTatTtTtTtT  110169    7594230       111445    29872050      -
conn  1521911720.637761 Cmgywj2O8KZAHHjddb 10.47.1.154  49582     134.71.3.17  443       tcp   -       1266.367457 1594682    53255700   OTH        -          -          0            ^dtADTatTtTtTtTW 131516    8407458       142488    110641641     -
conn  1521911720.705347 CWtQuI2IMNyE1pX47j 10.47.6.161  52121     134.71.3.17  443       tcp   -       1269.320626 2267243    54791018   OTH        -          -          0            DTadtATttTtTtT   152819    10575303      158738    113518994     -
conn  1521912785.415532 Cy3R5w2pfv8oSEpa2j 10.47.8.19   49376     10.128.0.214 443       tcp   -       202.457994  4862366    1614249    S1         -          -          0            ShAdtttDTaTTTt   7280      10015980      6077      3453020       -
```

The same operators also work when comparing characters in `string`-type values, such as this search that finds DNS requests that were issued for hostnames at the high end of the alphabet.

#### Example:
```zq-command
zq -f table 'query > zippy' *.log.gz
```

#### Output:
```zq-output
_PATH TS                UID               ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                                                    QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                TTLS       REJECTED
dns   1521912609.841740 Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001694 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   1521912609.841742 Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001697 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   1521912892.637234 CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019491 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
dns   1521912892.637238 CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019493 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
```

### Wildcard Field Names

Since the data type of the value is considered in field/value matches, it's possible to search for the value across any fields of the value's type by entering a wildcard (`*`) in place of the field name.

For example, the following search finds many `ssl` and `conn` events that contain the value `10.150.0.85` in an `addr`-type field such as `id.resp_h`. Compare this with our [bare word](#bare-word) example where it also matched as a substring of the `string`-type field named `certificate.subject`.

#### Example:
```zq-command
zq -f table '*=10.150.0.85' *.log.gz
```

#### Output:
```zq-output
_PATH  TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P FUID               FILE_MIME_TYPE FILE_DESC PROTO NOTE                     MSG                                                              SUB                                                          SRC          DST         P   N PEER_DESCR ACTIONS            SUPPRESS_FOR REMOTE_LOCATION.COUNTRY_CODE REMOTE_LOCATION.REGION REMOTE_LOCATION.CITY REMOTE_LOCATION.LATITUDE REMOTE_LOCATION.LONGITUDE
notice 1521911732.521729 Ckwqsn2ZSiVGtyiFO5 10.47.24.186 55782     10.150.0.85 443       FZW30y2Nwc9i0qmdvg -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (self signed certificate) CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 10.47.24.186 10.150.0.85 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
```

Similarly, the following search will only match when the value appears in a complex field of type `set[addr]` or `array[addr]`, such as `tx_hosts` in this case.

#### Example:
```zq-command
zq -f table '10.150.0.85 in *' *.log.gz
```

#### Output:
```zq-output head:5
_PATH TS                FUID               TX_HOSTS    RX_HOSTS     CONN_UIDS          SOURCE DEPTH ANALYZERS     MIME_TYPE                    FILENAME DURATION LOCAL_ORIG IS_ORIG SEEN_BYTES TOTAL_BYTES MISSING_BYTES OVERFLOW_BYTES TIMEDOUT PARENT_FUID MD5                              SHA1                                     SHA256 EXTRACTED EXTRACTED_CUTOFF EXTRACTED_SIZE
files 1521911732.519299 FZW30y2Nwc9i0qmdvg 10.150.0.85 10.47.24.186 Ckwqsn2ZSiVGtyiFO5 SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 1521911742.635094 Fo9ltu1O8DGE0KAgC  10.150.0.85 10.47.8.10   CqwJmZ2Lzd42fuvg4k SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 1521911746.548292 F7oQQK1qo9HfmlN048 10.150.0.85 10.47.27.186 CvTTHG2M6xPqDMDLB7 SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 1521911747.493786 FdBWBA3eODh6nHFt82 10.150.0.85 10.10.18.2   ChpfSB4FWhg3xHI3yb SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
...
```

### Other Examples

The other behaviors we described previously for general [value matching](#value-match) still apply the same for field/value matches. Below are some exercises you can try to observe this with the sample data. Search with `zq` against `*.log.gz` in all cases.

1. Compare the result of our previous [quoted word](#quoted-word) value search for `"O=Internet Widgits"` with a field/value search for `certificate.subject=*Widgits*`. Note how the former showed many types of Zeek events while the latter shows _only_ `x509` events, since only these events contain the field named `certificate.subject`.
2. Compare the result of our previous [glob wildcard](#glob-wildcards) value search for `www.*cdn*.com` with a field/value search for `server_name=www.*cdn*.com`. Note how the former showed mostly Zeek `dns` events and a couple `ssl` events, while the latter shows _only_ `ssl` events, since only these events contain the field named `server_name`.
3. Compare the result of our previous [regexp](#regular-expressions) value search for `/www.google(ad|tag)services.com/` with a field/value search for `query=/www.google(ad|tag)services.com/`. Note how the former showed a mix of Zeek `dns` and `ssl` events, while the latter shows _only_ `dns` events, since only these events contain the field named `query`.

## Boolean Operators

Your searches can be further refined by using boolean operators `and`, `or`, and `not`. These operators are case-insensitive, so `AND`, `OR`, and `NOT` can also be used.

### `and`

If you enter multiple [value match](#value-match) or [field/value match](#fieldvalue-match) terms separated by blank space, ZQL implicitly applies a boolean `and` between them, such that events are only returned if they match on _all_ terms.

For example, when introducing [glob wildcard](#glob-wildcards), we performed a search for `www.*cdn*.com` that returned mostly `dns` events along with a couple `ssl` events. You could quickly isolate just the the SSL events by leveraging this implicit `and`.

#### Example:
```zq-command
zq -f table 'www.*cdn*.com _path=ssl' *.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P VERSION CIPHER                                CURVE     SERVER_NAME       RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS                                                            CLIENT_CERT_CHAIN_FUIDS SUBJECT            ISSUER                                  CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521912180.244457 CUG0fiQAzL4rNWxai  10.47.2.100 36150     52.85.83.228 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FXKmyTbr7HlvyL1h8,FADhCTvkq1ILFnD3j,FoVjYR16c3UIuXj4xk,FmiRYe1P53KOolQeVi   (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
ssl   1521912240.189735 CSbGJs3jOeB6glWLJj 10.47.7.154 27137     52.85.83.215 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FuW2cZ3leE606wXSia,Fu5kzi1BUwnF0bSCsd,FyTViI32zPvCmNXgSi,FwV6ff3JGj4NZcVPE4 (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
```

* **Note**: You may also include the `and` operator explicitly if you wish:

        www.*cdn*.com and _path=ssl

### `or`

The `or` operator returns the union of the matches from multiple terms.

For example, we can revisit two of our previous example searches that each only returned a few events, searching now with `or` to see them all at once.

#### Example:
```zq-command
zq -f table 'orig_bytes > 1000000 or query > zippy' *.log.gz
```

#### Output:

```zq-output head:10
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION   ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521912315.208232 CVimRo24ubbKqFvNu7 172.30.255.1 11        10.128.0.207 0         icmp  -       100.721937 1647088    0          OTH        -          -          0            -       44136     2882896       0         0             -
_PATH TS                UID               ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                                                    QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                TTLS       REJECTED
dns   1521912609.841740 Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001694 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   1521912609.841742 Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001697 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   1521912892.637234 CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019491 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
dns   1521912892.637238 CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019493 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521911720.630818 CO0MhB2NCc08xWaly8 10.47.1.154 49814     134.71.3.17  443       tcp   -       1269.512465 1618740    12880888   OTH        -          -          0            ^dtADTatTtTtTtT  110169    7594230       111445    29872050      -
conn  1521911720.637761 Cmgywj2O8KZAHHjddb 10.47.1.154 49582     134.71.3.17  443       tcp   -       1266.367457 1594682    53255700   OTH        -          -          0            ^dtADTatTtTtTtTW 131516    8407458       142488    110641641     -
...
```

### `not`

Use the `not` operator to invert the matching logic of everything to the right of it in your search expression.

For example, suppose you've noticed that the vast majority of the sample Zeek events are of log types like `conn`, `dns`, `files`, etc. You could review some of the less-common Zeek event types by inverting the logic of a [regexp match](#regular-expressions).

#### Example:
```zq-command
zq -f table 'not _path=~/conn|dns|files|ssl|x509|http|weird/' *.log.gz
```

#### Output:

```zq-output head:10
_PATH TS                PEER MEM PKTS_PROC BYTES_RECV PKTS_DROPPED PKTS_LINK PKT_LAG EVENTS_PROC EVENTS_QUEUED ACTIVE_TCP_CONNS ACTIVE_UDP_CONNS ACTIVE_ICMP_CONNS TCP_CONNS UDP_CONNS ICMP_CONNS TIMERS ACTIVE_TIMERS FILES ACTIVE_FILES DNS_REQUESTS ACTIVE_DNS_REQUESTS REASSEM_TCP_SIZE REASSEM_FILE_SIZE REASSEM_FRAG_SIZE REASSEM_UNKNOWN_SIZE
stats 1521911720.600725 zeek 74  26        29375      -            -         -       404         11            1                0                0                 1         0         0          36     32            0     0            0            0                   1528             0                 0                 0
_PATH  TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P FUID               FILE_MIME_TYPE FILE_DESC PROTO NOTE                     MSG                                                                             SUB                                                                                                                                                                                                            SRC           DST         P   N PEER_DESCR ACTIONS            SUPPRESS_FOR REMOTE_LOCATION.COUNTRY_CODE REMOTE_LOCATION.REGION REMOTE_LOCATION.CITY REMOTE_LOCATION.LATITUDE REMOTE_LOCATION.LONGITUDE
notice 1521911720.629574 C9zBQP1nnfBHxUTEY1 10.164.94.120 39611     10.47.3.200 443       FYNFkU3KccxXgIuUg5 -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (unable to get local issuer certificate) unstructuredName=1315656901\\,564d7761726520496e632e,CN=localhost.localdomain,emailAddress=ssl-certificates@vmware.com,OU=VMware ESX Server Default Certificate,O=VMware\\, Inc,L=Palo Alto,ST=California,C=US 10.164.94.120 10.47.3.200 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
notice 1521911720.788325 C4kACn2RY2rQd0keMe 10.164.94.120 42545     10.47.8.200 443       FW8nz6IQ4FHNxgyVg  -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (unable to get local issuer certificate) unstructuredName=1315656901\\,564d7761726520496e632e,CN=localhost.localdomain,emailAddress=ssl-certificates@vmware.com,OU=VMware ESX Server Default Certificate,O=VMware\\, Inc,L=Palo Alto,ST=California,C=US 10.164.94.120 10.47.8.200 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
notice 1521911720.921208 CNBo0M1CKShxFq4N38 10.164.94.120 43551     10.47.27.80 443       FNKiW53te1DL8dclxd -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (self signed certificate)                CN=www.example.com,OU=Certificate generated at installation time,O=Bitnami                                                                                                                                     10.164.94.120 10.47.27.80 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
_PATH TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P PROTO ANALYZER FAILURE_REASON
dpd   1521911721.155638 CYGOnV3BIdoiWKveXg 10.164.94.120 36171     10.47.8.218 80        tcp   HTTP     not a http request line
_PATH TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P COOKIE RESULT    SECURITY_PROTOCOL CLIENT_CHANNELS KEYBOARD_LAYOUT CLIENT_BUILD CLIENT_NAME CLIENT_DIG_PRODUCT_ID DESKTOP_WIDTH DESKTOP_HEIGHT REQUESTED_COLOR_DEPTH CERT_TYPE CERT_COUNT CERT_PERMANENT ENCRYPTION_LEVEL ENCRYPTION_METHOD
rdp   1521911721.258458 C8Tful1TvM3Zf5x8fl 10.164.94.120 39681     10.47.3.155 3389      -      encrypted HYBRID            -               -               -            -           -                     -             -              -                     -         0          -              -                -
...
```

* **Note**: The `!` operator can also be used as alternative shorthand:

        zq -f table '! _path=~/conn|dns|files|ssl|x509|http|weird/' *.log.gz

### Parentheses & Order of Evaluation

Unless wrapped in parentheses, a search expression is evaluated in _left-to-right order_.

For example, the following search leverages the implicit boolean `and` to find all `smb_mapping` events in which the `share_type` field is set to a value other than `DISK`.

#### Example:
```zq-command
zq -f table '_path=smb_mapping not share_type=DISK' *.log.gz
```

#### Output:
```zq-output head:5
_PATH       TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 1521911721.625534 ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911722.021668 C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911724.619169 C2byFA2Y10G1GLUXgb 10.164.94.120 35337     10.47.27.80  445       \\\\PC-NEWMAN\\IPC$      -       -                  PIPE
smb_mapping 1521911725.562072 C3kUnM2kEJZnvZmSp7 10.164.94.120 45903     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      -       -                  PIPE
...
```

If we change the order of the terms to what's shown below, now we match almost every event we have. This is due to the left-to-right evaluation: Since the `not` comes first, it inverts the logic of _everything that comes after it_, hence giving us all stored events _other than_ `smb_mapping` events that have the value of their `share_type` field set to `DISK`.

#### Example:
```zq-command
zq -f table 'not share_type=DISK _path=smb_mapping' *.log.gz
```

#### Output:
```zq-output head:9
_PATH TS                PEER MEM PKTS_PROC BYTES_RECV PKTS_DROPPED PKTS_LINK PKT_LAG EVENTS_PROC EVENTS_QUEUED ACTIVE_TCP_CONNS ACTIVE_UDP_CONNS ACTIVE_ICMP_CONNS TCP_CONNS UDP_CONNS ICMP_CONNS TIMERS ACTIVE_TIMERS FILES ACTIVE_FILES DNS_REQUESTS ACTIVE_DNS_REQUESTS REASSEM_TCP_SIZE REASSEM_FILE_SIZE REASSEM_FRAG_SIZE REASSEM_UNKNOWN_SIZE
stats 1521911720.600725 zeek 74  26        29375      -            -         -       404         11            1                0                0                 1         0         0          36     32            0     0            0            0                   1528             0                 0                 0
_PATH TS                UID               ID.ORIG_H   ID.ORIG_P ID.RESP_H      ID.RESP_P NAME                          ADDL NOTICE PEER
weird 1521911720.600843 C1zOivgBT6dBmknqk 10.47.1.152 49562     23.217.103.245 80        TCP_ack_underflow_or_misorder -    F      zeek
weird 1521911720.608108 -                 -           -         -              -         truncated_header              -    F      zeek
_PATH TS                UID               ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P TRANS_DEPTH METHOD HOST        URI                                       REFERRER VERSION USER_AGENT                                                      ORIGIN REQUEST_BODY_LEN RESPONSE_BODY_LEN STATUS_CODE STATUS_MSG        INFO_CODE INFO_MSG TAGS    USERNAME PASSWORD PROXIED ORIG_FUIDS ORIG_FILENAMES ORIG_MIME_TYPES RESP_FUIDS         RESP_FILENAMES RESP_MIME_TYPES
http  1521911720.609736 CpQfkTi8xytq87HW2 10.164.94.120 36729     10.47.3.200 80        1           GET    10.47.3.200 /chassis/config/GeneralChassisConfig.html -        1.1     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                56                301         Moved Permanently -         -        (empty) -        -        -       -          -              -               FnHkIl1kylqZ3O9xhg -              text/html
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P NAME                             ADDL NOTICE PEER
weird 1521911720.610033 C45Ff03lESjMQQQej1 10.47.5.155 40712     91.189.91.23 80        above_hole_data_without_any_acks -    F      zeek
...
```

Terms wrapped in parentheses along with their operators will be evaluated _first_, overriding the default left-to-right evaluation.

For example, we can rewrite our reordered search as shown below to restore its logic to that of the original.

#### Example:
```zq-command
zq -f table '(not share_type=DISK) _path=smb_mapping' *.log.gz
```

#### Output:

```zq-output head:5
_PATH       TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 1521911721.625534 ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911722.021668 C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911724.619169 C2byFA2Y10G1GLUXgb 10.164.94.120 35337     10.47.27.80  445       \\\\PC-NEWMAN\\IPC$      -       -                  PIPE
smb_mapping 1521911725.562072 C3kUnM2kEJZnvZmSp7 10.164.94.120 45903     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      -       -                  PIPE
...
```

Parentheses can also be nested.

#### Example:
```zq-command
zq -f table '((not share_type=DISK) and (service=IPC)) _path=smb_mapping' *.log.gz
```

#### Output:
```zq-output head:5
_PATH       TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 1521911721.625534 ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911722.021668 C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 1521911731.475945 Cvaqhu3VhuXlDOMgXg 10.164.94.120 37127     10.47.3.151  445       \\\\COTTONCANDY4\\IPC$   IPC     -                  PIPE
smb_mapping 1521911736.306275 CsZ7Be4NlqaJSNNie4 10.164.94.120 33921     10.47.23.166 445       \\\\PARKINGGARAGE\\IPC$  IPC     -                  PIPE
...
```

Except when writing the most common searches that leverage only the implicit `and`, it's generally good practice to use parentheses even when not strictly necessary, just to make sure your queries clearly communicate their intended logic.
