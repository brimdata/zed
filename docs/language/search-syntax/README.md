# Search Syntax

  * [Search all events](#search-all-events)
  * [Value Match](#value-match)
    + [Bare Word](#bare-word)
    + [Quoted Word](#quoted-word)
    + [Glob Wildcards](#glob-wildcards)
    + [Regular Expressions](#regular-expressions)
  * [Field/Value Match](#fieldvalue-match)
    + [Role of Data Types](#role-of-data-types)
    + [Pattern Matches](#pattern-matches)
    + [Containment](#containment)
    + [Comparisons](#comparisons)
    + [Wildcard Field Names](#wildcard-field-names)
    + [Other Examples](#other-examples)
  * [Boolean Operators](#boolean-operators)
    + [`and`](#and)
    + [`or`](#or)
    + [`not`](#not)
    + [Parentheses & Order of Evaluation](#parentheses--order-of-evaluation)

## Search all events

The simplest possible Zed search is a match against all events. This search is expressed in `zq` with the wildcard `*`. The response will be a dump of all events. The default `zq` output is binary ZNG, a compact format that's ideal for working in pipelines. However, in these docs we'll often make use of the `-z` and `-Z` options to output the text-based [ZSON](../../formats/zson.md) format, which is readable at the command line.

#### Example:
```zq-command
zq -z '*' people.zson
```

#### Output:
```zq-output
{name:"morgan",likes:"tart",age:61} (=person)
{name:"quinn",likes:"sweet",age:14} (person)
{name:"jessie",likes:"plain",age:30} (person)
```

If the Zed query argument is left out entirely, this wildcard is the default search. The following shorthand command line would produce the same output shown above.

```
zq -z people.zson
```

To start a Zed pipeline with this default search, you can similarly leave out the leading `* |` before invoking your first [processor](#../processors/README.md) or [aggregate function](#../aggregate-functions/README.md).

#### Example #1:

The following is shorthand for `zq -z '* | cut name'`:

```zq-command
zq -z 'cut name' people.zson
```

#### Output:
```zq-output
{name:"morgan"}
{name:"quinn"}
{name:"jessie"}
```

#### Example #2:

The following is shorthand for `zq -z '* | count() by flavor'`:
```zq-command
zq -z 'count() by flavor' fruit.zson
```

#### Output:
```zq-output
{flavor:"tart",count:2 (uint64)} (=0)
{flavor:"sweet",count:3} (0)
{flavor:"plain",count:1} (0)
```

## Value Match

The search result can be narrowed to include only events that contain certain values in any field(s).

### Bare Word

The simplest form of such a search is a "bare" word (not wrapped in quotes), which will match against any field that contains the word, whether it's an exact match to the data type and value of a field or the word appears as a substring in a field.

For example, searching across all our logs for `14` matches against records that contain fields of numeric types that contain this precise value (such as the `age` field for our `person` records) and also where it appears within string-type fields (such as in our activity `note` records.)

#### Example:
```zq-command
zq -z 14 *
```

#### Output:
```zq-output
{time:"2021-04-27T00:10:33Z",note:"morgan ate 14 apples"}
{name:"quinn",likes:"sweet",age:14} (=person)
```

By comparison, the section below on [Field/Value Match](#fieldvalue-match) describes ways to perform searches against only fields of a specific [data type](../data-types/README.md).

### Quoted Word

Sometimes you may need to search for sequences of multiple words or words that contain special characters. To achieve this, wrap your search term in quotes.

Let's say we want to find the `note` record that contains the text `basket empty-handed`. If typed  "bare" as our Zed query, we'd experience two problems:

   1. `empty-handed` would be interpreted as a mathematical expression to subtract the value in a field called `handed` from the value in a field called `empty`, then search for the computed result.
   2. The space character would cause the input to be interpreted as two separate words and hence the search would not be as strict.

However, wrapping in quotes gives the desired result.

#### Example:
```zq-command
zq -Z '"basket empty-handed"' *
```

#### Output:
```zq-output
{
    time: 2021-04-27T00:19:02Z,
    note: "quinn returrned to the fig basket empty-handed"
}
```

### Glob Wildcards

To find values that may contain arbitrary substrings between or alongside the desired word(s), one or more [glob](https://en.wikipedia.org/wiki/Glob_(programming))-style wildcards can be used.

For example, the following search finds the references to `banana` as well as the reference to the fig-producer `alanar`.

#### Example:
```zq-command
zq -z '*a*ana*' *
```

#### Output:
```zq-output
{name:"banana",color:"yellow",flavor:"sweet"} (=fruit)
{name:"chiquita",fruit:"banana",website:"www.chiquita.com",siteaddr:104.124.1.147} (=producer)
{name:"alanar",fruit:"fig",website:"www.alanar.com.tr",siteaddr:85.95.231.46} (producer)
```

   * **Note:** Our use of `*` to [search all events](#search-all-events) as shown previously is the simplest example of using a glob wildcard.

Glob wildcards only have effect when used with [bare word](#bare-word) searches. An asterisk in a [quoted word](#quoted-word) search will match explicitly against an asterisk character. For example, the following search matches our activity log entry regarding date theft.

#### Example:
```zq-command
zq -Z '"*actually stole*"' *
```

#### Output:
```zq-output
{
    time: 2021-04-27T00:43:23Z,
    note: "someone *actually stole* some dates!"
} (=activity)
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
_PATH TS                          UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                     QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                             TTLS      REJECTED
dns   2018-03-24T17:15:46.07484Z  CYjLXM1Yp1ZuuVJQV1 10.47.6.154 12478     10.10.6.1  53        udp   49089    0.001342 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0         F
dns   2018-03-24T17:15:46.074842Z CYjLXM1Yp1ZuuVJQV1 10.47.6.154 12478     10.10.6.1  53        udp   49089    0.001375 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0         F
dns   2018-03-24T17:15:46.07805Z  Cn1BpA2bKVzWn7IvVe 10.47.6.154 38992     10.10.6.1  53        udp   14171    0.000262 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0         F
dns   2018-03-24T17:15:46.078051Z Cn1BpA2bKVzWn7IvVe 10.47.6.154 38992     10.10.6.1  53        udp   14171    0.000265 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                             0         F
dns   2018-03-24T17:15:46.078071Z CtUHnV2nyFWejoYQ23 10.47.6.154 48071     10.10.6.1  53        udp   64736    0.009286 www.googletagservices.com 1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  F  0 pagead46.l.doubleclick.net,2607:f8b0:4007:804::2002 44266,53  F
dns   2018-03-24T17:15:46.078072Z CtUHnV2nyFWejoYQ23 10.47.6.154 48071     10.10.6.1  53        udp   64736    0.009287 www.googletagservices.com 1      C_INTERNET  28    AAAA       0     NOERROR    F  F  T  F  0 pagead46.l.doubleclick.net,2607:f8b0:4007:804::2002 44266,53  F
dns   2018-03-24T17:16:09.132486Z CUsIaD4CHJDv2dMpp  10.47.7.10  51674     10.0.0.100 53        udp   12049    0.00132  www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0         F
dns   2018-03-24T17:16:09.132491Z CUsIaD4CHJDv2dMpp  10.47.7.10  51674     10.0.0.100 53        udp   12049    0.001316 www.googletagservices.com 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0         F
dns   2018-03-24T17:16:17.181981Z CfofM11rhswW1NDNS  10.47.7.10  52373     10.0.0.100 53        udp   61544    0.000881 www.googleadservices.com  1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 0.0.0.0                                             0         F
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
_PATH TS                          UID               ID.ORIG_H    ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO SERVICE DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:36:30.158539Z ChhAfsfyuz4n2hFMe 10.239.34.35 56602     10.47.6.51 873       tcp   -       0.000004 0          0          S0         -          -          0            S       2         88            0         0             -
 ```

### Role of Data Types

When working with named fields, the data type of the field becomes significant in two ways.

1. To match successfully, the value entered must be comparable to the data type of the named field. For instance, the `host` field of the `http` events in our sample data are of `string` type, since it logs an HTTP header that is often a hostname or an IP address.

   ```zq-command
   zq -z 'count() by host | sort count,host' http.log.gz
   ```

   #### Output:
   ```zq-output head:3
   {host:"0988253c66242502070643933dd49e88.clo.footprintdns.com" (bstring),count:1 (uint64)} (=0)
   {host:"10.47.21.1",count:1} (0)
   {host:"10.47.21.80/..",count:1} (0)
   ...
   ```

   An attempted field/value match `host=10.47.21.1` would not match the event counted in the middle row of this table, since ZQL recognizes the bare value `10.47.21.1` as an IP address before comparing it to all the fields named `host` that it sees in the input stream. However, `host="10.47.21.1"` would match, since the quotes cause ZQL to treat the value as a string.

2.  The correct operator must be chosen based on whether the field type is primitive or complex.  For example, `id.resp_h=10.150.0.85` will match in our sample data because `id.resp_h` is a primitive type, `addr`. However, to check if the same IP had been a transmitting host in a `files` event, the syntax `10.150.0.85 in tx_hosts` would be used because `tx_hosts` is a complex type, `set[addr]`. See the section below on [Containment](#containment) for details regarding the `in` operator.

See the [Data Types](../data-types/README.md) page for more details on types and the operators for working with them.

### Pattern Matches

An important distinction is that a "bare" field/value match is treated as an _exact_ match. If we take one of the results from our [bare word value match](#bare-word) example and attempt to look for `Widgits`, but only on a field named `certificate.subject`, there will be no matches. This is because `Widgits` only happens to appear as a _substring_ of `certificate.subject` fields in our sample data.

#### Example:
```zq-command
zq -f table 'certificate.subject=Widgits' *.log.gz         # Produces no output
```

#### Output:
```zq-output
```

To achieve this with a field/value match, we can use [glob wildcards](#glob-wildcards).

#### Example:
```zq-command
zq -f table 'certificate.subject=*Widgits*' *.log.gz
```

#### Output:

```zq-output head:5
_PATH TS                          ID                 CERTIFICATE.VERSION CERTIFICATE.SERIAL CERTIFICATE.SUBJECT                                          CERTIFICATE.ISSUER                                           CERTIFICATE.NOT_VALID_BEFORE CERTIFICATE.NOT_VALID_AFTER CERTIFICATE.KEY_ALG CERTIFICATE.SIG_ALG     CERTIFICATE.KEY_TYPE CERTIFICATE.KEY_LENGTH CERTIFICATE.EXPONENT CERTIFICATE.CURVE SAN.DNS SAN.URI SAN.EMAIL SAN.IP BASIC_CONSTRAINTS.CA BASIC_CONSTRAINTS.PATH_LEN
x509  2018-03-24T17:15:32.519299Z FZW30y2Nwc9i0qmdvg 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 2018-03-22T14:22:37Z         2045-08-06T14:20:00Z        rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  2018-03-24T17:15:42.635094Z Fo9ltu1O8DGE0KAgC  3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 2018-03-22T14:22:37Z         2045-08-06T14:20:00Z        rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  2018-03-24T17:15:46.548292Z F7oQQK1qo9HfmlN048 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 2018-03-22T14:22:37Z         2045-08-06T14:20:00Z        rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
x509  2018-03-24T17:15:47.493786Z FdBWBA3eODh6nHFt82 3                   C5F8CDF3FFCBBF2D   CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 2018-03-22T14:22:37Z         2045-08-06T14:20:00Z        rsaEncryption       sha256WithRSAEncryption rsa                  2048                   65537                -                 -       -       -         -      T                    -
...
```

[Regular expressions](#regular-expressions) can also be used in field/value matches.

#### Example:
```zq-command
zq -f table 'uri=/scripts\/waE8_BuNCEKM.(pl|sh)/' http.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P TRANS_DEPTH METHOD HOST        URI                         REFERRER VERSION USER_AGENT                                                      ORIGIN REQUEST_BODY_LEN RESPONSE_BODY_LEN STATUS_CODE STATUS_MSG INFO_CODE INFO_MSG TAGS    USERNAME PASSWORD PROXIED ORIG_FUIDS ORIG_FILENAMES ORIG_MIME_TYPES RESP_FUIDS         RESP_FILENAMES RESP_MIME_TYPES
http  2018-03-24T17:17:41.67439Z  Cq3Knz2CEXSJB8ktj  10.164.94.120 40913     10.47.3.142 5800      1           GET    10.47.3.142 /scripts/waE8_BuNCEKM.sh    -        1.0     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                151               404         Not Found  -         -        (empty) -        -        -       -          -              -               F8Jbkj1K2qm2xUR1Bj -              text/html
http  2018-03-24T17:17:42.427215Z C5yUcM3CEFl86YIfY7 10.164.94.120 34369     10.47.3.142 5800      1           GET    10.47.3.142 /scripts/waE8_BuNCEKM.pl    -        1.0     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                151               404         Not Found  -         -        (empty) -        -        -       -          -              -               F5M3Jc4B8xeR13JrP3 -              text/html
http  2018-03-24T17:17:43.933983Z CxJhWB3aN4LcZP59S1 10.164.94.120 37999     10.47.3.142 5800      1           GET    10.47.3.142 /scripts/waE8_BuNCEKM.shtml -        1.0     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                151               404         Not Found  -         -        (empty) -        -        -       -          -              -               Fq7wId3B4sZn24Jrf6 -              text/html
http  2018-03-24T17:17:47.556312Z CgbtuX3gXoYFmEF82l 10.164.94.120 37311     10.47.3.142 8080      23          GET    10.47.3.142 /scripts/waE8_BuNCEKM.sh    -        1.1     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                1635              404         Not Found  -         -        (empty) -        -        -       -          -              -               FRErxf1PYkI30aUNCh -              text/html
http  2018-03-24T17:17:47.561097Z CgbtuX3gXoYFmEF82l 10.164.94.120 37311     10.47.3.142 8080      24          GET    10.47.3.142 /scripts/waE8_BuNCEKM.pl    -        1.1     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                1635              404         Not Found  -         -        (empty) -        -        -       -          -              -               F0fseM1cd8JVpXcnK9 -              text/html
http  2018-03-24T17:17:47.57066Z  CgbtuX3gXoYFmEF82l 10.164.94.120 37311     10.47.3.142 8080      26          GET    10.47.3.142 /scripts/waE8_BuNCEKM.shtml -        1.1     Mozilla/4.0 (compatible; MSIE 8.0; Windows NT 5.1; Trident/4.0) -      0                1635              404         Not Found  -         -        (empty) -        -        -       -          -              -               FdKLBd3fhPSqFIDFWc -              text/html
```

### Containment

Rather than testing for strict equality or pattern matches, you may want to determine if a value is among the many possible elements of a complex field. This is performed with the `in` operator.

Our Zeek `dns` events include the `answers` field, which is an array of the multiple responses that may have been returned for a query. To determine which responses included hostname `e5803.b.akamaiedge.net`, we'll use `in`.

#### Example:
```zq-command
zq -f table '"e5803.b.akamaiedge.net" in answers' dns.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID                ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                               TTLS         REJECTED
dns   2018-03-24T17:20:25.827504Z CATruWimwi1KR0gec  10.47.3.10 63576     10.0.0.100 53        udp   16678    0.072468 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17936,20 F
dns   2018-03-24T17:20:25.827506Z CATruWimwi1KR0gec  10.47.3.10 63576     10.0.0.100 53        udp   16678    0.072468 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17936,20 F
dns   2018-03-24T17:25:29.650694Z CHx5jo2qosRtQOZs1  10.47.6.10 55186     10.0.0.100 53        udp   30327    0.095174 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17632,20 F
dns   2018-03-24T17:25:29.650698Z CHx5jo2qosRtQOZs1  10.47.6.10 55186     10.0.0.100 53        udp   30327    0.095173 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17632,20 F
dns   2018-03-24T17:30:24.694336Z CG5CeD4zyD41L4yt0d 10.47.6.10 55135     10.0.0.100 53        udp   2542     0.032114 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17337,20 F
dns   2018-03-24T17:30:24.694339Z CG5CeD4zyD41L4yt0d 10.47.6.10 55135     10.0.0.100 53        udp   2542     0.032113 www.techrepublic.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 www.techrepublic.com.edgekey.net,e5803.b.akamaiedge.net,23.55.209.124 180,17337,20 F
```

Notice that we wrapped the hostname in quotes. If we'd left it "bare", it would have been interpreted as an attempt to find records with a field called `e5803.b.akamaiedge.net` whose value is contained in the `answers` array of the same record. Since there's no field called `e5803.b.akamaiedge.net` in our data, this would have returning nothing. However, the `query` field does exist in our `dns` events, so the following example does return matches.

#### Example:
```zq-command
zq -f table 'query in answers' dns.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID                ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY      QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS    TTLS REJECTED
dns   2018-03-24T17:24:06.142423Z CCd3Uu1nPHikbjizuc 10.47.7.10 53280     10.0.0.100 53        udp   25252    0.000868 10.47.7.30 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 10.47.7.30 0    F
dns   2018-03-24T17:24:06.142426Z CCd3Uu1nPHikbjizuc 10.47.7.10 53280     10.0.0.100 53        udp   25252    0.000869 10.47.7.30 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 10.47.7.30 0    F
dns   2018-03-24T17:30:43.213667Z CV4T3j1mb4LbxNNgBl 10.47.7.10 53647     10.0.0.100 53        udp   45561    0.001054 10.47.7.30 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 10.47.7.30 0    F
dns   2018-03-24T17:30:43.213671Z CV4T3j1mb4LbxNNgBl 10.47.7.10 53647     10.0.0.100 53        udp   45561    0.001053 10.47.7.30 1      C_INTERNET  1     A          0     NOERROR    T  F  T  T  0 10.47.7.30 0    F
```

Determining whether the value of a Zeek `addr`-type field is contained within a subnet also uses `in`.

#### Example:
```zq-command
zq -f table 'id.resp_h in 208.78.0.0/16' conn.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H     ID.RESP_P PROTO SERVICE DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:32:44.212387Z CngWP41W7wzyQtMG4k 10.47.26.25 59095     208.78.71.136 53        udp   dns     0.003241 72         402        SF         -          -          0            Dd      2         128           2         458           -
conn  2018-03-24T17:32:52.32455Z  CgZ2D84oSTX0Xw2fEl 10.47.26.25 59095     208.78.70.136 53        udp   dns     0.004167 144        804        SF         -          -          0            Dd      4         256           4         916           -
conn  2018-03-24T17:33:07.538564Z CGfWHn2Y6IDSBra1K4 10.47.26.25 59095     208.78.71.31  53        udp   dns     3.044438 276        1188       SF         -          -          0            Dd      6         444           6         1356          -
conn  2018-03-24T17:35:07.721609Z CCbNQn22j5UPZ4tute 10.47.26.25 59095     208.78.70.136 53        udp   dns     0.1326   176        870        SF         -          -          0            Dd      4         288           4         982           -
```

### Comparisons

In addition to testing for equality and pattern matching via `=`, other common comparison operators `!=`, `<`, `>`, `<=`, and `>=` are also available.

For example, the following search finds connections that have transferred many bytes.

#### Example:
```zq-command
zq -f table 'orig_bytes > 1000000' *.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:25:15.208232Z CVimRo24ubbKqFvNu7 172.30.255.1 11        10.128.0.207 0         icmp  -       100.721937  1647088    0          OTH        -          -          0            -                44136     2882896       0         0             -
conn  2018-03-24T17:15:20.630818Z CO0MhB2NCc08xWaly8 10.47.1.154  49814     134.71.3.17  443       tcp   -       1269.512465 1618740    12880888   OTH        -          -          0            ^dtADTatTtTtTtT  110169    7594230       111445    29872050      -
conn  2018-03-24T17:15:20.637761Z Cmgywj2O8KZAHHjddb 10.47.1.154  49582     134.71.3.17  443       tcp   -       1266.367457 1594682    53255700   OTH        -          -          0            ^dtADTatTtTtTtTW 131516    8407458       142488    110641641     -
conn  2018-03-24T17:15:20.705347Z CWtQuI2IMNyE1pX47j 10.47.6.161  52121     134.71.3.17  443       tcp   -       1269.320626 2267243    54791018   OTH        -          -          0            DTadtATttTtTtT   152819    10575303      158738    113518994     -
conn  2018-03-24T17:33:05.415532Z Cy3R5w2pfv8oSEpa2j 10.47.8.19   49376     10.128.0.214 443       tcp   -       202.457994  4862366    1614249    S1         -          -          0            ShAdtttDTaTTTt   7280      10015980      6077      3453020       -
```

The same operators also work when comparing characters in `string`-type values, such as this search that finds DNS requests that were issued for hostnames at the high end of the alphabet.

#### Example:
```zq-command
zq -f table 'query > zippy' *.log.gz
```

#### Output:
```zq-output
_PATH TS                          UID               ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                                                    QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                TTLS       REJECTED
dns   2018-03-24T17:30:09.84174Z  Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001694 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:30:09.841742Z Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001697 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:34:52.637234Z CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019491 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
dns   2018-03-24T17:34:52.637238Z CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019493 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
```

### Wildcard Field Names

It's possible to search across _all_ top-level fields of a value's data type by entering a wildcard where you'd normally enter the field name.

In the following search for the `addr`-type value `10.150.0.85`, we match only a single `notice` event, as this is the only event in our data with a matching top-level field of the `addr` type (the `dst` field).

#### Example:
```zq-command
zq -f table '*=10.150.0.85' *.log.gz
```

#### Output:
```zq-output
_PATH  TS                          UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H   ID.RESP_P FUID               FILE_MIME_TYPE FILE_DESC PROTO NOTE                     MSG                                                              SUB                                                          SRC          DST         P   N PEER_DESCR ACTIONS            SUPPRESS_FOR REMOTE_LOCATION.COUNTRY_CODE REMOTE_LOCATION.REGION REMOTE_LOCATION.CITY REMOTE_LOCATION.LATITUDE REMOTE_LOCATION.LONGITUDE
notice 2018-03-24T17:15:32.521729Z Ckwqsn2ZSiVGtyiFO5 10.47.24.186 55782     10.150.0.85 443       FZW30y2Nwc9i0qmdvg -              -         tcp   SSL::Invalid_Server_Cert SSL certificate validation failed with (self signed certificate) CN=10.150.0.85,O=Internet Widgits Pty Ltd,ST=Some-State,C=AU 10.47.24.186 10.150.0.85 443 - -          Notice::ACTION_LOG 3600         -                            -                      -                    -                        -
```

This same address `10.150.0.85` appears in other IP address fields in our data such as `id.resp_h`, but these were not matched because these happend to be _nested_ fields (e.g. `resp_h` is a field nested inside the record called `id`). An enhancement with an alternate syntax is planned to allow type-specific searches to reach into nested records when desired (see [#2250](https://github.com/brimdata/zed/issues/2250) and [#1428](https://github.com/brimdata/zed/issues/1428)). Compare this with the [bare word](#bare-word) searches we showed previously that perform type-independent matches for values in all locations, including in nested records and complex fields.

The `*` wildcard can also be used to match when the value appears in a complex top-level field. Searching again for our `addr`-type value `10.150.0.85`, here we'll match in complex fields of type `set[addr]` or `array[addr]`, such as `tx_hosts` in this case.

#### Example:
```zq-command
zq -f table '10.150.0.85 in *' *.log.gz
```

#### Output:
```zq-output head:5
_PATH TS                          FUID               TX_HOSTS    RX_HOSTS     CONN_UIDS          SOURCE DEPTH ANALYZERS     MIME_TYPE                    FILENAME DURATION LOCAL_ORIG IS_ORIG SEEN_BYTES TOTAL_BYTES MISSING_BYTES OVERFLOW_BYTES TIMEDOUT PARENT_FUID MD5                              SHA1                                     SHA256 EXTRACTED EXTRACTED_CUTOFF EXTRACTED_SIZE
files 2018-03-24T17:15:32.519299Z FZW30y2Nwc9i0qmdvg 10.150.0.85 10.47.24.186 Ckwqsn2ZSiVGtyiFO5 SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 2018-03-24T17:15:42.635094Z Fo9ltu1O8DGE0KAgC  10.150.0.85 10.47.8.10   CqwJmZ2Lzd42fuvg4k SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 2018-03-24T17:15:46.548292Z F7oQQK1qo9HfmlN048 10.150.0.85 10.47.27.186 CvTTHG2M6xPqDMDLB7 SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
files 2018-03-24T17:15:47.493786Z FdBWBA3eODh6nHFt82 10.150.0.85 10.10.18.2   ChpfSB4FWhg3xHI3yb SSL    0     MD5,SHA1,X509 application/x-x509-user-cert -        0        -          F       909        -           0             0              F        -           9fb39c2b34d22a7ba507dedb4e155101 d95fcbd453c842d6b432e5ec74a720c700c50393 -      -         -                -
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
_PATH TS                          UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P VERSION CIPHER                                CURVE     SERVER_NAME       RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS                                                            CLIENT_CERT_CHAIN_FUIDS SUBJECT            ISSUER                                  CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   2018-03-24T17:23:00.244457Z CUG0fiQAzL4rNWxai  10.47.2.100 36150     52.85.83.228 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FXKmyTbr7HlvyL1h8,FADhCTvkq1ILFnD3j,FoVjYR16c3UIuXj4xk,FmiRYe1P53KOolQeVi   (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
ssl   2018-03-24T17:24:00.189735Z CSbGJs3jOeB6glWLJj 10.47.7.154 27137     52.85.83.215 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FuW2cZ3leE606wXSia,Fu5kzi1BUwnF0bSCsd,FyTViI32zPvCmNXgSi,FwV6ff3JGj4NZcVPE4 (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
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
_PATH TS                          UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:25:15.208232Z CVimRo24ubbKqFvNu7 172.30.255.1 11        10.128.0.207 0         icmp  -       100.721937  1647088    0          OTH        -          -          0            -                44136     2882896       0         0             -
conn  2018-03-24T17:15:20.630818Z CO0MhB2NCc08xWaly8 10.47.1.154  49814     134.71.3.17  443       tcp   -       1269.512465 1618740    12880888   OTH        -          -          0            ^dtADTatTtTtTtT  110169    7594230       111445    29872050      -
conn  2018-03-24T17:15:20.637761Z Cmgywj2O8KZAHHjddb 10.47.1.154  49582     134.71.3.17  443       tcp   -       1266.367457 1594682    53255700   OTH        -          -          0            ^dtADTatTtTtTtTW 131516    8407458       142488    110641641     -
conn  2018-03-24T17:15:20.705347Z CWtQuI2IMNyE1pX47j 10.47.6.161  52121     134.71.3.17  443       tcp   -       1269.320626 2267243    54791018   OTH        -          -          0            DTadtATttTtTtT   152819    10575303      158738    113518994     -
conn  2018-03-24T17:33:05.415532Z Cy3R5w2pfv8oSEpa2j 10.47.8.19   49376     10.128.0.214 443       tcp   -       202.457994  4862366    1614249    S1         -          -          0            ShAdtttDTaTTTt   7280      10015980      6077      3453020       -
_PATH TS                          UID               ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                                                    QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                TTLS       REJECTED
dns   2018-03-24T17:30:09.84174Z  Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001694 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:30:09.841742Z Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001697 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:34:52.637234Z CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019491 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
...
```

### `not`

Use the `not` operator to invert the matching logic in the term that comes to the right of it in your search.

For example, suppose you've noticed that the vast majority of the sample Zeek events are of log types like `conn`, `dns`, `files`, etc. You could review some of the less-common Zeek event types by inverting the logic of a [regexp match](#regular-expressions).

#### Example:
```zq-command
zq -f table 'not _path=/conn|dns|files|ssl|x509|http|weird/' *.log.gz
```

#### Output:

```zq-output head:10
_PATH        TS                          TS_DELTA   PEER GAPS ACKS    PERCENT_LOST
capture_loss 2018-03-24T17:30:20.600852Z 900.000127 zeek 1400 1414346 0.098986
capture_loss 2018-03-24T17:36:30.158766Z 369.557914 zeek 919  663314  0.138547
_PATH   TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H   ID.RESP_P RTT      NAMED_PIPE     ENDPOINT              OPERATION
dce_rpc 2018-03-24T17:15:25.396014Z CgxsNA1p2d0BurXd7c 10.164.94.120 36643     10.47.3.151 1030      0.000431 1030           samr                  SamrConnect2
dce_rpc 2018-03-24T17:15:41.35659Z  CveQB24ujSZ3l34LRi 10.128.0.233  33692     10.47.21.25 135       0.000684 135            IObjectExporter       ComplexPing
dce_rpc 2018-03-24T17:15:54.621588Z CWyKrz4YlSyPGoE8Bf 10.128.0.214  41717     10.47.8.142 445       0.002721 \\pipe\\ntsvcs svcctl                OpenSCManagerW
dce_rpc 2018-03-24T17:15:54.63042Z  CWyKrz4YlSyPGoE8Bf 10.128.0.214  41717     10.47.8.142 445       0.054631 \\pipe\\ntsvcs svcctl                CreateServiceW
dce_rpc 2018-03-24T17:15:54.69324Z  CWyKrz4YlSyPGoE8Bf 10.128.0.214  41717     10.47.8.142 445       0.008842 \\pipe\\ntsvcs svcctl                StartServiceW
dce_rpc 2018-03-24T17:15:54.711445Z CWyKrz4YlSyPGoE8Bf 10.128.0.214  41717     10.47.8.142 445       0.068546 \\pipe\\ntsvcs svcctl                DeleteService
...
```

* **Note**: The `!` operator can also be used as alternative shorthand:

        zq -f table '! _path=/conn|dns|files|ssl|x509|http|weird/' *.log.gz

### Parentheses & Order of Evaluation

Unless wrapped in parentheses, a search expression is evaluated in _left-to-right order_.

For example, the following search leverages the implicit boolean `and` to find all `smb_mapping` events in which the `share_type` field is set to a value other than `DISK`.

#### Example:
```zq-command
zq -f table 'not share_type=DISK _path=smb_mapping' *.log.gz
```

#### Output:
```zq-output head:5
_PATH       TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 2018-03-24T17:15:21.625534Z ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:22.021668Z C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:24.619169Z C2byFA2Y10G1GLUXgb 10.164.94.120 35337     10.47.27.80  445       \\\\PC-NEWMAN\\IPC$      -       -                  PIPE
smb_mapping 2018-03-24T17:15:25.562072Z C3kUnM2kEJZnvZmSp7 10.164.94.120 45903     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      -       -                  PIPE
...
```

Terms wrapped in parentheses along with their operators will be evaluated _first_, overriding the default left-to-right evaluation. If we wrap the search terms as shown below, now we match almost every event we have. This is because the `not` is now inverting the logic of everything in the parentheses, hence giving us all stored events _other than_ `smb_mapping` events that have the value of their `share_type` field set to `DISK`.

#### Example:
```zq-command
zq -f table 'not (share_type=DISK _path=smb_mapping)' *.log.gz
```

#### Output:
```zq-output head:9
_PATH        TS                          TS_DELTA   PEER GAPS ACKS    PERCENT_LOST
capture_loss 2018-03-24T17:30:20.600852Z 900.000127 zeek 1400 1414346 0.098986
capture_loss 2018-03-24T17:36:30.158766Z 369.557914 zeek 919  663314  0.138547
_PATH TS                          UID                ID.ORIG_H      ID.ORIG_P ID.RESP_H     ID.RESP_P PROTO SERVICE  DURATION ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY     ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:15:21.255387Z C8Tful1TvM3Zf5x8fl 10.164.94.120  39681     10.47.3.155   3389      tcp   -        0.004266 97         19         RSTR       -          -          0            ShADTdtr    10        730           6         342           -
conn  2018-03-24T17:15:21.411148Z CXWfTK3LRdiuQxBbM6 10.47.25.80    50817     10.128.0.218  23189     tcp   -        0.000486 0          0          REJ        -          -          0            Sr          2         104           2         80            -
conn  2018-03-24T17:15:21.926018Z CM59GGQhNEoKONb5i  10.47.25.80    50817     10.128.0.218  23189     tcp   -        0.000538 0          0          REJ        -          -          0            Sr          2         104           2         80            -
conn  2018-03-24T17:15:22.690601Z CuKFds250kxFgkhh8f 10.47.25.80    50813     10.128.0.218  27765     tcp   -        0.000546 0          0          REJ        -          -          0            Sr          2         104           2         80            -
conn  2018-03-24T17:15:23.205187Z CBrzd94qfowOqJwCHa 10.47.25.80    50813     10.128.0.218  27765     tcp   -        0.000605 0          0          REJ        -          -          0            Sr          2         104           2         80            -
...
```

Parentheses can also be nested.

#### Example:
```zq-command
zq -f table '((not share_type=DISK) and (service=IPC)) _path=smb_mapping' *.log.gz
```

#### Output:
```zq-output head:5
_PATH       TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 2018-03-24T17:15:21.625534Z ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:22.021668Z C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:31.475945Z Cvaqhu3VhuXlDOMgXg 10.164.94.120 37127     10.47.3.151  445       \\\\COTTONCANDY4\\IPC$   IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:36.306275Z CsZ7Be4NlqaJSNNie4 10.164.94.120 33921     10.47.23.166 445       \\\\PARKINGGARAGE\\IPC$  IPC     -                  PIPE
...
```

Except when writing the most common searches that leverage only the implicit `and`, it's generally good practice to use parentheses even when not strictly necessary, just to make sure your queries clearly communicate their intended logic.
