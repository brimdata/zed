# Search Syntax

  * [Search all records](#search-all-records)
  * [Value Match](#value-match)
    + [Bare Word](#bare-word)
    + [Quoted Word](#quoted-word)
    + [Glob Wildcards](#glob-wildcards)
    + [Regular Expressions](#regular-expressions)
  * [Field/Value Match](#fieldvalue-match)
    + [Role of Data Types](#role-of-data-types)
    + [Finding Patterns with `matches`](#finding-patterns-with-matches)
    + [Containment](#containment)
    + [Comparisons](#comparisons)
    + [Other Examples](#other-examples)
  * [Boolean Logic](#boolean-logic)
    + [`and`](#and)
    + [`or`](#or)
    + [`not`](#not)
    + [Parentheses & Order of Evaluation](#parentheses--order-of-evaluation)

## Search all records

The simplest possible Zed search is a match of all records. This search is
expressed in `zq` with the wildcard `*`. The response will be a dump of all
records. The default `zq` output is binary [ZNG](../../formats/zng.md), a
compact format that's ideal for working in pipelines. However, in these docs
we'll sometimes make use of the `-z` option to output the text-based
[ZSON](../../formats/zson.md) format, which is readable at the command line.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z '*' schools.zson
```

#### Output:
```mdtest-output head
{School:"'3R' Middle",District:"Nevada County Office of Education",City:"Nevada City",County:"Nevada",Zip:"95959",Latitude:null (float64),Longitude:null (float64),Magnet:null (bool),OpenDate:1995-10-30T00:00:00Z,ClosedDate:1996-06-28T00:00:00Z,Phone:null (string),StatusType:"Merged",Website:null (string)} (=school)
{School:"100 Black Men of the Bay Area Community",District:"Oakland Unified",City:"Oakland",County:"Alameda",Zip:"94607-1404",Latitude:37.745418,Longitude:-122.14067,Magnet:null,OpenDate:2012-08-06T00:00:00Z,ClosedDate:2014-10-28T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.100school.org"} (school)
{School:"101 Elementary",District:"Victor Elementary",City:"Victorville",County:"San Bernardino",Zip:"92395-3360",Latitude:null,Longitude:null,Magnet:null,OpenDate:1996-02-07T00:00:00Z,ClosedDate:2005-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.charter101.org"} (school)
{School:"180 Program",District:"Novato Unified",City:"Novato",County:"Marin",Zip:"94947-4004",Latitude:38.097792,Longitude:-122.57617,Magnet:null,OpenDate:2012-08-22T00:00:00Z,ClosedDate:2014-06-13T00:00:00Z,Phone:null,StatusType:"Closed",Website:null} (school)
...
```

If the query argument is left out entirely, this wildcard is the default
search. The following shorthand command line would produce the same output
shown above.

```
zq -z schools.zson
```

To start a Zed pipeline with this default search, you can similarly leave out
the leading `* |` before invoking your first
[operator](#../operators/README.md) or
[aggregate function](#../aggregate-functions/README.md). The following example
is shorthand for:

```
zq -z '* | cut School,City' schools.zson
```

#### Example:

```mdtest-command zed-sample-data/edu/zson
zq -z 'cut School,City' schools.zson
```

#### Output:
```mdtest-output head
{School:"'3R' Middle",City:"Nevada City"}
{School:"100 Black Men of the Bay Area Community",City:"Oakland"}
{School:"101 Elementary",City:"Victorville"}
...
```

## Value Match

The search result can be narrowed to include only records that contain certain
values in any field(s).

### Bare Word

The simplest form of such a search is a _bare_ word (not wrapped in quotes),
which will match against any field that contains the word, whether it's an
exact match to the data type and value of a field or the word appears as a
substring in a field.

For example, searching across all our logs for `596` matches records that
contain numeric fields of this precise value (such as from the SAT test scores
in our sample data) and also where it appears within string-typed fields (such
as the zip code and phone number fields.)

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z '596' *.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:591 (uint16),AvgScrRead:610 (uint16),AvgScrWrite:596 (uint16),cname:"Los Angeles",dname:"William S. Hart Union High",sname:"Academy of the Canyons"} (=satscore)
{AvgScrMath:614,AvgScrRead:596,AvgScrWrite:592,cname:"Alameda",dname:"Pleasanton Unified",sname:"Amador Valley High"} (satscore)
{AvgScrMath:620,AvgScrRead:596,AvgScrWrite:590,cname:"Yolo",dname:"Davis Joint Unified",sname:"Davis Senior High"} (satscore)
{School:"Achieve Charter School of Paradise Inc.",District:"Paradise Unified",City:"Paradise",County:"Butte",Zip:"95969-3913",Latitude:39.760323,Longitude:-121.62078,Magnet:false,OpenDate:2005-09-12T00:00:00Z,ClosedDate:null (time),Phone:"(530) 872-4100",StatusType:"Active",Website:"www.achievecharter.org"} (=school)
{School:"Alliance Ouchi-O'Donovan 6-12 Complex",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90043-2622",Latitude:33.993484,Longitude:-118.32246,Magnet:false,OpenDate:2006-09-05T00:00:00Z,ClosedDate:null,Phone:"(323) 596-2290",StatusType:"Active",Website:"http://ouchihs.org"} (school)
...
```

By comparison, the section below on [Field/Value Match](#fieldvalue-match)
describes ways to perform searches against only fields of a specific
[data type](../data-types/README.md).

### Quoted Word

Sometimes you may need to search for sequences of multiple words or words that
contain special characters. To achieve this, wrap your search term in quotes.

Let's say we've noticed that a couple of the school names in our sample data
include the string `Defunct=`. An attempt to enter this as a bare word search
causes an error because the language parser interpreted this as the start of
an attempted [field/value match](#fieldvalue-match) for a field named
`Defunct`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'Defunct=' *.zson || true
```

#### Output:
```mdtest-output
zq: error parsing Zed at column 8:
Defunct=
   === ^ ===
```

However, wrapping in quotes gives the desired result.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z '"Defunct="' schools.zson
```

#### Output:
```mdtest-output
{School:"Lincoln Elem 'Defunct=",District:"Modesto City Elementary",City:null (string),County:"Stanislaus",Zip:null (string),Latitude:null (float64),Longitude:null (float64),Magnet:null (bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null (string),StatusType:"Closed",Website:null (string)} (=school)
{School:"Lovell Elem 'Defunct=",District:"Cutler-Orosi Joint Unified",City:null,County:"Tulare",Zip:null,Latitude:null,Longitude:null,Magnet:null,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:null} (school)
```

Wrapping in quotes is particularly handy when you're looking for long, specific
strings that may have several special characters in them. For example, let's
say we're looking for information on the Union Hill Elementary district.
Entered without quotes, we up matching way more records than we intended since
each space character between words is treated as a [boolean `and`](#and) .

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'Union Hill Elementary' schools.zson
```

#### Output:
```mdtest-output head
{School:"A. M. Thomas Middle",District:"Lost Hills Union Elementary",City:"Lost Hills",County:"Kern",Zip:"93249-0158",Latitude:35.615269,Longitude:-119.69955,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null (time),Phone:"(661) 797-2626",StatusType:"Active",Website:null (string)} (=school)
{School:"Alview Elementary",District:"Alview-Dairyland Union Elementary",City:"Chowchilla",County:"Madera",Zip:"93610-9225",Latitude:37.050632,Longitude:-120.4734,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(559) 665-2275",StatusType:"Active",Website:null} (school)
{School:"Anaverde Hills",District:"Westside Union Elementary",City:"Palmdale",County:"Los Angeles",Zip:"93551-5518",Latitude:34.564651,Longitude:-118.18012,Magnet:false,OpenDate:2005-08-15T00:00:00Z,ClosedDate:null,Phone:"(661) 575-9923",StatusType:"Active",Website:null} (school)
{School:"Apple Blossom",District:"Twin Hills Union Elementary",City:"Sebastopol",County:"Sonoma",Zip:"95472-3917",Latitude:38.387396,Longitude:-122.84954,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(707) 823-1041",StatusType:"Active",Website:null} (school)
...
```

However, wrapping the entire term in quotes allows us to search for the
complete string, spaces included.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z '"Union Hill Elementary"' schools.zson
```

#### Output:
```mdtest-output
{School:"Highland Oaks Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:null (float64),Longitude:null (float64),Magnet:null (bool),OpenDate:1997-09-02T00:00:00Z,ClosedDate:2003-07-02T00:00:00Z,Phone:null (string),StatusType:"Closed",Website:null (string)} (=school)
{School:"Union Hill 3R Community Day",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:39.229055,Longitude:-121.07127,Magnet:null,OpenDate:2003-08-20T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.uhsd.k12.ca.us"} (school)
{School:"Union Hill Charter Home",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1995-07-14T00:00:00Z,ClosedDate:2015-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.uhsd.k12.ca.us"} (school)
{School:"Union Hill Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"} (school)
{School:"Union Hill Middle",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"94945-8805",Latitude:39.205006,Longitude:-121.03778,Magnet:false,OpenDate:2013-08-14T00:00:00Z,ClosedDate:null,Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"} (school)
{School:null,District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8730",Latitude:39.208869,Longitude:-121.03551,Magnet:null,OpenDate:null,ClosedDate:null,Phone:"(530) 273-0647",StatusType:"Active",Website:"www.uhsd.k12.ca.us"} (school)
```

### Glob Wildcards

To find values that may contain arbitrary substrings between or alongside the
desired word(s), one or more
[glob](https://en.wikipedia.org/wiki/Glob_(programming))-style wildcards can be
used.

For example, the following search finds records that contain school names
that have some additional text between `ACE` and `Academy`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'ACE*Academy' schools.zson
```

#### Output:
```mdtest-output head
{School:"ACE Empower Academy",District:"Santa Clara County Office of Education",City:"San Jose",County:"Santa Clara",Zip:"95116-3423",Latitude:37.348601,Longitude:-121.8446,Magnet:false,OpenDate:2008-08-26T00:00:00Z,ClosedDate:null (time),Phone:"(408) 729-3920",StatusType:"Active",Website:"www.acecharter.org"} (=school)
{School:"ACE Inspire Academy",District:"San Jose Unified",City:"San Jose",County:"Santa Clara",Zip:"95112-6334",Latitude:37.350981,Longitude:-121.87205,Magnet:false,OpenDate:2015-08-03T00:00:00Z,ClosedDate:null,Phone:"(408) 295-6008",StatusType:"Active",Website:"www.acecharter.org"} (school)
```

> **Note:** Our use of `*` to [search all records](#search-all-records) as
> shown previously is the simplest example of using a glob wildcard.

Glob wildcards only have effect when used with [bare word](#bare-word)
searches. An asterisk in a [quoted word](#quoted-word) search will match
explicitly against an asterisk character.

### Regular Expressions

For matching that requires more precision than can be achieved with
[glob wildcards](#glob-wildcards), regular expressions (regexps) are also
available. To use them, simply place a `/` character before and after the
regexp.

For example, since there's so many high schools in our sample data, to find
only records containing strings that _begin_ with the word `High`:

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z '/^High /' schools.zson
```

#### Output:
```mdtest-output head
{School:"High Desert",District:"Soledad-Agua Dulce Union Eleme",City:"Acton",County:"Los Angeles",Zip:"93510",Latitude:34.490977,Longitude:-118.19646,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1993-06-30T00:00:00Z,Phone:null (string),StatusType:"Merged",Website:null (string)} (=school)
{School:"High Desert",District:"Acton-Agua Dulce Unified",City:"Acton",County:"Los Angeles",Zip:"93510-1757",Latitude:34.492578,Longitude:-118.19039,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(661) 269-0310",StatusType:"Active",Website:null} (school)
{School:"High Desert Academy",District:"Eastern Sierra Unified",City:"Benton",County:"Mono",Zip:"93512-0956",Latitude:37.818597,Longitude:-118.47712,Magnet:null,OpenDate:1996-09-03T00:00:00Z,ClosedDate:2012-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.esusd.org"} (school)
{School:"High Desert Academy of Applied Arts and Sciences",District:"Victor Valley Union High",City:"Victorville",County:"San Bernardino",Zip:"92394",Latitude:34.531144,Longitude:-117.31697,Magnet:null,OpenDate:2004-09-07T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null,StatusType:"Closed",Website:"www.hdaaas.org"} (school)
...
```

Regexps are a detailed topic all their own. For details, reference the
[documentation for re2](https://github.com/google/re2/wiki/Syntax), which is
the library that Zed uses to provide regexp support.

## Field/Value Match

The search result can be narrowed to include only records that contain a
certain value in a particular named field. For example, the following search
will only match records containing the field called `District` where it is set
to the precise string value `Marin County ROP`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'District=="Winton"' schools.zson
```

#### Output:

```mdtest-output
{School:"Frank Sparkes Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.382084,Longitude:-120.61847,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null (time),Phone:"(209) 357-6180",StatusType:"Active",Website:null (string)} (=school)
{School:"Sybil N. Crookham Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0130",Latitude:37.389501,Longitude:-120.61636,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(209) 357-6182",StatusType:"Active",Website:null} (school)
{School:"Winfield Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388",Latitude:37.389121,Longitude:-120.60442,Magnet:false,OpenDate:2007-08-13T00:00:00Z,ClosedDate:null,Phone:"(209) 357-6891",StatusType:"Active",Website:null} (school)
{School:"Winton Middle",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-1477",Latitude:37.379938,Longitude:-120.62263,Magnet:false,OpenDate:1990-07-20T00:00:00Z,ClosedDate:null,Phone:"(209) 357-6189",StatusType:"Active",Website:null} (school)
{School:null,District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.389467,Longitude:-120.6147,Magnet:null,OpenDate:null,ClosedDate:null,Phone:"(209) 357-6175",StatusType:"Active",Website:"www.winton.k12.ca.us"} (school)
```

Because the right-hand-side value we were comparing to the `District` field
was a string, it was necessary to wrap it in quotes. If we'd left it bare, it
would have been interpreted as a field name.

For example, to see the records in which the school and district name are the
same:

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'School==District' schools.zson
```

#### Output:

```mdtest-output head
{School:"Adelanto Elementary",District:"Adelanto Elementary",City:"Adelanto",County:"San Bernardino",Zip:"92301-1734",Latitude:34.576166,Longitude:-117.40944,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null (time),Phone:"(760) 246-5892",StatusType:"Active",Website:null (string)} (=school)
{School:"Allensworth Elementary",District:"Allensworth Elementary",City:"Allensworth",County:"Tulare",Zip:"93219-9709",Latitude:35.864487,Longitude:-119.39068,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(661) 849-2401",StatusType:"Active",Website:null} (school)
{School:"Alta Loma Elementary",District:"Alta Loma Elementary",City:"Alta Loma",County:"San Bernardino",Zip:"91701-5007",Latitude:34.12597,Longitude:-117.59744,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(909) 484-5000",StatusType:"Active",Website:null} (school)
...
```

### Role of Data Types

To match successfully when working with named fields, the value must be
comparable to the data type of the field.

For instance, the 'Zip' field in our schools data is of `string` type because
several values are of the extended format that includes a hyphen and four
additional digits.

```mdtest-command zed-sample-data/edu/zson
zq -z 'cut Zip' schools.zson
```

#### Output:
```mdtest-output head
{Zip:"95959"}
{Zip:"94607-1404"}
{Zip:"92395-3360"}
...
```

An attempted field/value match `Zip==95959` would _not_ match the top record
shown, since Zed recognizes the bare value `95959` as a number before
comparing it to all the fields named `Zip` that it sees in the input stream.
However, `Zip=="95959"` _would_ match, since the quotes cause Zed to treat the
value as a string.

See the [Data Types](../data-types/README.md) page for more details.

### Finding Patterns with `matches`

When comparing a named field to a quoted value, the quoted value is treated as
an _exact_ match.

For example, let's say we know there's several schools that start with
`Luther`, but only a couple districts do. Because `Luther` only appears as a
_substring_ of the district names in our sample data, the following example
produces no output.

#### Example:

```mdtest-command zed-sample-data/edu/zson
zq -z 'District=="Luther"' schools.zson
```

#### Output:
```mdtest-output
```

To achieve this with a field/value match, we enter `matches` before specifying
a [glob wildcard](#glob-wildcards).

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'District matches Luther*' schools.zson
```

#### Output:

```mdtest-output head
{School:"Luther Burbank Elementary",District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null (time),Phone:"(408) 295-1814",StatusType:"Active",Website:null (string)} (=school)
{School:null,District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:null,OpenDate:null,ClosedDate:null,Phone:"(408) 295-2450",StatusType:"Active",Website:"www.lbsd.k12.ca.us"} (school)
```

[Regular expressions](#regular-expressions) can also be used with `matches`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'School matches /^Sunset (Ranch|Ridge) Elementary/' schools.zson
```

#### Output:
```mdtest-output
{School:"Sunset Ranch Elementary",District:"Rocklin Unified",City:"Rocklin",County:"Placer",Zip:"95765-5441",Latitude:38.826425,Longitude:-121.2864,Magnet:false,OpenDate:2010-08-17T00:00:00Z,ClosedDate:null (time),Phone:"(916) 624-2048",StatusType:"Active",Website:"www.rocklin.k12.ca.us"} (=school)
{School:"Sunset Ridge Elementary",District:"Pacifica",City:"Pacifica",County:"San Mateo",Zip:"94044-2029",Latitude:37.653836,Longitude:-122.47919,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null,Phone:"(650) 738-6687",StatusType:"Active",Website:null} (school)
```

### Containment

Rather than testing for strict equality or pattern matches, you may want to
determine if a value is among the many possible elements of a complex field.
This is performed with `in`.

Since our sample data doesn't contain complex fields, we'll make one by
using the [`union`](../aggregate-functions/#union) aggregate functions to
create a set-typed field called `Schools` that contains the unique school names
per district. From these we'll attempt to observe each set that contains a
school named `Lincoln Elementary`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -Z 'Schools:=union(School) by District | sort | "Lincoln Elementary" in Schools' schools.zson
```

#### Output:
```mdtest-output head
{
    District: "Alpine County Unified",
    Schools: |[
        "",
        "Woodfords High",
        "Clay Elementary",
        "Bear Valley High",
        "Lincoln Elementary",
        "Jmms Satellite Campus",
        "Bear Valley Elementary",
        "Diamond Valley Elementary",
        "Kirkwood Meadows Elementary",
        "Alpine County Special Education",
        "Diamond Valley Independent Study",
        "Alpine County Secondary Community Day",
        "Alpine County Elementary Community Day"
    ]|
}
...
```

Determining whether the value of an `ip`-type field is contained within a
subnet also uses `in`.

The following example locates all schools whose web sites are hosted in an
IP address in the class A `38`.

#### Example:
```mdtest-command zed-sample-data/edu/zson
zq -z 'addr in 38.0.0.0/8' webaddrs.zson
```

#### Output:
```mdtest-output
{Website:"www.learningchoice.org",addr:38.95.129.245}
{Website:"www.mpcsd.org",addr:38.102.147.181}
```

### Comparisons

In addition to testing for equality via `==` and finding patterns via
`matches`, the other common methods of comparison `!=`, `<`, `>`, `<=`, and
`>=` are also available.

For example, the following search finds connections that have transferred many bytes.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'orig_bytes > 1000000' *.log.gz
```

#### Output:
```mdtest-output
_PATH TS                          UID                ID.ORIG_H    ID.ORIG_P ID.RESP_H    ID.RESP_P PROTO SERVICE DURATION    ORIG_BYTES RESP_BYTES CONN_STATE LOCAL_ORIG LOCAL_RESP MISSED_BYTES HISTORY          ORIG_PKTS ORIG_IP_BYTES RESP_PKTS RESP_IP_BYTES TUNNEL_PARENTS
conn  2018-03-24T17:25:15.208232Z CVimRo24ubbKqFvNu7 172.30.255.1 11        10.128.0.207 0         icmp  -       100.721937  1647088    0          OTH        -          -          0            -                44136     2882896       0         0             -
conn  2018-03-24T17:15:20.630818Z CO0MhB2NCc08xWaly8 10.47.1.154  49814     134.71.3.17  443       tcp   -       1269.512465 1618740    12880888   OTH        -          -          0            ^dtADTatTtTtTtT  110169    7594230       111445    29872050      -
conn  2018-03-24T17:15:20.637761Z Cmgywj2O8KZAHHjddb 10.47.1.154  49582     134.71.3.17  443       tcp   -       1266.367457 1594682    53255700   OTH        -          -          0            ^dtADTatTtTtTtTW 131516    8407458       142488    110641641     -
conn  2018-03-24T17:15:20.705347Z CWtQuI2IMNyE1pX47j 10.47.6.161  52121     134.71.3.17  443       tcp   -       1269.320626 2267243    54791018   OTH        -          -          0            DTadtATttTtTtT   152819    10575303      158738    113518994     -
conn  2018-03-24T17:33:05.415532Z Cy3R5w2pfv8oSEpa2j 10.47.8.19   49376     10.128.0.214 443       tcp   -       202.457994  4862366    1614249    S1         -          -          0            ShAdtttDTaTTTt   7280      10015980      6077      3453020       -
```

The same approach can be used to compare characters in `string`-type values,
such as this search that finds DNS requests that were issued for hostnames at
the high end of the alphabet.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'query > "zippy"' *.log.gz
```

#### Output:
```mdtest-output
_PATH TS                          UID               ID.ORIG_H  ID.ORIG_P ID.RESP_H  ID.RESP_P PROTO TRANS_ID RTT      QUERY                                                    QCLASS QCLASS_NAME QTYPE QTYPE_NAME RCODE RCODE_NAME AA TC RD RA Z ANSWERS                                                                TTLS       REJECTED
dns   2018-03-24T17:30:09.84174Z  Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001694 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:30:09.841742Z Csx7ymPvWeqIOHPi6 10.47.1.1  59144     10.10.1.1  53        udp   53970    0.001697 zn_9nquvazst1xipkt-cbs.siteintercept.qualtrics.com       1      C_INTERNET  1     A          0     NOERROR    F  F  T  F  0 0.0.0.0                                                                0          F
dns   2018-03-24T17:34:52.637234Z CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019491 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
dns   2018-03-24T17:34:52.637238Z CN9X7Y36SH6faoh8t 10.47.8.10 58340     10.0.0.100 53        udp   43239    0.019493 zn_0pxrmhobblncaad-hpsupport.siteintercept.qualtrics.com 1      C_INTERNET  1     A          0     NOERROR    F  F  T  T  0 cloud.qualtrics.com.edgekey.net,e3672.ksd.akamaiedge.net,23.55.215.198 3600,17,20 F
```

### Other Examples

The other behaviors we described previously for general
[value matching](#value-match) still apply the same for field/value matches.
Below are some exercises you can try to observe this with the sample data.
Search with `zq` against `*.log.gz` in all cases.

1. Compare the result of our previous [quoted word](#quoted-word) value search
   for `"O=Internet Widgits"` with a field/value search for
   `certificate.subject=*Widgits*`. Note how the former showed many types of
   Zeek records while the latter shows _only_ `x509` records, since only these
   records contain the field named `certificate.subject`.

2. Compare the result of our previous [glob wildcard](#glob-wildcards) value
   search for `www.*cdn*.com` with a field/value search for
   `server_name=www.*cdn*.com`. Note how the former showed mostly Zeek `dns`
   records and a couple `ssl` records, while the latter shows _only_ `ssl`
   records, since only these records contain the field named `server_name`.

3. Compare the result of our previous [regexp](#regular-expressions) value
   search for `/www.google(ad|tag)services.com/` with a field/value search for
   `query=/www.google(ad|tag)services.com/`. Note how the former showed a mix
   of Zeek `dns` and `ssl` records, while the latter shows _only_ `dns`
   records, since only these records contain the field named `query`.

## Boolean Logic

Your searches can be further refined by using boolean keywords `and`, `or`,
and `not`. These are case-insensitive, so `AND`, `OR`, and `NOT` can also be
used.

### `and`

If you enter multiple [value match](#value-match) or
[field/value match](#fieldvalue-match) terms separated by blank space, Zed
implicitly applies a boolean `and` between them, such that records are only
returned if they match on _all_ terms.

For example, when introducing [glob wildcards](#glob-wildcards), we performed a
search for `www.*cdn*.com` that returned mostly `dns` records along with a
couple `ssl` records. You could quickly isolate just the SSL records by
leveraging this implicit `and`.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'www.*cdn*.com _path=="ssl"' *.log.gz
```

#### Output:
```mdtest-output
_PATH TS                          UID                ID.ORIG_H   ID.ORIG_P ID.RESP_H    ID.RESP_P VERSION CIPHER                                CURVE     SERVER_NAME       RESUMED LAST_ALERT NEXT_PROTOCOL ESTABLISHED CERT_CHAIN_FUIDS                                                            CLIENT_CERT_CHAIN_FUIDS SUBJECT            ISSUER                                  CLIENT_SUBJECT CLIENT_ISSUER VALIDATION_STATUS
ssl   2018-03-24T17:23:00.244457Z CUG0fiQAzL4rNWxai  10.47.2.100 36150     52.85.83.228 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FXKmyTbr7HlvyL1h8,FADhCTvkq1ILFnD3j,FoVjYR16c3UIuXj4xk,FmiRYe1P53KOolQeVi   (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
ssl   2018-03-24T17:24:00.189735Z CSbGJs3jOeB6glWLJj 10.47.7.154 27137     52.85.83.215 443       TLSv12  TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 secp256r1 www.herokucdn.com F       -          h2            T           FuW2cZ3leE606wXSia,Fu5kzi1BUwnF0bSCsd,FyTViI32zPvCmNXgSi,FwV6ff3JGj4NZcVPE4 (empty)                 CN=*.herokucdn.com CN=Amazon,OU=Server CA 1B,O=Amazon,C=US -              -             ok
```

> **Note:** You may also include `and` explicitly if you wish:

        www.*cdn*.com and _path=ssl

### `or`

`or` returns the union of the matches from multiple terms.

For example, we can revisit two of our previous example searches that each only
returned a few records, searching now with `or` to see them all at once.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'orig_bytes > 1000000 or query > "zippy"' *.log.gz
```

#### Output:

```mdtest-output head
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

Use `not` to invert the matching logic in the term that comes to the right of
it in your search.

For example, suppose you've noticed that the vast majority of the sample Zeek
records are of log types like `conn`, `dns`, `files`, etc. You could review
some of the less-common Zeek record types by inverting the logic of a
[regexp match](#regular-expressions).

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'not _path matches /conn|dns|files|ssl|x509|http|weird/' *.log.gz
```

#### Output:

```mdtest-output head
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

> **Note:** `!` can also be used as alternative shorthand for `not`.

        zq -f table '! _path matches /conn|dns|files|ssl|x509|http|weird/' *.log.gz

### Parentheses & Order of Evaluation

Unless wrapped in parentheses, a search is evaluated in _left-to-right order_.

For example, the following search leverages the implicit boolean `and` to find
all `smb_mapping` records in which the `share_type` field is set to a value
other than `DISK`.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'not share_type=="DISK" _path=="smb_mapping"' *.log.gz
```

#### Output:
```mdtest-output head
_PATH       TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 2018-03-24T17:15:21.625534Z ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:22.021668Z C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:24.619169Z C2byFA2Y10G1GLUXgb 10.164.94.120 35337     10.47.27.80  445       \\\\PC-NEWMAN\\IPC$      -       -                  PIPE
smb_mapping 2018-03-24T17:15:25.562072Z C3kUnM2kEJZnvZmSp7 10.164.94.120 45903     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      -       -                  PIPE
...
```

Terms wrapped in parentheses will be evaluated _first_, overriding the default
left-to-right evaluation. If we wrap the search terms as shown below, now we
match almost every record we have. This is because the `not` is now inverting
the logic of everything in the parentheses, hence giving us all stored records
_other than_ `smb_mapping` records that have the value of their `share_type`
field set to `DISK`.

#### Example:
```mdtest-command zed-sample-data/zeek-default
zq -f table 'not (share_type=="DISK" _path=="smb_mapping")' *.log.gz
```

#### Output:
```mdtest-output head
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
```mdtest-command zed-sample-data/zeek-default
zq -f table '((not share_type=="DISK") and (service=="IPC")) _path=="smb_mapping"' *.log.gz
```

#### Output:
```mdtest-output head
_PATH       TS                          UID                ID.ORIG_H     ID.ORIG_P ID.RESP_H    ID.RESP_P PATH                     SERVICE NATIVE_FILE_SYSTEM SHARE_TYPE
smb_mapping 2018-03-24T17:15:21.625534Z ChZRry3Z4kv3i25TJf 10.164.94.120 36315     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:22.021668Z C0jyse1JYc82Acu4xl 10.164.94.120 34691     10.47.8.208  445       \\\\SNOZBERRY\\IPC$      IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:31.475945Z Cvaqhu3VhuXlDOMgXg 10.164.94.120 37127     10.47.3.151  445       \\\\COTTONCANDY4\\IPC$   IPC     -                  PIPE
smb_mapping 2018-03-24T17:15:36.306275Z CsZ7Be4NlqaJSNNie4 10.164.94.120 33921     10.47.23.166 445       \\\\PARKINGGARAGE\\IPC$  IPC     -                  PIPE
...
```

Except when writing the most common searches that leverage only the implicit
`and`, it's generally good practice to use parentheses even when not strictly
necessary, just to make sure your queries clearly communicate their intended
logic.
