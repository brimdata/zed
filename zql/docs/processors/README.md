# Processors

A pipeline may contain one or more _processors_ to filter or transform event data. You can imagine the data flowing left-to-right through a processor, with its functionality further determined by arguments you may set. Processor names are case-insensitive.

The following available processors are documented in detail below:

* [`cut`](#cut)
* [`filter`](#filter)
* [`head`](#head)
* [`put`](#put)
* [`sort`](#sort)
* [`tail`](#tail)
* [`uniq`](#uniq)

**Note**: In the examples below, we'll use the `zq -f table` output format for human readability. Due to the width of the Zeek events used as sample data, you may need to "scroll right" in the output to see some field values.

---

## `cut`

|                           |                                                             |
| ------------------------- | ----------------------------------------------------------- |
| **Description**           | Return the data only from the specified named fields.       |
| **Syntax**                | `cut <field-list>`                                          |
| **Required<br>arguments** | `<field-list>`<br>One or more comma-separated field names.  |
| **Optional<br>arguments** | None                                                        |
| **Caveats**               | The specified field names must exist in the input data. If a non-existent field appears in the `<field-list>`, the returned results will be empty. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Cut            |

#### Example:

To return only the `ts` and `uid` columns of `conn` events:

```zq-command
zq -f table '* | cut ts,uid' conn.log.gz
```

#### Output:
```zq-output head:4
TS                UID
1521911721.255387 C8Tful1TvM3Zf5x8fl
1521911721.411148 CXWfTK3LRdiuQxBbM6
1521911721.926018 CM59GGQhNEoKONb5i
...
```

---

## `filter`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Apply a search expression to potentially trim data from the pipeline. |
| **Syntax**                | `filter <search-expression>`                                          |
| **Required<br>arguments** | `<search-expression>`<br>Any valid expression in ZQL [search syntax](../search-syntax/README.md) |
| **Optional<br>arguments** | None                                                                  |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Filter                   |

#### Example #1:

To further trim the data returned in our [`cut`](#cut) example:

```zq-command
zq -f table '* | cut ts,uid | filter uid=CXWfTK3LRdiuQxBbM6' conn.log.gz
```

#### Output:
```zq-output
TS                UID
1521911721.411148 CXWfTK3LRdiuQxBbM6
```

#### Example #2:

An alternative syntax for our [`and` operator example](#../search-syntax/README.md#and):

```zq-command
zq -f table '* | filter www.*cdn*.com _path=ssl' *.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P VERSION CIPHER                                CURVE     SERVER_NAME       RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS                                                            CLIENT_CERT_CHAIN_FUIDS SUBJECT            ISSUER                                  CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   1521912180.244457 CUG0fiQAzL4rNWxai  10.47.2.100 36150     52.85.83.228 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FXKmyTbr7HlvyL1h8,FADhCTvkq1ILFnD3j,FoVjYR16c3UIuXj4xk,FmiRYe1P53KOolQeVi   (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
ssl   1521912240.189735 CSbGJs3jOeB6glWLJj 10.47.7.154 27137     52.85.83.215 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FuW2cZ3leE606wXSia,Fu5kzi1BUwnF0bSCsd,FyTViI32zPvCmNXgSi,FwV6ff3JGj4NZcVPE4 (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
```

---

## `head`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Return only the first N events.                                       |
| **Syntax**                | `head [N]`                                                            |
| **Required<br>arguments** | None. If no arguments are specified, only the first event is returned.| 
| **Optional<br>arguments** | `[N]`<br>An integer specifying the number of results to return. If not specified, defaults to `1`. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Head                     |

#### Example #1:

To see the first `dns` event:

```zq-command
zq -f table '* | head' dns.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT     QUERY          QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                        TTLS       REJECTED
dns   1521911720.865716 C2zK5f13SbCtKcyiW5 10.47.1.100 41772     10.0.0.100 53        udp   36329    0.00087 ise.wrccdc.org 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 ise.wrccdc.cpp.edu,134.71.3.16 2230,41830 F
```

#### Example #2:

To see the first five `conn` events with activity on port `80`:

```zq-command
zq -f table ':80 | head 5' conn.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P PROTO SERVICE DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY   ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521911720.602122 C4RZ6d4r5mJHlSYFI6 10.164.94.120 33299     10.47.3.200 80        tcp   -       0.003077 0          235        RSTO       -          -          0            ^dtfAR    4         208           4         678           -
conn  1521911720.606178 CnKmhv4RfyAZ3fVc8b 10.164.94.120 36125     10.47.3.200 80        tcp   -       0.000002 0          0          RSTOS0     -          -          0            R         2         104           0         0             -
conn  1521911720.604325 C65IMkEAWNlE1f6L8  10.164.94.120 45941     10.47.3.200 80        tcp   -       0.002708 0          242        RSTO       -          -          0            ^dtfAR    4         208           4         692           -
conn  1521911720.607031 CpQfkTi8xytq87HW2  10.164.94.120 36729     10.47.3.200 80        tcp   http    0.006238 325        263        RSTO       -          -          0            ShADTdftR 10        1186          6         854           -
conn  1521911720.607695 CpjMvj2Cvj048u6bF1 10.164.94.120 39169     10.47.3.200 80        tcp   http    0.007139 315        241        RSTO       -          -          0            ShADTdtfR 10        1166          6         810           -
```

---

## `put`

|                           |                                                 |
| ------------------------- | ----------------------------------------------- |
| **Description**           | Add/update fields based on the results of a computed expression |
| **Syntax**                | `put <field> = <expression>`                    |
| **Required arguments**    | `<field>` Field into which the computed value will be stored.<br>`<expression>` A valid ZQL expression (XXX citation needed) |
| **Optional arguments**    | None |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Put |

#### Example #1:

Compute a `total_bytes` field in `conn` records:

```zq-command
zq -q -f table 'put total_bytes = orig_bytes + resp_bytes | top 10 total_bytes | cut id, orig_bytes, resp_bytes, total_bytes' conn.log.gz
```

#### Output:
```zq-output
ID.ORIG_H     ID.ORIG_P ID.RESP_H       ID.RESP_P ORIG_BYTES RESP_BYTES TOTAL_BYTES
10.47.7.154   27300     52.216.132.61   443       859        1781771107 1781771966
10.164.94.120 33691     10.47.3.200     80        355        1543916493 1543916848
10.47.8.100   37110     128.101.240.215 80        16398      376626606  376643004
10.47.3.151   11120     198.255.68.110  80        392        274063633  274064025
10.47.1.155   56594     198.255.68.110  80        392        274063633  274064025
10.47.5.155   40736     91.189.91.23    80        22706      141877415  141900121
10.47.5.155   40726     91.189.91.23    80        23204      120752961  120776165
10.47.8.100   37126     52.216.162.155  443       1213       105936976  105938189
10.47.8.100   37127     52.216.162.155  443       1494       105936280  105937774
10.47.5.155   40728     91.189.91.23    80        21396      98972170   98993566
```

---

## `sort`

|                           |                                                                           |
| ------------------------- | ------------------------------------------------------------------------- |
| **Description**           | Sort events based on the order of values in the specified named field(s). | 
| **Syntax**                | `sort [-r] [-limit N] [-nulls first\|last] [field-list]`                   |
| **Required<br>arguments** | None                                                                      |
| **Optional<br>arguments** | `[-r]`<br>If specified, results will be sorted in reverse order.<br><br>`[-limit N]`<br>The maximum number of events that may be sorted at once. If not specified, defaults to `1000000`. Note that increasing the `limit` to a very large value may cause high memory consumption.<br><br>`[-nulls first\|last]`<br>Specifies whether null values (i.e., values that are unset or that are not present at all in an incoming record) should be placed in the output.<br><br>`[field-list]`<br>One or more comma-separated field names by which to sort. Results will be sorted based on the values of the first field named in the list, then based on values in the second field named in the list, and so on.<br><br>If no field list is provided, sort will automatically pick a field by which to sort. The pick is done by examining the first result returned and finding the first field in left-to-right order of one of the following [data types](../data-types/README.md). If no fields of the first data type are found, the next is considered, and so on:<br>- `count`<br>- `int`<br>- `double`<br>If no fields of those types are found, sorting will be performed on the first field found in left-to-right order that is _not_ of the `time` data type. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Sort                         |

#### Example #1:

To sort `x509` events by `certificate.subject`:

```zq-command
zq -f table 'sort certificate.subject' x509.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL                     CERTIFICATE.SUBJECT                                                                               CERTIFICATE.ISSUER                                                                                                                                       CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521912578.233315 Fn2Gkp2Qd434JylJX9 3                   CB11D05B561B4BB1                       C=/C=US/ST=HI/O=Goldner and Sons/OU=1080p/CN=goldner.sons.net/emailAddress=1080p@goldner.sons.net C=/C=US/ST=HI/O=Goldner and Sons/OU=1080p/CN=goldner.sons.net/emailAddress=1080p@goldner.sons.net                                                        1462788542.000000            1525860542.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            -       -         -      T                    -
x509  1521911928.524223 Fxq7P31K2FS3v7CBSh 3                   031489479BCD9C116EA7B6162E5E68E6       CN=*.adnxs.com,O=AppNexus\\, Inc.,L=New York,ST=New York,C=US                                     CN=DigiCert ECC Secure Server CA,O=DigiCert Inc,C=US                                                                                                     1516867200.000000            1548446400.000000           id-ecPublicKey      ecdsa-with-SHA256       ecdsa                256                    -                    prime256v1        *.adnxs.com,adnxs.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        -       -         -      F                    -
x509  1521911928.524679 F6WWPk3ajsHLrmNFdb 3                   031489479BCD9C116EA7B6162E5E68E6       CN=*.adnxs.com,O=AppNexus\\, Inc.,L=New York,ST=New York,C=US                                     CN=DigiCert ECC Secure Server CA,O=DigiCert Inc,C=US                                                                                                     1516867200.000000            1548446400.000000           id-ecPublicKey      ecdsa-with-SHA256       ecdsa                256                    -                    prime256v1        *.adnxs.com,adnxs.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        -       -         -      F                    -
x509  1521912580.661204 FEMo0JLdFfaiP3cCj  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912580.664443 Fx9w2e3ZeGeRVzB7wa 3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912580.971149 Fs71N02K3C48z0W8Rl 3                   08C2D95B922842FCD7EEC9C4AF3BB3C1       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912580.972007 FNfnZ84jkUdb1ELG4e 3                   08C2D95B922842FCD7EEC9C4AF3BB3C1       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912581.350977 FE774oxbdOCDlPx0i  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912581.351155 FQNOg4tbfGapYl4A7  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
...
```

#### Example #2:

Now we'll sort `x509` events first by `certificate.subject`, then by the `id`. Compared to the previous example, note how this changes the order of some events that had the same `certificate.subject` value.

```zq-command
zq -f table 'sort certificate.subject,id' x509.log.gz
```

#### Output:
```zq-output head:10
_PATH TS                ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL                     CERTIFICATE.SUBJECT                                                                               CERTIFICATE.ISSUER                                                                                                                                       CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                      SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  1521912578.233315 Fn2Gkp2Qd434JylJX9 3                   CB11D05B561B4BB1                       C=/C=US/ST=HI/O=Goldner and Sons/OU=1080p/CN=goldner.sons.net/emailAddress=1080p@goldner.sons.net C=/C=US/ST=HI/O=Goldner and Sons/OU=1080p/CN=goldner.sons.net/emailAddress=1080p@goldner.sons.net                                                        1462788542.000000            1525860542.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            -       -         -      T                    -
x509  1521911928.524679 F6WWPk3ajsHLrmNFdb 3                   031489479BCD9C116EA7B6162E5E68E6       CN=*.adnxs.com,O=AppNexus\\, Inc.,L=New York,ST=New York,C=US                                     CN=DigiCert ECC Secure Server CA,O=DigiCert Inc,C=US                                                                                                     1516867200.000000            1548446400.000000           id-ecPublicKey      ecdsa-with-SHA256       ecdsa                256                    -                    prime256v1        *.adnxs.com,adnxs.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        -       -         -      F                    -
x509  1521911928.524223 Fxq7P31K2FS3v7CBSh 3                   031489479BCD9C116EA7B6162E5E68E6       CN=*.adnxs.com,O=AppNexus\\, Inc.,L=New York,ST=New York,C=US                                     CN=DigiCert ECC Secure Server CA,O=DigiCert Inc,C=US                                                                                                     1516867200.000000            1548446400.000000           id-ecPublicKey      ecdsa-with-SHA256       ecdsa                256                    -                    prime256v1        *.adnxs.com,adnxs.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        -       -         -      F                    -
x509  1521912591.670293 F0hybM3L5RvvQnB0Af 3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912591.670418 F7QTmz23i9Wb9PxCec 3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912590.367386 FAquaM1YmnRYGrPM0j 3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912581.350977 FE774oxbdOCDlPx0i  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912580.661204 FEMo0JLdFfaiP3cCj  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
x509  1521912591.317347 FMITm2OyLT3OYnfq3  3                   068D4086AEB3472996E5DFA2EC521A41       CN=*.adobe.com,OU=IS,O=Adobe Systems Incorporated,L=San Jose,ST=California,C=US                   CN=DigiCert SHA2 Secure Server CA,O=DigiCert Inc,C=US                                                                                                    1515139200.000000            1546718400.000000           rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 *.adobe.com                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  -       -         -      F                    -
...
```

#### Example #3:

Here we'll find which originating IP addresses generated the most `conn` events using the `count()` [aggregate function](../aggregate-functions/README.md) and piping its output to a `sort` in reverse order. Note that even though we didn't list a field name as an explicit argument, the `sort` processor did what we wanted because it found a field of the `count` [data type](../data-types/README.md).

```zq-command
zq -f table 'count() by id.orig_h | sort -r' conn.log.gz
```

#### Output:
```zq-output head:5
ID.ORIG_H                COUNT
10.174.251.215           279014
10.47.24.81              162237
10.47.26.82              153056
10.224.110.133           67320
...
```

#### Example #4:

In this example we count the number of times each distinct username appears in `http` records, but deliberately put the unset username at the front of the list:

```zq-command
zq -f table 'count() by username | sort -nulls first username' http.log.gz
```

#### Output:
```zq-output
USERNAME     COUNT
-            139175
M32318       4854
agloop       1
cbucket      1
mteavee      1
poompaloompa 1
wwonka       1
```


#### Example #5:

Here we have more `conn` events than the defaults would let us sort, so we increase the limit.

```zq-command
zq -f table 'sort -limit 9999999 ts' conn.log.gz
```

#### Output:

```zq-output head:5
_PATH TS                UID                ID.ORIG_H      ID.ORIG_P ID.RESP_H      ID.RESP_P PROTO SERVICE  DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521911720.600725 C1zOivgBT6dBmknqk  10.47.1.152    49562     23.217.103.245 80        tcp   -        9.698493    0          90453565   SF         -          -          0            ^dtAttttFf       57490     2358856       123713    185470730     -
conn  1521911720.600800 CfbnHCmClhWXY99ui  10.128.0.207   13        10.47.19.254   14        icmp  -        0.001278    336        0          OTH        -          -          0            -                28        1120          0         0             -
conn  1521911720.601310 CD3zwQ1YDr4XiQzO1e 10.128.0.207   59777     10.47.28.6     443       tcp   -        0.000002    0          0          S0         -          -          0            S                2         88            0         0             -
conn  1521911720.601314 CL31Wl4WQoDATEz5Z8 10.164.94.120  34261     10.47.8.208    3389      tcp   -        0.004093    128        19         RSTRH      -          -          0            ^dtADTr          4         464           4         222           -
...
```

---

## `tail`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Return only the last N events.                                        |
| **Syntax**                | `tail [N]`                                                            |
| **Required<br>arguments** | None. If no arguments are specified, only the last event is returned. | 
| **Optional<br>arguments** | `[N]`<br>An integer specifying the number of results to return. If not specified, defaults to `1`. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Tail                     |

#### Example #1:

To see the last `dns` event:

```zq-command
zq -f table '* | tail' dns.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H ID.RESP_P PROTO TRANS_ID RTT QUERY           QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS TTLS REJECTED
dns   1521912990.151237 C0ybvu4HG3yWv6H5cb 172.31.255.5 60878     10.0.0.1  53        udp   36243    -   talk.google.com 1      C_INTERNET  1     A          -     -          F  F  T  F  0 -       -    F
```

#### Example #2:

To see the last five `conn` events with activity on port `80`:

```zq-command
zq -f table ':80 | tail 5' conn.log.gz
```

#### Output:
```zq-output
_PATH TS                UID                ID.ORIG_H      ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION  ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY    ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  1521912803.087149 CqPl942ft1MCpuNQgk 10.218.221.240 63812     10.47.2.20   80        tcp   -       15.607782 0          0          S1         -          -          0            Sh         2         88            10        440           -
conn  1521912985.557756 CKCuBO2N2sY6m8qkv6 10.128.0.247   30549     10.47.22.65  80        tcp   http    0.006639  334        271        SF         -          -          0            ShADTftFa  10        1092          6         806           -
conn  1521912920.422826 Cy1XB41BipfyCcCGVh 10.128.0.247   30487     10.47.2.58   80        tcp   http    68.309996 21249      15506      S1         -          -          0            ShADTadtTt 242       52202         270       41836         -
conn  1521912664.953409 CMxwGp14TBAF3QtEq  10.219.216.224 56004     10.47.24.186 80        tcp   -       31.235313 0          0          S1         -          -          0            Sh         2         88            12        528           -
conn  1521912988.752765 COICgc1FXHKteyFy67 10.0.0.227     61314     10.47.5.58   80        tcp   http    0.106754  1328       820        S1         -          -          0            ShADTadt   20        3720          12        2280          -
```

---

## `uniq`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Remove adjacent duplicate events from the output, leaving only unique results.<br><br>Note that due to the large number of fields in typical events, and many fields whose values change often in subtle ways between events (e.g. timestamps), this processor will most often apply to the trimmed output from the [`cut`](#cut) processor. Furthermore, since duplicate field values may not often be adjacent to one another, upstream use of [`sort`](#sort) may also often be appropriate.
| **Syntax**                | `uniq [-c]`                                                           |
| **Required<br>arguments** | None                                                                  | 
| **Optional<br>arguments** | `[-c]`<br>For each unique value shown, include a numeric count of how many times it appeared. |
| **Developer Docs**        | https://godoc.org/github.com/brimsec/zq/proc#Uniq                     |

#### Example:

To see a count of the top issuers of X.509 certificates:

```zq-command
zq -f table '* | cut certificate.issuer | sort | uniq -c | sort -r' x509.log.gz
```

#### Output:
```zq-output head:3
CERTIFICATE.ISSUER                                                                                                                                       _UNIQ
O=VMware Installer                                                                                                                                       1761
CN=Snozberry                                                                                                                                             1108
...
```
