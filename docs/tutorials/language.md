**This document will soon be updated as part of issue 3604**

# Data Types

Comprehensive documentation for working with data types in Zed is still a work
in progress. In the meantime, here's a few tips to get started with.

* Values are stored internally and treated in expressions using one of the Zed
  data types described in the
  [Primitive Values](../formats/zson.md#33-primitive-values) section of the
  ZSON spec.
* Users of [Zeek](../../zeek/README.md) logs should review the
  [Equivalent Types](../../zeek/Data-Type-Compatibility.md#equivalent-types)
  table for details on which Zed data types correspond to the
  [data types](https://docs.zeek.org/en/current/script-reference/types.html)
  that appear in Zeek logs.
* Zed allows for [type casting](https://en.wikipedia.org/wiki/Type_conversion)
  by specifying a destination Zed data type followed by the value to be
  converted to that type, enclosed in parentheses.

#### Example:

Consider the following NDJSON file `shipments.ndjson` that contains what
appear to be timestamped quantities of shipped items.

```mdtest-input shipments.ndjson
{"ts":"2021-10-07T13:55:22Z", "quantity": 873}
{"ts":"2021-10-07T17:23:44Z", "quantity": 436}
{"ts":"2021-10-07T23:01:34Z", "quantity": 123}
{"ts":"2021-10-08T09:12:45Z", "quantity": 998}
{"ts":"2021-10-08T12:44:12Z", "quantity": 744}
{"ts":"2021-10-09T20:01:19Z", "quantity": 2003}
{"ts":"2021-10-09T04:16:33Z", "quantity": 977}
{"ts":"2021-10-10T05:04:46Z", "quantity": 3004}
```

This data suffers from a notorious limitation of JSON: The lack of a native
"time" type requires storing timestamps as strings. As a result, if read into
`zq` as the strings they are, these `ts` values are not usable with
time-specific operations in Zed, such as this attempt to use
[time grouping]../zq/functions/bucket.md) to calculate total
quantities shipped per day.

```mdtest-command
zq -z 'sum(quantity) by every(1d) | sort ts' shipments.ndjson
```

#### Output:
```mdtest-output
{ts:error("every: time arg required"),sum:9158}
```

However, if we cast the `ts` field to the Zed `time` type, now the
calculation works as expected.

```mdtest-command
zq -f table 'ts:=time(ts) | sum(quantity) by every(1d) | sort ts' shipments.ndjson
```

#### Output:
```mdtest-output
ts                   sum
2021-10-07T00:00:00Z 1432
2021-10-08T00:00:00Z 1742
2021-10-09T00:00:00Z 2980
2021-10-10T00:00:00Z 3004
```


# Search Syntax

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

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
records. The default `zq` output to the terminal is the text-based
[ZSON](../formats/zson.md) format, whereas the compact binary
[ZNG](../formats/zng.md) format is used if the output is redirected or
piped.

In the examples, we'll be explicit in how we request our output format, using
`-z` for ZSON in this case.

#### Example:
```mdtest-command dir=testdata/edu
zq -z '*' schools.zson
```

#### Output:
```mdtest-output head
{School:"'3R' Middle",District:"Nevada County Office of Education",City:"Nevada City",County:"Nevada",Zip:"95959",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1995-10-30T00:00:00Z,ClosedDate:1996-06-28T00:00:00Z,Phone:null(string),StatusType:"Merged",Website:null(string)}
{School:"100 Black Men of the Bay Area Community",District:"Oakland Unified",City:"Oakland",County:"Alameda",Zip:"94607-1404",Latitude:37.745418,Longitude:-122.14067,Magnet:null(bool),OpenDate:2012-08-06T00:00:00Z,ClosedDate:2014-10-28T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.100school.org"}
{School:"101 Elementary",District:"Victor Elementary",City:"Victorville",County:"San Bernardino",Zip:"92395-3360",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1996-02-07T00:00:00Z,ClosedDate:2005-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.charter101.org"}
{School:"180 Program",District:"Novato Unified",City:"Novato",County:"Marin",Zip:"94947-4004",Latitude:38.097792,Longitude:-122.57617,Magnet:null(bool),OpenDate:2012-08-22T00:00:00Z,ClosedDate:2014-06-13T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
...
```

The `-Z` option is also available for "pretty-printed" ZSON output.

#### Example:
```mdtest-command dir=testdata/edu
zq -Z '*' schools.zson
```

#### Output:
```mdtest-output head
{
    School: "'3R' Middle",
    District: "Nevada County Office of Education",
    City: "Nevada City",
    County: "Nevada",
    Zip: "95959",
    Latitude: null (float64),
    Longitude: null (float64),
    Magnet: null (bool),
    OpenDate: 1995-10-30T00:00:00Z,
    ClosedDate: 1996-06-28T00:00:00Z,
    Phone: null (string),
    StatusType: "Merged",
    Website: null (string)
}
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
[operator](../zq/reference.md#operators) or
[aggregate function](../zq/reference.md#aggregate-functions). The following example
is shorthand for:

```
zq -z '* | cut School,City' schools.zson
```

#### Example:

```mdtest-command dir=testdata/edu
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

For example, searching across both our school and test score data sources for
`596` matches records that contain numeric fields of this precise value (such
as from the test scores) and also records that contain string fields
(such as the ZIP code and phone number fields in the school data.)

#### Example:
```mdtest-command dir=testdata/edu
zq -z '596' testscores.zson schools.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:591(uint16),AvgScrRead:610(uint16),AvgScrWrite:596(uint16),cname:"Los Angeles",dname:"William S. Hart Union High",sname:"Academy of the Canyons"}
{AvgScrMath:614(uint16),AvgScrRead:596(uint16),AvgScrWrite:592(uint16),cname:"Alameda",dname:"Pleasanton Unified",sname:"Amador Valley High"}
{AvgScrMath:620(uint16),AvgScrRead:596(uint16),AvgScrWrite:590(uint16),cname:"Yolo",dname:"Davis Joint Unified",sname:"Davis Senior High"}
{School:"Achieve Charter School of Paradise Inc.",District:"Paradise Unified",City:"Paradise",County:"Butte",Zip:"95969-3913",Latitude:39.760323,Longitude:-121.62078,Magnet:false,OpenDate:2005-09-12T00:00:00Z,ClosedDate:null(time),Phone:"(530) 872-4100",StatusType:"Active",Website:"www.achievecharter.org"}
{School:"Alliance Ouchi-O'Donovan 6-12 Complex",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90043-2622",Latitude:33.993484,Longitude:-118.32246,Magnet:false,OpenDate:2006-09-05T00:00:00Z,ClosedDate:null(time),Phone:"(323) 596-2290",StatusType:"Active",Website:"http://ouchihs.org"}
...
```

By comparison, the section below on [Field/Value Match](#fieldvalue-match)
describes ways to perform searches against only fields of a specific
[data type](../zq/language.md#data-types).

### Quoted Word

Sometimes you may need to search for sequences of multiple words or words that
contain special characters. To achieve this, wrap your search term in quotes.

Let's say we've noticed that a couple of the school names in our sample data
include the string `Defunct=`. An attempt to enter this as a [bare word](#bare-word)
search causes an error because the language parser interprets this as the
start of an attempted [field/value match](#fieldvalue-match) for a field named
`Defunct`.

#### Example:
```mdtest-command dir=testdata/edu fails
zq -z 'Defunct=' *.zson
```

#### Output:
```mdtest-output
zq: error parsing Zed at column 8:
Defunct=
   === ^ ===
```

However, wrapping in quotes gives the desired result.

#### Example:
```mdtest-command dir=testdata/edu
zq -z '"Defunct="' schools.zson
```

#### Output:
```mdtest-output
{School:"Lincoln Elem 'Defunct=",District:"Modesto City Elementary",City:null(string),County:"Stanislaus",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Lovell Elem 'Defunct=",District:"Cutler-Orosi Joint Unified",City:null(string),County:"Tulare",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
```

Wrapping in quotes is particularly handy when you're looking for long, specific
strings that may have several special characters in them. For example, let's
say we're looking for information on the Union Hill Elementary district.
Entered without quotes, we end up matching far more records than we intended
since each space character between words is treated as a [Boolean `and`](#and).

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'Union Hill Elementary' schools.zson
```

#### Output:
```mdtest-output head
{School:"A. M. Thomas Middle",District:"Lost Hills Union Elementary",City:"Lost Hills",County:"Kern",Zip:"93249-0158",Latitude:35.615269,Longitude:-119.69955,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 797-2626",StatusType:"Active",Website:null(string)}
{School:"Alview Elementary",District:"Alview-Dairyland Union Elementary",City:"Chowchilla",County:"Madera",Zip:"93610-9225",Latitude:37.050632,Longitude:-120.4734,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(559) 665-2275",StatusType:"Active",Website:null(string)}
{School:"Anaverde Hills",District:"Westside Union Elementary",City:"Palmdale",County:"Los Angeles",Zip:"93551-5518",Latitude:34.564651,Longitude:-118.18012,Magnet:false,OpenDate:2005-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(661) 575-9923",StatusType:"Active",Website:null(string)}
{School:"Apple Blossom",District:"Twin Hills Union Elementary",City:"Sebastopol",County:"Sonoma",Zip:"95472-3917",Latitude:38.387396,Longitude:-122.84954,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(707) 823-1041",StatusType:"Active",Website:null(string)}
...
```

However, wrapping the entire term in quotes allows us to search for the
complete string, including the spaces.

#### Example:
```mdtest-command dir=testdata/edu
zq -z '"Union Hill Elementary"' schools.zson
```

#### Output:
```mdtest-output
{School:"Highland Oaks Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1997-09-02T00:00:00Z,ClosedDate:2003-07-02T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Union Hill 3R Community Day",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:39.229055,Longitude:-121.07127,Magnet:null(bool),OpenDate:2003-08-20T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Charter Home",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1995-07-14T00:00:00Z,ClosedDate:2015-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Middle",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"94945-8805",Latitude:39.205006,Longitude:-121.03778,Magnet:false,OpenDate:2013-08-14T00:00:00Z,ClosedDate:null(time),Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
{School:null(string),District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8730",Latitude:39.208869,Longitude:-121.03551,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(530) 273-0647",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
```

### Glob Wildcards

To find values that may contain arbitrary substrings between or alongside the
desired word(s), one or more
[glob](https://en.wikipedia.org/wiki/Glob_(programming))-style wildcards can be
used.

For example, the following search finds records that contain school names
that have some additional text between `ACE` and `Academy`.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'ACE*Academy' schools.zson
```

#### Output:
```mdtest-output head
{School:"ACE Empower Academy",District:"Santa Clara County Office of Education",City:"San Jose",County:"Santa Clara",Zip:"95116-3423",Latitude:37.348601,Longitude:-121.8446,Magnet:false,OpenDate:2008-08-26T00:00:00Z,ClosedDate:null(time),Phone:"(408) 729-3920",StatusType:"Active",Website:"www.acecharter.org"}
{School:"ACE Inspire Academy",District:"San Jose Unified",City:"San Jose",County:"Santa Clara",Zip:"95112-6334",Latitude:37.350981,Longitude:-121.87205,Magnet:false,OpenDate:2015-08-03T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-6008",StatusType:"Active",Website:"www.acecharter.org"}
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

For example, since there are many high schools in our sample data, to find
only records containing strings that _begin_ with the word `High`:

#### Example:
```mdtest-command dir=testdata/edu
zq -z '/^High /' schools.zson
```

#### Output:
```mdtest-output head
{School:"High Desert",District:"Soledad-Agua Dulce Union Eleme",City:"Acton",County:"Los Angeles",Zip:"93510",Latitude:34.490977,Longitude:-118.19646,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1993-06-30T00:00:00Z,Phone:null(string),StatusType:"Merged",Website:null(string)}
{School:"High Desert",District:"Acton-Agua Dulce Unified",City:"Acton",County:"Los Angeles",Zip:"93510-1757",Latitude:34.492578,Longitude:-118.19039,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 269-0310",StatusType:"Active",Website:null(string)}
{School:"High Desert Academy",District:"Eastern Sierra Unified",City:"Benton",County:"Mono",Zip:"93512-0956",Latitude:37.818597,Longitude:-118.47712,Magnet:null(bool),OpenDate:1996-09-03T00:00:00Z,ClosedDate:2012-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.esusd.org"}
{School:"High Desert Academy of Applied Arts and Sciences",District:"Victor Valley Union High",City:"Victorville",County:"San Bernardino",Zip:"92394",Latitude:34.531144,Longitude:-117.31697,Magnet:null(bool),OpenDate:2004-09-07T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.hdaaas.org"}
...
```

Regexps are a detailed topic all their own. For details, reference the
[documentation for re2](https://github.com/google/re2/wiki/Syntax), which is
the library that Zed uses to provide regexp support.

## Field/Value Match

The search result can be narrowed to include only records that contain a
certain value in a particular named field. For example, the following search
will only match records containing the field called `District` where it is set
to the precise string value `Winton`.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'District=="Winton"' schools.zson
```

#### Output:

```mdtest-output
{School:"Frank Sparkes Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.382084,Longitude:-120.61847,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6180",StatusType:"Active",Website:null(string)}
{School:"Sybil N. Crookham Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0130",Latitude:37.389501,Longitude:-120.61636,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6182",StatusType:"Active",Website:null(string)}
{School:"Winfield Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388",Latitude:37.389121,Longitude:-120.60442,Magnet:false,OpenDate:2007-08-13T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6891",StatusType:"Active",Website:null(string)}
{School:"Winton Middle",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-1477",Latitude:37.379938,Longitude:-120.62263,Magnet:false,OpenDate:1990-07-20T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6189",StatusType:"Active",Website:null(string)}
{School:null(string),District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.389467,Longitude:-120.6147,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(209) 357-6175",StatusType:"Active",Website:"www.winton.k12.ca.us"}
```

Because the right-hand-side value to which we were comparing was a string, it
was necessary to wrap it in quotes. If we'd left it bare, it would have been
interpreted as a field name.

For example, to see the records in which the school and district name are the
same:

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'District==School' schools.zson
```

#### Output:

```mdtest-output head
{School:"Adelanto Elementary",District:"Adelanto Elementary",City:"Adelanto",County:"San Bernardino",Zip:"92301-1734",Latitude:34.576166,Longitude:-117.40944,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(760) 246-5892",StatusType:"Active",Website:null(string)}
{School:"Allensworth Elementary",District:"Allensworth Elementary",City:"Allensworth",County:"Tulare",Zip:"93219-9709",Latitude:35.864487,Longitude:-119.39068,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 849-2401",StatusType:"Active",Website:null(string)}
{School:"Alta Loma Elementary",District:"Alta Loma Elementary",City:"Alta Loma",County:"San Bernardino",Zip:"91701-5007",Latitude:34.12597,Longitude:-117.59744,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(909) 484-5000",StatusType:"Active",Website:null(string)}
...
```

### Role of Data Types

To match successfully when comparing values to the contents of named fields,
the value must be comparable to the _data type_ of the field.

For instance, the "Zip" field in our schools data is of `string` type because
several values are of the extended ZIP+4 format that includes a hyphen and four
additional digits and hence could not be represented in a numeric type.

```mdtest-command dir=testdata/edu
zq -z 'cut Zip' schools.zson
```

#### Output:
```mdtest-output head
{Zip:"95959"}
{Zip:"94607-1404"}
{Zip:"92395-3360"}
...
```

An attempted [field/value match](#fieldvalue-match) `Zip==95959` would _not_
match the top record shown, since Zed recognizes the bare value `95959` as a
number before comparing it to all the fields named `Zip` that it sees in the
input stream. However, `Zip=="95959"` _would_ match, since the quotes cause Zed
to treat the value as a string.

See the [Data Types](../zq/language.md#data-types) page for more details.

### Finding Patterns with `matches`

When comparing a named field to a quoted value, the quoted value is treated as
an _exact_ match.

For example, let's say we know there are several school names that start with
`Luther` but only a couple district names that do. Because `Luther` only appears
as a _substring_ of the district names in our sample data, the following example
produces no output.

#### Example:

```mdtest-command dir=testdata/edu
zq -z 'District=="Luther"' schools.zson
```

#### Output:
```mdtest-output
```

To achieve this with a field/value match, we enter `matches` before specifying
a [glob wildcard](#glob-wildcards).

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'District matches Luther*' schools.zson
```

#### Output:

```mdtest-output head
{School:"Luther Burbank Elementary",District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-1814",StatusType:"Active",Website:null(string)}
{School:null(string),District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(408) 295-2450",StatusType:"Active",Website:"www.lbsd.k12.ca.us"}
```

[Regular expressions](#regular-expressions) can also be used with `matches`.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'School matches /^Sunset (Ranch|Ridge) Elementary/' schools.zson
```

#### Output:
```mdtest-output
{School:"Sunset Ranch Elementary",District:"Rocklin Unified",City:"Rocklin",County:"Placer",Zip:"95765-5441",Latitude:38.826425,Longitude:-121.2864,Magnet:false,OpenDate:2010-08-17T00:00:00Z,ClosedDate:null(time),Phone:"(916) 624-2048",StatusType:"Active",Website:"www.rocklin.k12.ca.us"}
{School:"Sunset Ridge Elementary",District:"Pacifica",City:"Pacifica",County:"San Mateo",Zip:"94044-2029",Latitude:37.653836,Longitude:-122.47919,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(650) 738-6687",StatusType:"Active",Website:null(string)}
```

### Containment

Rather than testing for strict equality or pattern matches, you may want to
determine if a value is among the many possible elements of a complex field.
This is performed with `in`.

Since our sample data doesn't contain complex fields, we'll make one by
using the [`union`](../zq/aggregates/union.md) aggregate function to
create a [`set`](../formats/zson.md#343-set-value)-typed
field called `Schools` that contains all unique school names per district. From
these we'll find each set that contains a school named `Lincoln Elementary`.

#### Example:
```mdtest-command dir=testdata/edu
zq -Z 'Schools:=union(School) by District | sort | "Lincoln Elementary" in Schools' schools.zson
```

#### Output:
```mdtest-output head
{
    District: "Alpine County Unified",
    Schools: |[
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

The following example locates all schools whose websites are hosted on an
IP address in `38.0.0.0/8` network.

#### Example:
```mdtest-command dir=testdata/edu
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

For example, the following search finds the schools that reported the highest
math test scores.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'AvgScrMath > 690' testscores.zson
```

#### Output:
```mdtest-output
{AvgScrMath:698(uint16),AvgScrRead:639(uint16),AvgScrWrite:664(uint16),cname:"Santa Clara",dname:"Fremont Union High",sname:"Lynbrook High"}
{AvgScrMath:699(uint16),AvgScrRead:653(uint16),AvgScrWrite:671(uint16),cname:"Alameda",dname:"Fremont Unified",sname:"Mission San Jose High"}
{AvgScrMath:691(uint16),AvgScrRead:638(uint16),AvgScrWrite:657(uint16),cname:"Santa Clara",dname:"Fremont Union High",sname:"Monta Vista High"}
```

The same approach can be used to compare characters in `string`-type values,
such as this search that finds school names at the end of the alphabet.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'School > "Z"' schools.zson
```

#### Output:
```mdtest-output head
{School:"Zamora Elementary",District:"Woodland Joint Unified",City:"Woodland",County:"Yolo",Zip:"95695-5137",Latitude:38.658609,Longitude:-121.79355,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(530) 666-3641",StatusType:"Active",Website:null(string)}
{School:"Zamorano Elementary",District:"San Diego Unified",City:"San Diego",County:"San Diego",Zip:"92139-2989",Latitude:32.680338,Longitude:-117.03864,Magnet:true,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(619) 430-1400",StatusType:"Active",Website:"http://new.sandi.net/schools/zamorano"}
{School:"Zane (Catherine L.) Junior High",District:"Eureka City High",City:"Eureka",County:"Humboldt",Zip:"95501-3140",Latitude:40.788118,Longitude:-124.14903,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1998-06-30T00:00:00Z,Phone:null(string),StatusType:"Merged",Website:null(string)}
...
```

## Boolean Logic

Your searches can be further refined by using boolean keywords `and`, `or`,
and `not`. These are case-insensitive, so `AND`, `OR`, and `NOT` can also be
used.

### `and`

If you enter multiple [value match](#value-match) or
[field/value match](#fieldvalue-match) terms separated by blank space, Zed
implicitly applies a boolean `and` between them, such that records are only
returned if they match on _all_ terms.

For example, let's say we're searching for information about academies
that are flagged as being in a `Pending` status.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'StatusType=="Pending" academy' schools.zson
```

#### Output:
```mdtest-output
{School:"Equitas Academy 4",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90015-2412",Latitude:34.044837,Longitude:-118.27844,Magnet:false,OpenDate:2017-09-01T00:00:00Z,ClosedDate:null(time),Phone:"(213) 201-0440",StatusType:"Pending",Website:"http://equitasacademy.org"}
{School:"Pinnacle Academy Charter - Independent Study",District:"South Monterey County Joint Union High",City:"King City",County:"Monterey",Zip:"93930-3311",Latitude:36.208934,Longitude:-121.13286,Magnet:false,OpenDate:2016-08-08T00:00:00Z,ClosedDate:null(time),Phone:"(831) 385-4661",StatusType:"Pending",Website:"www.smcjuhsd.org"}
{School:"Rocketship Futuro Academy",District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:false,OpenDate:2016-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
{School:"Sherman Thomas STEM Academy",District:"Madera Unified",City:"Madera",County:"Madera",Zip:"93638",Latitude:36.982843,Longitude:-120.06665,Magnet:false,OpenDate:2017-08-09T00:00:00Z,ClosedDate:null(time),Phone:"(559) 674-1192",StatusType:"Pending",Website:"www.stcs.k12.ca.us"}
{School:null(string),District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
```

> **Note:** You may also include `and` explicitly if you wish:
> ```
> StatusType=="Pending" and academy
> ```

### `or`

`or` returns the union of the matches from multiple terms.

For example, we can revisit two of our previous example searches that each only
returned a couple records, searching now with `or` to see them all at once.

#### Example:
```mdtest-command dir=testdata/edu
zq -z '"Defunct=" or ACE*Academy' schools.zson
```

#### Output:

```mdtest-output
{School:"ACE Empower Academy",District:"Santa Clara County Office of Education",City:"San Jose",County:"Santa Clara",Zip:"95116-3423",Latitude:37.348601,Longitude:-121.8446,Magnet:false,OpenDate:2008-08-26T00:00:00Z,ClosedDate:null(time),Phone:"(408) 729-3920",StatusType:"Active",Website:"www.acecharter.org"}
{School:"ACE Inspire Academy",District:"San Jose Unified",City:"San Jose",County:"Santa Clara",Zip:"95112-6334",Latitude:37.350981,Longitude:-121.87205,Magnet:false,OpenDate:2015-08-03T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-6008",StatusType:"Active",Website:"www.acecharter.org"}
{School:"Lincoln Elem 'Defunct=",District:"Modesto City Elementary",City:null(string),County:"Stanislaus",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Lovell Elem 'Defunct=",District:"Cutler-Orosi Joint Unified",City:null(string),County:"Tulare",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
```

### `not`

Use `not` to invert the matching logic in the term that comes to the right of
it in your search.

For example, to find schools in the `Dixon Unified` district _other than_
elementary schools, we invert the logic of a search term.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'not elementary District=="Dixon Unified"' schools.zson
```

#### Output:

```mdtest-output head
{School:"C. A. Jacobs Intermediate",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620-3209",Latitude:38.446472,Longitude:-121.83631,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(707) 693-6350",StatusType:"Active",Website:"www.dixonusd.org"}
{School:"Dixon Adult",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620",Latitude:38.444818,Longitude:-121.82287,Magnet:null(bool),OpenDate:1996-09-09T00:00:00Z,ClosedDate:2016-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Dixon Community Day",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620",Latitude:38.44755,Longitude:-121.82001,Magnet:false,OpenDate:2003-08-23T00:00:00Z,ClosedDate:null(time),Phone:"(707) 693-6340",StatusType:"Active",Website:"www.dixonusd.org"}
{School:"Dixon High",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620-9301",Latitude:38.436088,Longitude:-121.81672,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(707) 693-6330",StatusType:"Active",Website:null(string)}
{School:"Dixon Montessori Charter",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620-2702",Latitude:38.447984,Longitude:-121.83186,Magnet:false,OpenDate:2010-08-11T00:00:00Z,ClosedDate:null(time),Phone:"(707) 678-8953",StatusType:"Active",Website:"www.dixonmontessori.org"}
{School:"Dixon Unified Alter. Educ.",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1993-08-26T00:00:00Z,ClosedDate:1994-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Maine Prairie High (Continuation)",District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620-3019",Latitude:38.447549,Longitude:-121.81986,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(707) 693-6340",StatusType:"Active",Website:null(string)}
{School:null(string),District:"Dixon Unified",City:"Dixon",County:"Solano",Zip:"95620-3447",Latitude:38.44468,Longitude:-121.82249,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(707) 693-6300",StatusType:"Active",Website:"www.dixonusd.org"}
```

> **Note:** `!` can also be used as alternative shorthand for `not`.
> ```
> ! elementary District=="Dixon Unified"
> ```

### Parentheses & Order of Evaluation

Unless wrapped in parentheses, a search is evaluated in _left-to-right order_.
Terms wrapped in parentheses will be evaluated _first_, overriding the default
left-to-right evaluation.

For example, we've noticed there are some test score records that have `null`
values for all three test scores.

#### Example:
```mdtest-command dir=testdata/edu
zq -z 'AvgScrMath==null AvgScrRead==null AvgScrWrite==null' testscores.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:null(uint16),AvgScrRead:null(uint16),AvgScrWrite:null(uint16),cname:"Riverside",dname:"Beaumont Unified",sname:"21st Century Learning Institute"}
{AvgScrMath:null(uint16),AvgScrRead:null(uint16),AvgScrWrite:null(uint16),cname:"Los Angeles",dname:"ABC Unified",sname:"ABC Secondary (Alternative)"}
...
```

We can easily filter these out by negating the search for these records.


#### Example:
```mdtest-command dir=testdata/edu
zq -z 'not (AvgScrMath==null AvgScrRead==null AvgScrWrite==null)' testscores.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:371(uint16),AvgScrRead:376(uint16),AvgScrWrite:368(uint16),cname:"Los Angeles",dname:"Los Angeles Unified",sname:"APEX Academy"}
{AvgScrMath:367(uint16),AvgScrRead:359(uint16),AvgScrWrite:369(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"ARISE High"}
{AvgScrMath:491(uint16),AvgScrRead:489(uint16),AvgScrWrite:484(uint16),cname:"Santa Clara",dname:"San Jose Unified",sname:"Abraham Lincoln High"}
...
```

Parentheses can also be nested.

#### Example:
```mdtest-command dir=testdata/edu
zq -z '(sname matches *High*) and (not (AvgScrMath==null AvgScrRead==null AvgScrWrite==null) and dname=="San Francisco Unified")' testscores.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:504(uint16),AvgScrRead:467(uint16),AvgScrWrite:467(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"Balboa High"}
{AvgScrMath:480(uint16),AvgScrRead:443(uint16),AvgScrWrite:431(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"Burton (Phillip and Sala) Academic High"}
{AvgScrMath:413(uint16),AvgScrRead:410(uint16),AvgScrWrite:395(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"City Arts and Tech High"}
...
```

Except when writing the most common searches that leverage only the implicit
`and`, it's generally good practice to use parentheses even when not strictly
necessary, just to make sure your queries clearly communicate their intended
logic.

# Expressions

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

Comprehensive documentation for Zed expressions is still a work in progress. In
the meantime, here's an example expression with simple math to get started:

```mdtest-command dir=testdata/edu
zq -f table 'AvgScrMath != null | put combined_scores:=AvgScrMath+AvgScrRead+AvgScrWrite | cut sname,combined_scores,AvgScrMath,AvgScrRead,AvgScrWrite | head 5' testscores.zson
```

#### Output:
```mdtest-output
sname                       combined_scores AvgScrMath AvgScrRead AvgScrWrite
APEX Academy                1115            371        376        368
ARISE High                  1095            367        359        369
Abraham Lincoln High        1464            491        489        484
Abraham Lincoln Senior High 1319            462        432        425
Academia Avance Charter     1148            386        380        382
```

# Summarize Aggregations

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

The `summarize` operator performs zero or more aggregations with
zero or more [grouping expressions](../zq/operators/summarize.md).
Each aggregation is performed by an
_aggregate function_ that operates on
batches of records to carry out a running computation over the values they
contain.  The `summarize` keyword is optional.

   * [General Usage](#general-usage)
     + [Invoking](#invoking)
     + [Field Naming](#field-naming)
     + [Grouping](#grouping)
     + [`where` filtering](#where-filtering)
   * [Available Aggregate Functions](#available-aggregate-functions)
     + [`and`](#and)
     + [`any`](#any)
     + [`avg`](#avg)
     + [`collect`](#collect)
     + [`count`](#count)
     + [`dcount`](#dcount)
     + [`max`](#max)
     + [`min`](#min)
     + [`or`](#or)
     + [`sum`](#sum)
     + [`union`](#union)

## General Usage

### Invoking

Multiple aggregate functions may be invoked at the same time.

#### Example:

To simultaneously calculate the minimum, maximum, and average of the math
test scores:

```mdtest-command dir=testdata/edu
zq -f table 'min(AvgScrMath),max(AvgScrMath),avg(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
min max avg
289 699 484.99019042123484
```

### Field Naming

As just shown, by default the result returned is placed in a field with the
same name as the aggregate function. You may instead use `:=` to specify an
explicit name for the generated field.

#### Example:

```mdtest-command dir=testdata/edu
zq -f table 'lowest:=min(AvgScrMath),highest:=max(AvgScrMath),typical:=avg(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
lowest highest typical
289    699     484.99019042123484
```

### Grouping

All aggregate functions may be invoked with one or more
[grouping](../zq/operators/summarize.md) options that define the batches of records on
which they operate. If explicit grouping is not used, an aggregate function
will operate over all records in the input stream.

### `where` filtering

A `where` clause may also be added to filter the values on which an aggregate
function will operate.

#### Example:

To calculate average math test scores for the cities of Los Angeles and San
Francisco:

```mdtest-command dir=testdata/edu
zq -Z 'LA_Math:=avg(AvgScrMath) where cname=="Los Angeles", SF_Math:=avg(AvgScrMath) where cname=="San Francisco"' testscores.zson
```

#### Output:
```mdtest-output
{
    LA_Math: 456.27341772151897,
    SF_Math: 485.3636363636364
}
```

---

## Available Aggregate Functions

### `and`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the Boolean value `true` if the provided expression evaluates to `true` for all inputs. Contrast with [`or`](#or). |
| **Syntax**                | `and(<expression>)`                                            |
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](../zq/language.md#expressions). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities in which all schools have a website.

```mdtest-command dir=testdata/edu
zq -Z 'all_schools_have_website:=and(Website!=null) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Acampo",
    all_schools_have_website: false
}
{
    City: "Acton",
    all_schools_have_website: false
}
{
    City: "Acton, CA",
    all_schools_have_website: true
}
...
```

---

### `any`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return one value observed for a specified field.               |
| **Syntax**                | `any(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the name of one of the schools in our sample data:

```mdtest-command dir=testdata/edu
zq -z 'any(School)' schools.zson
```

For small inputs that fit in memory, this will typically be the first such
field in the stream, but in general you should not rely upon this.  In this
case, the output is:

#### Output:
```mdtest-output
{any:"'3R' Middle"}
```

---

### `avg`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the mean (average) of the values of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `avg(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To calculate the average of the math test scores:

```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
avg
484.99019042123484
```

---

### `collect`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Assemble all input values into an array. Contrast with [`union`](#union). |
| **Syntax**                | `collect(<field-name>)`                                        |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example

For schools in Fresno county that include websites, the following query
constructs an ordered list per city of their websites along with a parallel
list of which school each website represents.

```mdtest-command dir=testdata/edu
zq -Z 'County=="Fresno" Website!=null | Websites:=collect(Website),Schools:=collect(School) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Auberry",
    Websites: [
        "www.sierra.k12.ca.us",
        "www.sierra.k12.ca.us",
        "www.pineridge.k12.ca.us",
        "www.pineridge.k12.ca.us"
    ],
    Schools: [
        "Auberry Elementary",
        "Balch Camp Elementary",
        "Pine Ridge Elementary"
    ]
}
{
    City: "Big Creek",
    Websites: [
        "www.bigcreekschool.com",
        "www.bigcreekschool.com"
    ],
    Schools: [
        "Big Creek Elementary"
    ]
}
...
```

---

### `count`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the number of records.                                  |
| **Syntax**                | `count([field-name])`                                          |
| **Required<br>arguments** | None                                                           |
| **Optional<br>arguments** | `[field-name]`<br>The name of a field. If specified, only records that contain this field will be counted. |

#### Example #1:

To count the number of records in each of our example data sources:

```mdtest-command dir=testdata/edu
zq -z 'count()' schools.zson
zq -z 'count()' testscores.zson
zq -z 'count()' webaddrs.zson
```

#### Output:
```mdtest-output
{count:17686(uint64)}
{count:2331(uint64)}
{count:2223(uint64)}
```

#### Example #2:

The `Website` field is known to be in our school and website address data
sources, but not in the test score data. To confirm this, we can count across
all data sources and specify the named field.

```mdtest-command dir=testdata/edu
zq -z 'count(Website)' *.zson
```

```mdtest-output
{count:19909(uint64)}
```

Since `17686 + 2223 = 19909`, the count result is what we expected.

---

### `dcount`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return a quick approximation of the number of unique values of a field.|
| **Syntax**                | `dcount(<field-name>)`                                  |
| **Required<br>arguments** | `<field-name>`<br>The name of a field containing values to be counted. |
| **Optional<br>arguments** | None                                                           |
| **Limitations**           | The potential inaccuracy of the calculated result is described in detail in the code and research linked from the [HyperLogLog repository](https://github.com/axiomhq/hyperloglog).<br><br>Also, partial aggregations are not yet implemented for `dcount` ([zed/2743](https://github.com/brimdata/zed/issues/2743)), so it may not work correctly in all circumstances. |

#### Example:

To see an approximate count of unique school names in our sample data set:

```mdtest-command dir=testdata/edu
zq -Z 'dcount(School)' schools.zson
```

#### Output:
```mdtest-output
{
    dcount: 13804 (uint64)
}
```

To see the precise value, which may take longer to execute:

```mdtest-command dir=testdata/edu
zq -Z 'count() by School | count()' schools.zson
```

#### Output:
```mdtest-output
{
    count: 13876 (uint64)
}
```

Here we saw the approximation was "off" by 0.3%.

---

### `max`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the maximum value of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `max(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the highest reported math test score:

```mdtest-command dir=testdata/edu
zq -f table 'max(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
max
699
```

---

### `min`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the minimum value of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `min(<field-name>)`                                            |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To see the lowest reported math test score:

```mdtest-command dir=testdata/edu
zq -f table 'min(AvgScrMath)' testscores.zson
```

#### Output:
```mdtest-output
min
289
```

---

### `or`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Returns the Boolean value `true` if the provided expression evaluates to `true` for one or more inputs. Contrast with [`and`](#and). |
| **Syntax**                | `or(<expression>)`                                             |
| **Required<br>arguments** | `<expression>`<br>A valid Zed [expression](../zq/language.md#expressions). |
| **Optional<br>arguments** | None                                                           |

#### Example:

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities for which at least one school has
a listed website.

```mdtest-command dir=testdata/edu
zq -Z 'has_at_least_one_school_website:=or(Website!=null) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Acampo",
    has_at_least_one_school_website: true
}
{
    City: "Acton",
    has_at_least_one_school_website: true
}
{
    City: "Acton, CA",
    has_at_least_one_school_website: true
}
{
    City: "Adelanto",
    has_at_least_one_school_website: true
}
{
    City: "Adin",
    has_at_least_one_school_website: false
}
...
```

---

### `sum`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Return the total sum of the values of a specified field. Non-numeric values (including `null`) are ignored. |
| **Syntax**                | `sum(<field-name>)`                                            |
| **Required<br>arguments** | `<field- name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |

#### Example:

To calculate the total of all the math, reading, and writing test scores
across all schools:

```mdtest-command dir=testdata/edu
zq -Z 'AllMath:=sum(AvgScrMath),AllRead:=sum(AvgScrRead),AllWrite:=sum(AvgScrWrite)' testscores.zson
```

#### Output:
```mdtest-output
{
    AllMath: 840488 (uint64),
    AllRead: 832260 (uint64),
    AllWrite: 819632 (uint64)
}
```

---

## `union`

|                           |                                                                |
| ------------------------- | -------------------------------------------------------------- |
| **Description**           | Gather all unique input values into a set. Contrast with [`collect`](#collect). |
| **Syntax**                | `union(<field-name>)`                                          |
| **Required<br>arguments** | `<field-name>`<br>The name of a field.                         |
| **Optional<br>arguments** | None                                                           |
| **Limitations**           | The data type of the input values must be uniform.             |

#### Example:

For schools in Fresno county that include websites, the following query
constructs a set per city of all the unique websites for the schools in that
city.

```mdtest-command dir=testdata/edu
zq -Z 'County=="Fresno" Website!=null | Websites:=union(Website) by City | sort City' schools.zson
```

#### Output:
```mdtest-output head
{
    City: "Auberry",
    Websites: |[
        "www.sierra.k12.ca.us",
        "www.pineridge.k12.ca.us"
    ]|
}
{
    City: "Big Creek",
    Websites: |[
        "www.bigcreekschool.com"
    ]|
}
...
```

# Grouping

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

Zed includes _grouping_ options that partition the input stream into batches
that are aggregated separately based on field values. Grouping is most often
used with [aggregate functions](../zq/reference.md#aggregate-functions). If explicit
grouping is not used, an aggregate function will operate over all records in the
input stream.

Below you will find details regarding the available grouping mechanisms and
tips for their effective use.

- [Value Grouping - `by`](#value-grouping---by)
- [Note: Undefined Order](#note-undefined-order)

# Value Grouping - `by`

To create batches of records based on the values of fields or the results of
[expressions](../zq/language.md#expressions), specify
`by <field-name | name:=expression> [, <field-name | name:=expression> ...]`
after invoking your aggregate function(s).

#### Example #1:

The simplest example summarizes the unique values of the named field(s), which
requires no aggregate function. To see the different categories of status for
the schools in our example data:

```mdtest-command dir=testdata/edu
zq -z 'by StatusType | sort' schools.zson
```

#### Output:
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```

If you work a lot at the UNIX/Linux shell, you might have sought to accomplish
the same via a familiar, verbose idiom. This works in Zed, but the `by`
shorthand is preferable.

```mdtest-command dir=testdata/edu
zq -z 'cut StatusType | sort | uniq' schools.zson
```

#### Output:
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```

#### Example #2:

By specifying multiple comma-separated field names, one batch is formed for each
unique combination of values found in those fields. To see the average reading
test scores and school count for each county/district pairing:

```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrRead),count() by cname,dname | sort -r count' testscores.zson
```

#### Output:
```mdtest-output head
cname           dname                                              avg                count
Los Angeles     Los Angeles Unified                                416.83522727272725 202
San Diego       San Diego Unified                                  472                44
Alameda         Oakland Unified                                    414.95238095238096 27
San Francisco   San Francisco Unified                              454.36842105263156 26
...
```

#### Example #3:

Instead of a simple field name, any of the comma-separated `by` groupings could
be based on the result of an [expression](../zq/language.md#expressions). The
expression must be preceded by the name of the expression result
for further processing/presentation downstream in your Zed pipeline.

To see a count of how many school names of a particular character length
appear in our example data:

```mdtest-command dir=testdata/edu
zq -f table 'count() by Name_Length:=len(School) | sort -r' schools.zson
```

#### Output:
```mdtest-output head
Name_Length count
89          2
85          2
84          2
83          1
...
```

#### Example #4

The fields referenced in a `by` grouping may or may not be present, or may be
inconsistently present, in a given record and the grouping will still have effect.
When a value is missing for a specified key, it will appear as `error("missing")`.

For instance, if we'd made an typographical error in our
[prior example](#example-2) when attempting to reference the `dname` field,
the misspelled column would appear as embedded missing errors:

```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrRead),count() by cname,dnmae | sort -r count' testscores.zson
```

#### Output:
```mdtest-output head
cname           dnmae avg                count
Los Angeles     -     450.83037974683543 469
San Diego       -     496.74789915966386 168
San Bernardino  -     465.11764705882354 117
Riverside       -     463.8170731707317  110
Orange          -     510.91011235955057 107
...
```
# Note: Undefined Order

The order of results from a grouped aggregation is undefined. If you want to
ensure a specific order, a [`sort` operator](../zq/operators/sort.md)
should be used downstream of the aggregation in the Zed pipeline.
It is for this reason that our examples above all included an explicit
`| sort` at the end of each pipeline.

XXX FROM OPERATORS DOCS


## `head`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Return only the first N records.                                      |
| **Syntax**                | `head [N]`                                                            |
| **Required<br>arguments** | None. If no arguments are specified, only the first record is returned.|
| **Optional<br>arguments** | `[N]`<br>An integer specifying the number of records to return. If not specified, defaults to `1`. |

#### Example #1:

To see the first school record:

```mdtest-command dir=testdata/edu
zq -Z 'head' schools.zson
```

#### Output:
```mdtest-output
{
    School: "'3R' Middle",
    District: "Nevada County Office of Education",
    City: "Nevada City",
    County: "Nevada",
    Zip: "95959",
    Latitude: null (float64),
    Longitude: null (float64),
    Magnet: null (bool),
    OpenDate: 1995-10-30T00:00:00Z,
    ClosedDate: 1996-06-28T00:00:00Z,
    Phone: null (string),
    StatusType: "Merged",
    Website: null (string)
}
```

#### Example #2:

To see the first five school records in Los Angeles county:

```mdtest-command dir=testdata/edu
zq -z 'County=="Los Angeles" | head 5' schools.zson
```

#### Output:
```mdtest-output
{School:"ABC Adult",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90703-2801",Latitude:33.878924,Longitude:-118.07128,Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(562) 229-7960",StatusType:"Active",Website:"www.abcadultschool.com"}
{School:"ABC Charter Middle",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90017",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:2008-09-03T00:00:00Z,ClosedDate:2009-06-10T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.abcsf.us"}
{School:"ABC Evening High School",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90701",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1994-11-23T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"ABC Secondary (Alternative)",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90703-2301",Latitude:33.881547,Longitude:-118.04635,Magnet:false,OpenDate:1991-09-05T00:00:00Z,ClosedDate:null(time),Phone:"(562) 229-7768",StatusType:"Active",Website:null(string)}
{School:"APEX Academy",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90028-8526",Latitude:34.052234,Longitude:-118.24368,Magnet:false,OpenDate:2008-09-03T00:00:00Z,ClosedDate:null(time),Phone:"(323) 817-6550",StatusType:"Active",Website:null(string)}
```

## `tail`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Return only the last N records.                                       |
| **Syntax**                | `tail [N]`                                                            |
| **Required<br>arguments** | None. If no arguments are specified, only the last record is returned.|
| **Optional<br>arguments** | `[N]`<br>An integer specifying the number of records to return. If not specified, defaults to `1`. |

#### Example #1:

To see the last school record:

```mdtest-command dir=testdata/edu
zq -Z 'tail' schools.zson
```

#### Output:
```mdtest-output
{
    School: null (string),
    District: "Wheatland Union High",
    City: "Wheatland",
    County: "Yuba",
    Zip: "95692-9798",
    Latitude: 38.998968,
    Longitude: -121.45497,
    Magnet: null (bool),
    OpenDate: null (time),
    ClosedDate: null (time),
    Phone: "(530) 633-3100",
    StatusType: "Active",
    Website: "www.wheatlandhigh.org"
}
```

#### Example #2:

To see the last five school records in Los Angeles county:

```mdtest-command dir=testdata/edu
zq -z 'County=="Los Angeles" | tail 5' schools.zson
```

#### Output:
```mdtest-output
{School:null(string),District:"Wiseburn Unified",City:"Hawthorne",County:"Los Angeles",Zip:"90250-6462",Latitude:33.920462,Longitude:-118.37839,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(310) 643-3025",StatusType:"Active",Website:"www.wiseburn.k12.ca.us"}
{School:null(string),District:"SBE - Anahuacalmecac International University Preparatory of North America",City:"Los Angeles",County:"Los Angeles",Zip:"90032-1942",Latitude:34.085085,Longitude:-118.18154,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 352-3148",StatusType:"Active",Website:"www.dignidad.org"}
{School:null(string),District:"SBE - Academia Avance Charter",City:"Highland Park",County:"Los Angeles",Zip:"90042-4005",Latitude:34.107313,Longitude:-118.19811,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 230-7270",StatusType:"Active",Website:"www.academiaavance.com"}
{School:null(string),District:"SBE - Prepa Tec Los Angeles High",City:"Huntington Park",County:"Los Angeles",Zip:"90255-4138",Latitude:33.983752,Longitude:-118.22344,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 800-2741",StatusType:"Active",Website:"www.prepatechighschool.org"}
{School:null(string),District:"California Advancing Pathways for Students in Los Angeles County ROC/P",City:"Bellflower",County:"Los Angeles",Zip:"90706",Latitude:33.882509,Longitude:-118.13442,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(562) 866-9011",StatusType:"Active",Website:"www.CalAPS.org"}
```



## `cut`

|                           |                                                   |
| ------------------------- | ------------------------------------------------- |
| **Description**           | Return the data only from the specified named fields, where available. |
| **Syntax**                | `cut <field-list>`                                |
| **Required<br>arguments** | `<field-list>`<br>One or more comma-separated field names or assignments.  |

#### Example #1:

To return only the name and opening date from our school records:

```mdtest-command dir=testdata/edu
zq -Z 'cut School,OpenDate' schools.zson
```

#### Output:
```mdtest-output head
{
    School: "'3R' Middle",
    OpenDate: 1995-10-30T00:00:00Z
}
{
    School: "100 Black Men of the Bay Area Community",
    OpenDate: 2012-08-06T00:00:00Z
}
...
```

#### Example #2:

As long as some of the named fields are present, they are returned while absent
fields are `error("missing")`. For instance, the following
query is run against all three of our data sources and returns values from our
school data that includes fields for both `School` and `Website`, values from
our web address data that have the `Website` and `addr` fields, and the
missing value from the test score data since it has none of these fields.

```mdtest-command dir=testdata/edu
zq -z 'yosemiteuhsd | cut School,Website,addr' *.zson
```

#### Output:
```mdtest-output
{School:null(string),Website:"www.yosemiteuhsd.com",addr:error("missing")}
{School:error("missing"),Website:"www.yosemiteuhsd.com",addr:104.253.209.210}
```

#### Example #3:

To return only the `sname` and `dname` fields of the test scores while also
renaming the fields:

```mdtest-command dir=testdata/edu
zq -z 'cut School:=sname,District:=dname' testscores.zson
```

#### Output:
```mdtest-output head
{School:"21st Century Learning Institute",District:"Beaumont Unified"}
{School:"ABC Secondary (Alternative)",District:"ABC Unified"}
...
```


## `drop`

|                           |                                                             |
| ------------------------- | ----------------------------------------------------------- |
| **Description**           | Return the data from all but the specified named fields.    |
| **Syntax**                | `drop <field-list>`                                         |
| **Required<br>arguments** | `<field-list>`<br>One or more comma-separated field names or assignments.  |

#### Example #1:

To return all the fields _other than_ the score values in our test score data:

```mdtest-command dir=testdata/edu
zq -z 'drop AvgScrMath,AvgScrRead,AvgScrWrite' testscores.zson
```

#### Output:
```mdtest-output head
{cname:"Riverside",dname:"Beaumont Unified",sname:"21st Century Learning Institute"}
{cname:"Los Angeles",dname:"ABC Unified",sname:"ABC Secondary (Alternative)"}
...
```


## `filter`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Apply a search to potentially trim data from the pipeline.            |
| **Syntax**                | `filter <search>`                                                     |
| **Required<br>arguments** | `<search>`<br>Any valid Zed [search syntax](../zq/language.md#search-expressions) |
| **Optional<br>arguments** | None                                                                  |

> **Note:** As searches may appear anywhere in a Zed pipeline, it is not
> strictly necessary to enter the explicit `filter` operator name before your
> search. However, you may find it useful to include it to help express the
> intent of your query.

#### Example #1:

To further trim the data returned in our [`cut`](#cut) example:

```mdtest-command dir=testdata/edu
zq -Z 'cut School,OpenDate | where School=="Breeze Hill Elementary"' schools.zson
```

#### Output:
```mdtest-output
{
    School: "Breeze Hill Elementary",
    OpenDate: 1992-07-06T00:00:00Z
}
```

#### Example #2:

An alternative syntax for our [`and` example](../zq/language.md#search-expressions):

```mdtest-command dir=testdata/edu
zq -z 'where StatusType=="Pending" academy' schools.zson
```

#### Output:
```mdtest-output
{School:"Equitas Academy 4",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90015-2412",Latitude:34.044837,Longitude:-118.27844,Magnet:false,OpenDate:2017-09-01T00:00:00Z,ClosedDate:null(time),Phone:"(213) 201-0440",StatusType:"Pending",Website:"http://equitasacademy.org"}
{School:"Pinnacle Academy Charter - Independent Study",District:"South Monterey County Joint Union High",City:"King City",County:"Monterey",Zip:"93930-3311",Latitude:36.208934,Longitude:-121.13286,Magnet:false,OpenDate:2016-08-08T00:00:00Z,ClosedDate:null(time),Phone:"(831) 385-4661",StatusType:"Pending",Website:"www.smcjuhsd.org"}
{School:"Rocketship Futuro Academy",District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:false,OpenDate:2016-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
{School:"Sherman Thomas STEM Academy",District:"Madera Unified",City:"Madera",County:"Madera",Zip:"93638",Latitude:36.982843,Longitude:-120.06665,Magnet:false,OpenDate:2017-08-09T00:00:00Z,ClosedDate:null(time),Phone:"(559) 674-1192",StatusType:"Pending",Website:"www.stcs.k12.ca.us"}
{School:null(string),District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
```


## `fuse`

|                           |                                                   |
| ------------------------- | ------------------------------------------------- |
| **Description**           | Transforms input records into output records that unify the field and type information across all records in the query result. |
| **Syntax**                | `fuse`                                            |
| **Required<br>arguments** | None                                              |
| **Optional<br>arguments** | None                                              |
| **Limitations**           | Because `fuse` must make a first pass through the data to assemble a unified schema, results from queries that use `fuse` will not begin streaming back immediately. |

#### Example:

Let's say you'd started with table-formatted output of all records in our data
that reference the town of Geyserville.

```mdtest-command dir=testdata/edu
zq -f table 'Geyserville' *.zson
```

#### Output:
```mdtest-output
School                            District            City        County Zip        Latitude  Longitude  Magnet OpenDate             ClosedDate           Phone          StatusType Website
Buena Vista High                  Geyserville Unified Geyserville Sonoma 95441-9670 38.722005 -122.89123 F      1980-07-01T00:00:00Z                      (707) 857-3592 Active     -
Geyserville Community Day         Geyserville Unified Geyserville Sonoma 95441      38.722005 -122.89123 -      2004-09-01T00:00:00Z 2010-06-30T00:00:00Z -              Closed     -
Geyserville Educational Park High Geyserville Unified Geyserville Sonoma 95441      38.722005 -122.89123 -      1980-07-01T00:00:00Z 2014-06-30T00:00:00Z -              Closed     -
Geyserville Elementary            Geyserville Unified Geyserville Sonoma 95441-0108 38.705895 -122.90296 F      1980-07-01T00:00:00Z                      (707) 857-3410 Active     www.gusd.com
Geyserville Middle                Geyserville Unified Geyserville Sonoma 95441      38.722005 -122.89123 -      1980-07-01T00:00:00Z 2014-06-30T00:00:00Z -              Closed     -
Geyserville New Tech Academy      Geyserville Unified Geyserville Sonoma 95441-9670 38.72015  -122.88534 F      2014-07-01T00:00:00Z                      (707) 857-3592 Active     www.gusd.com
-                                 Geyserville Unified Geyserville Sonoma 95441-9670 38.722005 -122.89123 -                                                (707) 857-3592 Active     www.gusd.com
AvgScrMath AvgScrRead AvgScrWrite cname  dname               sname
-          -          -           Sonoma Geyserville Unified Geyserville New Tech Academy
-          -          -           Sonoma Geyserville Unified -
```

School records were output first, so the preceding header row describes the
names of those fields. Later on, two test score records were also output, so
a header row describing its fields was also printed. This presentation
accurately conveys the heterogeneous nature of the data, but changing schemas
mid-stream is not allowed in formats such as CSV or other downstream tooling
such as SQL. Indeed, `zq` halts its output in this case.

```mdtest-command dir=testdata/edu fails
zq -f csv 'Geyserville' *.zson
```

#### Output:
```mdtest-output
School,District,City,County,Zip,Latitude,Longitude,Magnet,OpenDate,ClosedDate,Phone,StatusType,Website
Buena Vista High,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.722005,-122.89123,false,1980-07-01T00:00:00Z,,(707) 857-3592,Active,
Geyserville Community Day,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,2004-09-01T00:00:00Z,2010-06-30T00:00:00Z,,Closed,
Geyserville Educational Park High,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,1980-07-01T00:00:00Z,2014-06-30T00:00:00Z,,Closed,
Geyserville Elementary,Geyserville Unified,Geyserville,Sonoma,95441-0108,38.705895,-122.90296,false,1980-07-01T00:00:00Z,,(707) 857-3410,Active,www.gusd.com
Geyserville Middle,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,1980-07-01T00:00:00Z,2014-06-30T00:00:00Z,,Closed,
Geyserville New Tech Academy,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.72015,-122.88534,false,2014-07-01T00:00:00Z,,(707) 857-3592,Active,www.gusd.com
,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.722005,-122.89123,,,,(707) 857-3592,Active,www.gusd.com
CSV output requires uniform records but multiple types encountered (consider 'fuse')
```

By using `fuse`, the unified schema of field names and types across all records
is assembled in a first pass through the data stream, which enables the
presentation of the results under a single, wider header row with no further
interruptions between the subsequent data rows.

```mdtest-command dir=testdata/edu
zq -f csv 'Geyserville | fuse' *.zson
```

#### Output:
```mdtest-output
School,District,City,County,Zip,Latitude,Longitude,Magnet,OpenDate,ClosedDate,Phone,StatusType,Website,AvgScrMath,AvgScrRead,AvgScrWrite,cname,dname,sname
Buena Vista High,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.722005,-122.89123,false,1980-07-01T00:00:00Z,,(707) 857-3592,Active,,,,,,,
Geyserville Community Day,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,2004-09-01T00:00:00Z,2010-06-30T00:00:00Z,,Closed,,,,,,,
Geyserville Educational Park High,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,1980-07-01T00:00:00Z,2014-06-30T00:00:00Z,,Closed,,,,,,,
Geyserville Elementary,Geyserville Unified,Geyserville,Sonoma,95441-0108,38.705895,-122.90296,false,1980-07-01T00:00:00Z,,(707) 857-3410,Active,www.gusd.com,,,,,,
Geyserville Middle,Geyserville Unified,Geyserville,Sonoma,95441,38.722005,-122.89123,,1980-07-01T00:00:00Z,2014-06-30T00:00:00Z,,Closed,,,,,,,
Geyserville New Tech Academy,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.72015,-122.88534,false,2014-07-01T00:00:00Z,,(707) 857-3592,Active,www.gusd.com,,,,,,
,Geyserville Unified,Geyserville,Sonoma,95441-9670,38.722005,-122.89123,,,,(707) 857-3592,Active,www.gusd.com,,,,,,
,,,,,,,,,,,,,,,,Sonoma,Geyserville Unified,Geyserville New Tech Academy
,,,,,,,,,,,,,,,,Sonoma,Geyserville Unified,
```

Other output formats invoked via `zq -f` that benefit greatly from the use of
`fuse` include `table` and `zeek`.


## `join`

|                           |                                               |
| ------------------------- | --------------------------------------------- |
| **Description**           | Return records derived from two inputs when particular values match between them.<br><br>The inputs must be sorted in the same order by their respective join keys. If an input source is already known to be sorted appropriately (either in an input file/object/stream, or if the data is pulled from a [Zed Lake](../zed/README.md) that's ordered by this key) an explicit upstream [`sort`](#sort) is not required. ||
| **Syntax**                | `[anti\|inner\|left\|right] join on <left-key>=<right-key> [field-list]`          |
| **Required<br>arguments** | `<left-key>`<br>A field in the left-hand input whose contents will be checked for equality against the `<right-key>`<br><br>`<right-key>`<br>A field in the right-hand input whose contents will be checked for equality against the `<left-key>` |
| **Optional<br>arguments** | `[anti\|inner\|left\|right]`<br>The type of join that should be performed.<br>• `anti` - Return all records from the left-hand input for which `<left-key>` exists but that match no records from the right-hand input<br>• `inner` - Return only records that have matching key values in both inputs (default)<br>• `left` - Return all records from the left-hand input, and matched records from the right-hand input<br>• `right` - Return all records from the right-hand input, and matched records from the left-hand input<br><br>`[field-list]`<br>One or more comma-separated field names or assignments. The values in the field(s) specified will be copied from the _opposite_ input (right-hand side for an `anti`, `inner`, or `left` join, left-hand side for a `right` join) into the joined results. If no field list is provided, no fields from the opposite input will appear in the joined results (see [zed/2815](https://github.com/brimdata/zed/issues/2815) regarding expected enhancements in this area). |
| **Limitations**           | • The order of the left/right key names in the equality test must follow the left/right order of the input sources that precede the `join` ([zed/2228](https://github.com/brimdata/zed/issues/2228))<br>• Only a simple equality test (not an arbitrary expression) is currently possible ([zed/2766](https://github.com/brimdata/zed/issues/2766)) |

The first input data source for our usage examples is `fruit.ndjson`, which describes
the characteristics of some fresh produce.

```mdtest-input fruit.ndjson
{"name":"apple","color":"red","flavor":"tart"}
{"name":"banana","color":"yellow","flavor":"sweet"}
{"name":"avocado","color":"green","flavor":"savory"}
{"name":"strawberry","color":"red","flavor":"sweet"}
{"name":"dates","color":"brown","flavor":"sweet","note":"in season"}
{"name":"figs","color":"brown","flavor":"plain"}
```

The other input data source is `people.ndjson`, which describes the traits
and preferences of some potential eaters of fruit.

```mdtest-input people.ndjson
{"name":"morgan","age":61,"likes":"tart"}
{"name":"quinn","age":14,"likes":"sweet","note":"many kids enjoy sweets"}
{"name":"jessie","age":30,"likes":"plain"}
{"name":"chris","age":47,"likes":"tart"}
```

#### Example #1 - Inner join

We'll start by outputting only the fruits liked by at least one person.
The name of the matching person is copied into a field of a different name in
the joined results.

Because we're performing an inner join (the default), the inclusion of the
explicit `inner` is not strictly necessary, but may be included to help make
the Zed self-documenting.

Notice how each input is specified separately within the parentheses-wrapped
`from()` block before the `join` appears in our Zed pipeline.

The Zed script `inner-join.zed`:
```mdtest-input inner-join.zed
from (
  file fruit.ndjson => sort flavor
  file people.ndjson => sort likes
) | inner join on flavor=likes eater:=name
```

Executing the Zed script:

```mdtest-command
zq -z -I inner-join.zed
```

#### Output:
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
```

#### Example #2 - Left join

By performing a left join that targets the same key fields, now all of our
fruits will be shown in the results even if no one likes them (e.g., `avocado`).

As another variation, we'll also copy over the age of the matching person. By
referencing only the field name rather than using `:=` for assignment, the
original field name `age` is maintained in the results.

The Zed script `left-join.zed`:

```mdtest-input left-join.zed
from (
  file fruit.ndjson => sort flavor
  file people.ndjson => sort likes
) | left join on flavor=likes eater:=name,age
```

Executing the Zed script:

```mdtest-command
zq -z -I left-join.zed
```

#### Output:
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie",age:30}
{name:"avocado",color:"green",flavor:"savory"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn",age:14}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn",age:14}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn",age:14}
{name:"apple",color:"red",flavor:"tart",eater:"morgan",age:61}
{name:"apple",color:"red",flavor:"tart",eater:"chris",age:47}
```

#### Example #3 - Right join

Next we'll change the join type from `left` to `right`. Notice that this causes
the `note` field from the right-hand input to appear in the joined results.

The Zed script `right-join.zed`:

```mdtest-input right-join.zed
from (
  file fruit.ndjson => sort flavor
  file people.ndjson => sort likes
) | right join on flavor=likes fruit:=name
```

Executing the Zed script:

```mdtest-command
zq -z -I right-join.zed
```

#### Output:
```mdtest-output
{name:"jessie",age:30,likes:"plain",fruit:"figs"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"banana"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"strawberry"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"dates"}
{name:"morgan",age:61,likes:"tart",fruit:"apple"}
{name:"chris",age:47,likes:"tart",fruit:"apple"}
```

#### Example #4 - Inputs from Pools

As our prior examples all used `zq`, we used `file` in our `from()` block to
pull our respective inputs from named file sources. However, if the inputs are
stored in Pools in a Zed lake, the Pool names would instead be specified in the
`from()` block.

Here we'll load our input data to Pools in a temporary Zed Lake, then execute
our inner join using `zed query`.

Notice that because we happened to use `-orderby` to sort our Pools by the same
keys that we reference in our `join`, we did not need to use any explicit
upstream `sort`.

The Zed script `inner-join-pools.zed`:

```mdtest-input inner-join-pools.zed
from (
  pool fruit
  pool people
) | inner join on flavor=likes eater:=name
```

Populating the Pools, then executing the Zed script:

```mdtest-command
mkdir lake
export ZED_LAKE=lake
zed init -q
zed create -q -orderby flavor:asc fruit
zed create -q -orderby likes:asc people
zed load -q -use fruit@main fruit.ndjson
zed load -q -use people@main people.ndjson
zed query -z -I inner-join-pools.zed
```

#### Output:
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
```

#### Example #5 - Streamed input

In addition to the named files and Pools like we've used in the prior examples,
Zed is also intended to work on streams of data. Here we'll combine our file
sources into a stream that we'll pipe into `zq` via stdin. Because join requires
two separate inputs, here we'll use the `has()` function to identify the
records in the stream that will be treated as the left and right sides.

The Zed script `inner-join-streamed.zed`:

```mdtest-input inner-join-streamed.zed
switch (
  case has(color) => sort flavor
  case has(age) => sort likes
) | inner join on flavor=likes eater:=name
```

Executing the Zed script:
```mdtest-command
cat fruit.ndjson people.ndjson | zq -z -I inner-join-streamed.zed -
```

#### Output:
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
```

#### Example #6 - Multi-value join

The equality test in a Zed join accepts only one named key from each input.
However, joins on multiple matching values can still be performed by making the
values available in comparable complex types, such as embedded records.

To illustrate this, we'll introduce some new input data `inventory.ndjson`
that represents a vendor's available quantity of fruit for sale. As the colors
indicate, they separately offer both ripe and unripe fruit.

```mdtest-input inventory.ndjson
{"name":"banana","color":"yellow","quantity":1000}
{"name":"banana","color":"green","quantity":5000}
{"name":"strawberry","color":"red","quantity":3000}
{"name":"strawberry","color":"white","quantity":6000}
```

Let's assume we're interested in seeing the available quantities of only the
immediately-edible fruit/color combinations shown in our `fruit.ndjson`
records. In the Zed script `multi-value-join.zed`, we create the keys as
embedded records inside each input record, using the same field names and data
types in each. We'll leave the created `fruitkey` records intact to show what
they look like, but since it represents redundant data, in practice we'd
typically [`drop`](#drop) it after the `join` in our Zed pipeline.

```mdtest-input multi-value-join.zed
from (
  file fruit.ndjson => put fruitkey:={name:string(name),color:string(color)} | sort fruitkey
  file inventory.ndjson => put invkey:={name:string(name),color:string(color)} | sort invkey
) | inner join on fruitkey=invkey quantity
```

Executing the Zed script:
```mdtest-command
zq -z -I multi-value-join.zed
```

#### Output:
```mdtest-output
{name:"banana",color:"yellow",flavor:"sweet",fruitkey:{name:"banana",color:"yellow"},quantity:1000}
{name:"strawberry",color:"red",flavor:"sweet",fruitkey:{name:"strawberry",color:"red"},quantity:3000}
```

#### Example #7 - Embedding the entire opposite record

As previously noted, until [zed/2815](https://github.com/brimdata/zed/issues/2815)
is addressed, explicit entries must be provided in the `[field-list]` in order
to copy values from the opposite input into the joined results. This can be
cumbersome if your goal is to copy over many fields or you don't know the
names of all desired fields.

One way to work around this limitation is to specify `this` in the field list
to copy the contents of the _entire_ opposite record into an embedded record
in the result.

The Zed script `embed-opposite.zed`:

```mdtest-input embed-opposite.zed
from (
  file fruit.ndjson => sort flavor
  file people.ndjson => sort likes
) | inner join on flavor=likes eaterinfo:=this
```

Executing the Zed script:

```mdtest-command
zq -z -I embed-opposite.zed
```

#### Output:
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eaterinfo:{name:"jessie",age:30,likes:"plain"}}
{name:"banana",color:"yellow",flavor:"sweet",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"strawberry",color:"red",flavor:"sweet",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"apple",color:"red",flavor:"tart",eaterinfo:{name:"morgan",age:61,likes:"tart"}}
{name:"apple",color:"red",flavor:"tart",eaterinfo:{name:"chris",age:47,likes:"tart"}}
```

## `put`

|                           |                                                 |
| ------------------------- | ----------------------------------------------- |
| **Description**           | Add/update fields based on the results of an expression.<br><br>If evaluation of any expression fails, a missing error
is emitted for the respective field.<br><br>As this operation is very common, the `put` keyword is optional. |
| **Syntax**                | `[put] <field> := <expression> [, (<field> := <expression>)...]` |
| **Required arguments**    | One or more of:<br><br>`<field> := <expression>`<br>Any valid Zed [expression](../zq/language.md#expressions), preceded by the assignment operator `:=` and the name of a field in which to store the result. |
| **Optional arguments**    | None |
| **Limitations**           | If multiple fields are written in a single `put`, all the new field values are computed first and then they are all written simultaneously.  As a result, a computed value cannot be referenced in another expression.  If you need to re-use a computed result, this can be done by chaining multiple `put` operators.  For example, this will not work:<br>`put N:=len(somelist), isbig:=N>10`<br>But it could be written instead as:<br>`put N:=len(somelist) \| put isbig:=N>10` |

#### Example #1:

Add a field to our test score records to hold the computed average of the math,
reading, and writing scores for each school that reported them.

```mdtest-command dir=testdata/edu
zq -Z 'AvgScrMath!=null | put AvgAll:=(AvgScrMath+AvgScrRead+AvgScrWrite)/3.0' testscores.zson
```

#### Output:
```mdtest-output head
{
    AvgScrMath: 371 (uint16),
    AvgScrRead: 376 (uint16),
    AvgScrWrite: 368 (uint16),
    cname: "Los Angeles",
    dname: "Los Angeles Unified",
    sname: "APEX Academy",
    AvgAll: 371.6666666666667
}
...
```

#### Example #2:

As noted above the `put` keyword is entirely optional. Here we omit
it and create a new field to hold the lowercase representation of
the school `District` field.

```mdtest-command dir=testdata/edu
zq -Z 'cut District | lower_district:=to_lower(District)' schools.zson
```

#### Output:
```mdtest-output head
{
    District: "Nevada County Office of Education",
    lower_district: "nevada county office of education"
}
...
```

## `rename`

|                           |                                                 |
| ------------------------- | ----------------------------------------------- |
| **Description**           | Rename fields in a record.                      |
| **Syntax**                | `rename <newname> := <oldname> [, <newname> := <oldname> ...]`     |
| **Required arguments**    | One or more field assignment expressions. Renames are applied left to right; each rename observes the effect of all renames that preceded it. |
| **Optional arguments**    | None |
| **Limitations**           | A field can only be renamed within its own record. |

#### Example #1:

To rename some fields in our test score data to match the field names from
our school data:

```mdtest-command dir=testdata/edu
zq -Z 'rename School:=sname,District:=dname,City:=cname' testscores.zson
```

#### Output:
```mdtest-output head
{
    AvgScrMath: null (uint16),
    AvgScrRead: null (uint16),
    AvgScrWrite: null (uint16),
    City: "Riverside",
    District: "Beaumont Unified",
    School: "21st Century Learning Institute"
}
...
```

#### Example #2:

As mentioned above, a field can only be renamed within its own record. In
other words, a field cannot move between nested levels when being renamed.

For example, consider this sample input data `nested.zson`:

```mdtest-input nested.zson
{
    outer: {
        inner: "MyValue"
    }
}
```

The field `inner` can be renamed within that nested record.

```mdtest-command
zq -Z 'rename outer.renamed:=outer.inner' nested.zson
```

#### Output:
```mdtest-output
{
    outer: {
        renamed: "MyValue"
    }
}
```

However, an attempt to rename it to a top-level field will fail.

```mdtest-command fails
zq -Z 'rename toplevel:=outer.inner' nested.zson
```

#### Output:
```mdtest-output
cannot rename outer.inner to toplevel
```

This could instead be achieved by combining [`put`](#put) and [`drop`](#drop).

```mdtest-command
zq -Z 'put toplevel:=outer.inner | drop outer.inner' nested.zson
```

#### Output:
```mdtest-output
{
    toplevel: "MyValue"
}
```

## `sort`

|                           |                                                                           |
| ------------------------- | ------------------------------------------------------------------------- |
| **Description**           | Sort records based on the order of values in the specified named field(s).|
| **Syntax**                | `sort [-r] [-nulls first\|last] [field-list]`                             |
| **Required<br>arguments** | None                                                                      |
| **Optional<br>arguments** | `[-r]`<br>If specified, results will be sorted in reverse order.<br><br>`[-nulls first\|last]`<br>Specifies where null values should be placed in the output.<br><br>`[field-list]`<br>One or more comma-separated field names by which to sort. Results will be sorted based on the values of the first field named in the list, then based on values in the second field named in the list, and so on.<br><br>If no field list is provided, `sort` will automatically pick a field by which to sort. It does so by examining the first input record and finding the first field in left-to-right order that is of a Zed integer [data type](../zq/language.md#data-types) (`int8`, `uint8`, `int16`, `uint16`, `int32`, `uint32`, `int64`, `uint64`) or, if no integer field is found, the first field that is of a floating point data type (`float16`, `float32`, `float64`). If no such numeric field is found, `sort` finds the first field in left-to-right order that is _not_ of the `time` data type. Note that there are some cases (such as the output of a [grouped aggregation](../zq/operators/summarize.md) performed on heterogeneous data) where the first input record to `sort` may vary even when the same query is executed repeatedly against the same data. If you require a query to show deterministic output on repeated execution, an explicit field list must be provided. |

#### Example #1:

To sort our test score records by average reading score:

```mdtest-command dir=testdata/edu
zq -z 'sort AvgScrRead' testscores.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:352(uint16),AvgScrRead:308(uint16),AvgScrWrite:327(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"Oakland International High"}
{AvgScrMath:289(uint16),AvgScrRead:314(uint16),AvgScrWrite:312(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"Gompers (Samuel) Continuation"}
{AvgScrMath:450(uint16),AvgScrRead:321(uint16),AvgScrWrite:318(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"S.F. International High"}
{AvgScrMath:314(uint16),AvgScrRead:324(uint16),AvgScrWrite:321(uint16),cname:"Los Angeles",dname:"Norwalk-La Mirada Unified",sname:"El Camino High (Continuation)"}
{AvgScrMath:307(uint16),AvgScrRead:324(uint16),AvgScrWrite:328(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"North Campus Continuation"}
...
```

#### Example #2:

Now we'll sort the test score records first by average reading score and then
by average math score. Note how this changed the order of the bottom two
records in the result.

```mdtest-command dir=testdata/edu
zq -z 'sort AvgScrRead,AvgScrMath' testscores.zson
```

#### Output:
```mdtest-output head
{AvgScrMath:352(uint16),AvgScrRead:308(uint16),AvgScrWrite:327(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"Oakland International High"}
{AvgScrMath:289(uint16),AvgScrRead:314(uint16),AvgScrWrite:312(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"Gompers (Samuel) Continuation"}
{AvgScrMath:450(uint16),AvgScrRead:321(uint16),AvgScrWrite:318(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"S.F. International High"}
{AvgScrMath:307(uint16),AvgScrRead:324(uint16),AvgScrWrite:328(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"North Campus Continuation"}
{AvgScrMath:314(uint16),AvgScrRead:324(uint16),AvgScrWrite:321(uint16),cname:"Los Angeles",dname:"Norwalk-La Mirada Unified",sname:"El Camino High (Continuation)"}
...
```

#### Example #3:

Here we'll find the counties with the most schools by using the
[`count()`](../zq/aggregates/count.md) aggregate function and piping its
output to a `sort` in reverse order. Note that even though we didn't list a
field name as an explicit argument, the `sort` operator did what we wanted
because it found a field of the `uint64` [data type](../zq/language.md#data-types).

```mdtest-command dir=testdata/edu
zq -z 'count() by County | sort -r' schools.zson
```

#### Output:
```mdtest-output head
{County:"Los Angeles",count:3636(uint64)}
{County:"San Diego",count:1139(uint64)}
{County:"Orange",count:886(uint64)}
...
```

#### Example #4:

Next we'll count the number of unique websites mentioned in our school
records. Since we know some of the records don't include a website, we'll
deliberately put the null values at the front of the list so we can see how
many there are.

```mdtest-command dir=testdata/edu
zq -z 'count() by Website | sort -nulls first Website' schools.zson
```

#### Output:
```mdtest-output head
{Website:null(string),count:10722(uint64)}
{Website:"acornstooakscharter.org",count:1(uint64)}
{Website:"atlascharter.org",count:1(uint64)}
{Website:"bizweb.lightspeed.net/~leagles",count:1(uint64)}
...
```




## `uniq`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Remove adjacent duplicate records from the output, leaving only unique results.<br><br>Note that due to the large number of fields in typical records, and many fields whose values change often in subtle ways between records (e.g., timestamps), this operator will most often apply to the trimmed output from [`cut`](#cut). Furthermore, since duplicate field values may not often be adjacent to one another, upstream use of [`sort`](#sort) may also often be appropriate.
| **Syntax**                | `uniq [-c]`                                                           |
| **Required<br>arguments** | None                                                                  |
| **Optional<br>arguments** | `[-c]`<br>For each unique value shown, include a numeric count of how many times it appeared. |

#### Example:

Let's say you'd been looking at the contents of just the `District` and
`County` fields in the order they appear in the school data.

```mdtest-command dir=testdata/edu
zq -z 'cut District,County' schools.zson
```

#### Output:
```mdtest-output head
{District:"Nevada County Office of Education",County:"Nevada"}
{District:"Oakland Unified",County:"Alameda"}
{District:"Victor Elementary",County:"San Bernardino"}
{District:"Novato Unified",County:"Marin"}
{District:"Beaumont Unified",County:"Riverside"}
{District:"Nevada County Office of Education",County:"Nevada"}
{District:"Nevada County Office of Education",County:"Nevada"}
{District:"San Bernardino City Unified",County:"San Bernardino"}
{District:"San Bernardino City Unified",County:"San Bernardino"}
{District:"Ojai Unified",County:"Ventura"}
...
```

To eliminate the adjacent lines that share the same field/value pairs:

```mdtest-command dir=testdata/edu
zq -z 'cut District,County | uniq' schools.zson
```

#### Output:
```mdtest-output head
{District:"Nevada County Office of Education",County:"Nevada"}
{District:"Oakland Unified",County:"Alameda"}
{District:"Victor Elementary",County:"San Bernardino"}
{District:"Novato Unified",County:"Marin"}
{District:"Beaumont Unified",County:"Riverside"}
{District:"Nevada County Office of Education",County:"Nevada"}
{District:"San Bernardino City Unified",County:"San Bernardino"}
{District:"Ojai Unified",County:"Ventura"}
...
```




## `yield`

|                           |                                                 |
| ------------------------- | ----------------------------------------------- |
| **Description**           | For each input value, produces one or more values downstream.
| **Syntax**                | `yield <expression> [, (<expression>)...]` |
| **Required arguments**    | One or more of `<expression>`, which may be any valid Zed [expression](../zq/language.md#expressions).
| **Optional arguments**    | None |

#### Example #1:

This example produce two simpler records for every school record listing
the average math score with the school name and the county name.

```mdtest-command dir=testdata/edu
zq -Z 'AvgScrMath!=null | yield {school:sname,avg:AvgScrMath}, {county:cname,zvg:AvgScrMath}' testscores.zson
```

>

#### Output:
```mdtest-output head 4
{
    school: "APEX Academy",
    avg: 371 (uint16)
}
{
    county: "Los Angeles",
    zvg: 371 (uint16)
}
{
    school: "ARISE High",
    avg: 367 (uint16)
}
{
    county: "Alameda",
    zvg: 367 (uint16)
}
...
```
