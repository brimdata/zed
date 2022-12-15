---
sidebar_position: 4
sidebar_label: Schools Data
---

# Zed and Schools Data

> This document provides a beginner's overview of the Zed language
using the [zq command](../commands/zq.md) and
[real-world data](https://github.com/brimdata/zed/blob/main/testdata/edu/README.md) relating to California schools
and test scores.

## 1. Getting Started

If you want to follow along by running the examples, simply
[install zq](../install.md) and copy the
data files used here into your working directory:
```
curl https://raw.githubusercontent.com/brimdata/zed/main/testdata/edu/schools.zson > schools.zson
curl https://raw.githubusercontent.com/brimdata/zed/main/testdata/edu/testscores.zson > testscores.zson
curl https://raw.githubusercontent.com/brimdata/zed/main/testdata/edu/webaddrs.zson > webaddrs.zson
```
These files are all encoded in the human-readable [ZSON format](../formats/zson.md)
so you can easily have a look at them.  ZSON is not optimized for speed but these
files are small enough that the example queries here will all run fast enough.

## 2. Exploring the Data

It's always a good idea to get a feel for any new data, which is easy to do
with Zed.  Zed's [sample operator](../language/operators/sample.md) is just the ticket ---
`sample` will select one representative value from each "shape" of data present
in the input, e.g.,
```mdtest-command dir=testdata/edu
zq -Z 'sample | sort this' schools.zson testscores.zson webaddrs.zson
```
displays
```mdtest-output
{
    AvgScrMath: null (uint16),
    AvgScrRead: null (uint16),
    AvgScrWrite: null (uint16),
    cname: "Riverside",
    dname: "Beaumont Unified",
    sname: "21st Century Learning Institute"
}
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
{
    Website: "abbott.lynwood.edlioschool.com",
    addr: 151.101.0.80
}
```
>Note that the `-Z` option tells `zq` to "pretty print" the output in
the [ZSON](../formats/zson.md) format.
Furthermore, you will notice these examples often include a `-z` to indicate
line-oriented ZSON, which is the default when `zq` is writing to standard output.
You can omit `-z` when running these commands on the terminal but we include
them here for clarity and because all of the examples are tied to automated testing,
which does not utilize a terminal for standard output.

You can also quickly see a list of the leaf-value data types with this query:
```mdtest-command dir=testdata/edu
zq -Z "sample | over this | by typeof(value) | yield typeof | sort" schools.zson testscores.zson webaddrs.zson
```
which emits
```mdtest-output
<uint16>
<time>
<float64>
<bool>
<string>
<ip>
```
Nothing too tricky here.  After a quick review of the shapes and types,
you will notice they are just three relatively simple tables, which is no surprise
since we obtained the original data from
[SQLite database files](https://github.com/brimdata/zed/blob/main/testdata/edu/README.md).

## 3. Searching

Searching with Zed is easy but powerful because it blends together the
keyword search patterns of Web or email search with the more precise
predicate matching patterns of query languages like SQL.

With this in mind, you can simply start typing keyword search phrases in Zed
and they will usually do the right thing.

### 3.1 Keyword Search

With keyword search, you can just type a keyword that you want to look for, e.g.,
```mdtest-command dir=testdata/edu
zq -z Ygnacio schools.zson
```
which gives the one matching record:
```mdtest-output
{School:"Valencia (Ygnacio) High (Alternative)",District:"Delano Joint Union High",City:"Delano",County:"Kern",Zip:"93215-1526",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:2009-08-01T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Ygnacio Valley Elementary",District:"Mt. Diablo Unified",City:"Concord",County:"Contra Costa",Zip:"94518-2595",Latitude:37.950182,Longitude:-122.0283,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(925) 682-9336",StatusType:"Active",Website:"www.mdusd.org"}
{School:"Ygnacio Valley High",District:"Mt. Diablo Unified",City:"Concord",County:"Contra Costa",Zip:"94518-2899",Latitude:37.936674,Longitude:-122.02325,Magnet:true,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(925) 685-8414",StatusType:"Active",Website:"www.mdusd.org"}
```
As with keyword search, you can simply concantenate keywords to require both
of them to match (i.e., a "logical AND" of the two search predicates), e.g.
we can whittle down the two records above by adding the keyword _Delano_
```mdtest-command dir=testdata/edu
zq -z 'Ygnacio Delano' schools.zson
```
and we get just the one record that matches:
```mdtest-output
{School:"Valencia (Ygnacio) High (Alternative)",District:"Delano Joint Union High",City:"Delano",County:"Kern",Zip:"93215-1526",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:2009-08-01T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
```
Under the covers, a keyword search translates to Zed's [grep function](../language/functions/grep.md),
which lets you search specific fields instead of the entire input value, e.g.,
we can search for the string "bar" in the `City` field and list all the unique
cities that match with a [group-by](#52-grouping):
```mdtest-command dir=testdata/edu
zq -f text 'grep("bar", City) | by City | yield City | sort' schools.zson
```
produces
```mdtest-output
Barstow
Big Bar
Diamond Bar
Long Barn
Santa Barbara
Sawyers Bar
Somes Bar
```
In this example, we use the [yield operator](#8-value-construction) here to pull
the `City` field out of the record result and we used `-f text` to output the
results in "text" format instead of ZSON so the strings are printed
without quotes.  The text format is often useful for piping the output to
other Unix tools that might not expect quotes.

When the keyword you want to search for doesn't fit into the keyword syntax,
i.e., it has spaces or special characters, you should use a
[literal string search](#34-literal-search).

### 3.2 Globs

To find values that may contain arbitrary substrings between or alongside the
desired word(s), one or more
[glob](https://en.wikipedia.org/wiki/Glob_(programming))-style wildcards can be
used.

For example, the following search finds records that contain school names
that have some additional text between `ACE` and `Academy`:
```mdtest-command dir=testdata/edu
zq -z 'ACE*Academy' schools.zson
```
produces
```mdtest-output head
{School:"ACE Empower Academy",District:"Santa Clara County Office of Education",City:"San Jose",County:"Santa Clara",Zip:"95116-3423",Latitude:37.348601,Longitude:-121.8446,Magnet:false,OpenDate:2008-08-26T00:00:00Z,ClosedDate:null(time),Phone:"(408) 729-3920",StatusType:"Active",Website:"www.acecharter.org"}
{School:"ACE Inspire Academy",District:"San Jose Unified",City:"San Jose",County:"Santa Clara",Zip:"95112-6334",Latitude:37.350981,Longitude:-121.87205,Magnet:false,OpenDate:2015-08-03T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-6008",StatusType:"Active",Website:"www.acecharter.org"}
```

Glob wildcards only have effect when used within [keywords](#31-keyword-search)
searches. An asterisk in a [string literal search](#34-literal-search) will match
using the literal asterisk character embedded in the string.

### 3.3 Regular Expressions

For pattern matching beyond [glob wildcards](#32-globs),
regular expressions (regexps) are also
available. To use them, simply place a `/` character before and after the
regexp.

For example, since there are many high schools in our sample data, to find
only records containing strings that _begin_ with the word `High`:
```mdtest-command dir=testdata/edu
zq -z '/^High /' schools.zson
```
produces
```mdtest-output head
{School:"High Desert",District:"Soledad-Agua Dulce Union Eleme",City:"Acton",County:"Los Angeles",Zip:"93510",Latitude:34.490977,Longitude:-118.19646,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1993-06-30T00:00:00Z,Phone:null(string),StatusType:"Merged",Website:null(string)}
{School:"High Desert",District:"Acton-Agua Dulce Unified",City:"Acton",County:"Los Angeles",Zip:"93510-1757",Latitude:34.492578,Longitude:-118.19039,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 269-0310",StatusType:"Active",Website:null(string)}
{School:"High Desert Academy",District:"Eastern Sierra Unified",City:"Benton",County:"Mono",Zip:"93512-0956",Latitude:37.818597,Longitude:-118.47712,Magnet:null(bool),OpenDate:1996-09-03T00:00:00Z,ClosedDate:2012-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.esusd.org"}
{School:"High Desert Academy of Applied Arts and Sciences",District:"Victor Valley Union High",City:"Victorville",County:"San Bernardino",Zip:"92394",Latitude:34.531144,Longitude:-117.31697,Magnet:null(bool),OpenDate:2004-09-07T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.hdaaas.org"}
...
```
Further details for regular expressions are available in
the [Zed language documention](../language/overview.md#711-regular-expressions).

### 3.4 Literal Search

Sometimes you want to search for values that aren't strings, e.g., numbers
or IP addresses.  Zed can search for any
[primitive-type](../formats/zed.md#1-primitive-types) value just typing
that value like a keyword.   In this case, the search looks for
both fields of the value's type for an exact match as well as a substring
match for the value as typed in any strings encountered.

For example, searching across both our school and test score data sources for
the number `596` matches records that contain numeric fields of this precise value
(such as from the test scores) and also records that contain string fields
(such as the ZIP code and phone number fields in the school data), e.g.,
```mdtest-command dir=testdata/edu
zq -z '596' testscores.zson schools.zson
```
finds these records
```mdtest-output head
{AvgScrMath:591(uint16),AvgScrRead:610(uint16),AvgScrWrite:596(uint16),cname:"Los Angeles",dname:"William S. Hart Union High",sname:"Academy of the Canyons"}
{AvgScrMath:614(uint16),AvgScrRead:596(uint16),AvgScrWrite:592(uint16),cname:"Alameda",dname:"Pleasanton Unified",sname:"Amador Valley High"}
{AvgScrMath:620(uint16),AvgScrRead:596(uint16),AvgScrWrite:590(uint16),cname:"Yolo",dname:"Davis Joint Unified",sname:"Davis Senior High"}
{School:"Achieve Charter School of Paradise Inc.",District:"Paradise Unified",City:"Paradise",County:"Butte",Zip:"95969-3913",Latitude:39.760323,Longitude:-121.62078,Magnet:false,OpenDate:2005-09-12T00:00:00Z,ClosedDate:null(time),Phone:"(530) 872-4100",StatusType:"Active",Website:"www.achievecharter.org"}
{School:"Alliance Ouchi-O'Donovan 6-12 Complex",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90043-2622",Latitude:33.993484,Longitude:-118.32246,Magnet:false,OpenDate:2006-09-05T00:00:00Z,ClosedDate:null(time),Phone:"(323) 596-2290",StatusType:"Active",Website:"http://ouchihs.org"}
...
```
Literal search also works for string values.  This is useful when the
string value to search cannot be represented as a keyword due to embedded
spaces or special characters.

Let's say we've noticed that a couple of the school names in our sample data
include the string `Defunct=`. An attempt to enter this as a [keyword](#31-keyword-search)
search causes a parse error, e.g.,
```mdtest-command dir=testdata/edu fails
zq -z 'Defunct=' *.zson
```
produces
```mdtest-output
zq: error parsing Zed at column 8:
Defunct=
   === ^ ===
```
However, wrapping in quotes to performa a string-literal search
gives the desired result:
```mdtest-command dir=testdata/edu
zq -z '"Defunct="' schools.zson
```
produces
```mdtest-output
{School:"Lincoln Elem 'Defunct=",District:"Modesto City Elementary",City:null(string),County:"Stanislaus",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Lovell Elem 'Defunct=",District:"Cutler-Orosi Joint Unified",City:null(string),County:"Tulare",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
```
Quoted strings are particularly handy when you're looking for long, specific
strings that may have several special characters in them. For example, let's
say we're looking for information on the Union Hill Elementary district.
Entered without quotes, we end up matching far more records than we intended
since each space character between words is treated as a [Boolean `and`](#541-and), e.g.,
```mdtest-command dir=testdata/edu
zq -z 'Union Hill Elementary' schools.zson
```
produces
```mdtest-output head
{School:"A. M. Thomas Middle",District:"Lost Hills Union Elementary",City:"Lost Hills",County:"Kern",Zip:"93249-0158",Latitude:35.615269,Longitude:-119.69955,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 797-2626",StatusType:"Active",Website:null(string)}
{School:"Alview Elementary",District:"Alview-Dairyland Union Elementary",City:"Chowchilla",County:"Madera",Zip:"93610-9225",Latitude:37.050632,Longitude:-120.4734,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(559) 665-2275",StatusType:"Active",Website:null(string)}
{School:"Anaverde Hills",District:"Westside Union Elementary",City:"Palmdale",County:"Los Angeles",Zip:"93551-5518",Latitude:34.564651,Longitude:-118.18012,Magnet:false,OpenDate:2005-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(661) 575-9923",StatusType:"Active",Website:null(string)}
{School:"Apple Blossom",District:"Twin Hills Union Elementary",City:"Sebastopol",County:"Sonoma",Zip:"95472-3917",Latitude:38.387396,Longitude:-122.84954,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(707) 823-1041",StatusType:"Active",Website:null(string)}
...
```
However, wrapping the entire search term in quotes allows us to search for the
complete string, including the spaces, e.g.,
```mdtest-command dir=testdata/edu
zq -z '"Union Hill Elementary"' schools.zson
```
produces
```mdtest-output
{School:"Highland Oaks Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1997-09-02T00:00:00Z,ClosedDate:2003-07-02T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Union Hill 3R Community Day",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945",Latitude:39.229055,Longitude:-121.07127,Magnet:null(bool),OpenDate:2003-08-20T00:00:00Z,ClosedDate:2011-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Charter Home",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1995-07-14T00:00:00Z,ClosedDate:2015-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Elementary",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8805",Latitude:39.204457,Longitude:-121.03829,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
{School:"Union Hill Middle",District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"94945-8805",Latitude:39.205006,Longitude:-121.03778,Magnet:false,OpenDate:2013-08-14T00:00:00Z,ClosedDate:null(time),Phone:"(530) 273-8456",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
{School:null(string),District:"Union Hill Elementary",City:"Grass Valley",County:"Nevada",Zip:"95945-8730",Latitude:39.208869,Longitude:-121.03551,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(530) 273-0647",StatusType:"Active",Website:"www.uhsd.k12.ca.us"}
```

### 3.5 Predicate Search

Search terms can also be include Boolean predicates adhering
to Zed's [expression syntax](../language/overview.md#6-expressions).

In particular, a search result can be narrowed down
to include only records that contain a
certain value in a particular named field. For example, the following search
will only match records containing the field called `District` where it is set
to the precise string value `Winton`:
```mdtest-command dir=testdata/edu
zq -z 'District=="Winton"' schools.zson
```
produces
```mdtest-output
{School:"Frank Sparkes Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.382084,Longitude:-120.61847,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6180",StatusType:"Active",Website:null(string)}
{School:"Sybil N. Crookham Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0130",Latitude:37.389501,Longitude:-120.61636,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6182",StatusType:"Active",Website:null(string)}
{School:"Winfield Elementary",District:"Winton",City:"Winton",County:"Merced",Zip:"95388",Latitude:37.389121,Longitude:-120.60442,Magnet:false,OpenDate:2007-08-13T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6891",StatusType:"Active",Website:null(string)}
{School:"Winton Middle",District:"Winton",City:"Winton",County:"Merced",Zip:"95388-1477",Latitude:37.379938,Longitude:-120.62263,Magnet:false,OpenDate:1990-07-20T00:00:00Z,ClosedDate:null(time),Phone:"(209) 357-6189",StatusType:"Active",Website:null(string)}
{School:null(string),District:"Winton",City:"Winton",County:"Merced",Zip:"95388-0008",Latitude:37.389467,Longitude:-120.6147,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(209) 357-6175",StatusType:"Active",Website:"www.winton.k12.ca.us"}
```
Because the right-hand-side value to which we were comparing was a string, it
was necessary to wrap it in quotes. If this string were written as a keyword,
it would have been interpreted as a field name as
Zed [field references](../language/overview.md#24-implied-field-references)
look like keywords in the context of an expression.

For example, to see the records in which the school and district name are the
same:
```mdtest-command dir=testdata/edu
zq -z 'District==School' schools.zson
```
produces
```mdtest-output head
{School:"Adelanto Elementary",District:"Adelanto Elementary",City:"Adelanto",County:"San Bernardino",Zip:"92301-1734",Latitude:34.576166,Longitude:-117.40944,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(760) 246-5892",StatusType:"Active",Website:null(string)}
{School:"Allensworth Elementary",District:"Allensworth Elementary",City:"Allensworth",County:"Tulare",Zip:"93219-9709",Latitude:35.864487,Longitude:-119.39068,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(661) 849-2401",StatusType:"Active",Website:null(string)}
{School:"Alta Loma Elementary",District:"Alta Loma Elementary",City:"Alta Loma",County:"San Bernardino",Zip:"91701-5007",Latitude:34.12597,Longitude:-117.59744,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(909) 484-5000",StatusType:"Active",Website:null(string)}
...
```

#### 3.5.1 Type Dependence

When comparing values to the named fields,
the value must be comparable to the _data type_ of the field.

For instance, the "Zip" field in the schools data is a `string` rather than
a number because of the extended ZIP+4 format that includes a hyphen and four
additional digits and hence could not be represented in a numeric type, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'cut Zip' schools.zson
```
produces
```mdtest-output head
{Zip:"95959"}
{Zip:"94607-1404"}
{Zip:"92395-3360"}
...
```
Because Zed does not coerce strings to numbers in expressions,
the predicate `Zip==95959` would _not_
match the top record shown, since Zed recognizes the bare value `95959` as a
number before comparing it to all the fields named `Zip`.
However, `Zip=="95959"` _would_ match, since the quotes cause Zed
to treat the value as a string.

When confronted with messy data like this, you can usually cleaned it up
to achieve the intent of your searches.  For example, the dash suffix
of the ZIP codes could be dropped, the string converted to an integer, then
integer comparisons performed, i.e.,
```mdtest-command dir=testdata/edu
zq -z 'cut Zip | int64(Zip[0:5])==94607' schools.zson
```
produces
```mdtest-output head
{Zip:"94607-1404"}
...
```

#### 3.5.2 Grep Predicates

When comparing a named field to a literal string, the quoted value is treated as
an _exact_ match.

For example, let's say we know there are several school names that start with
`Luther` but only a couple district names that do. Because `Luther` only appears
as a _substring_ of the district names in our sample data, the following example
produces no output, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'District=="Luther"' schools.zson
```
produces an empty output
```mdtest-output
```

To perform string searches inside of nested values, we can utilize the
[grep function](../language/functions/grep.md) with
a [glob](#32-globs), e.g.,
```mdtest-command dir=testdata/edu
zq -z 'grep(Luther*, District)' schools.zson
```
produces
```mdtest-output head
{School:"Luther Burbank Elementary",District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-1814",StatusType:"Active",Website:null(string)}
{School:null(string),District:"Luther Burbank",City:"San Jose",County:"Santa Clara",Zip:"95128-1931",Latitude:37.323556,Longitude:-121.9267,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(408) 295-2450",StatusType:"Active",Website:"www.lbsd.k12.ca.us"}
```

[Regular expressions](#33-regular-expressions) can also be used with `grep`, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'grep(/^Sunset (Ranch|Ridge) Elementary/, School)' schools.zson
```
produces
```mdtest-output
{School:"Sunset Ranch Elementary",District:"Rocklin Unified",City:"Rocklin",County:"Placer",Zip:"95765-5441",Latitude:38.826425,Longitude:-121.2864,Magnet:false,OpenDate:2010-08-17T00:00:00Z,ClosedDate:null(time),Phone:"(916) 624-2048",StatusType:"Active",Website:"www.rocklin.k12.ca.us"}
{School:"Sunset Ridge Elementary",District:"Pacifica",City:"Pacifica",County:"San Mateo",Zip:"94044-2029",Latitude:37.653836,Longitude:-122.47919,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(650) 738-6687",StatusType:"Active",Website:null(string)}
```

#### 3.5.3 Containment

Rather than testing for strict equality or pattern matches, you may want to
determine if a value is among the many possible elements of a complex field.
This is performed with `in`.

Since our sample data doesn't contain complex fields, we'll make one by
using the [`union`](../language/aggregates/union.md) aggregate function to
create a [`set`](../formats/zson.md#243-set-value)-typed
field called `Schools` that contains all unique school names per district. From
these we'll find each set that contains a school named `Lincoln Elementary`, e.g.,
```mdtest-command dir=testdata/edu
zq -Z 'Schools:=union(School) by District | "Lincoln Elementary" in Schools | sort this' schools.zson
```
produces
```mdtest-output head
{
    District: "Tulare City",
    Schools: |[
        "Alpine Vista",
        "Mulcahy Middle",
        "Tulare Support",
        "Live Oak Middle",
        "Los Tules Middle",
        "Maple Elementary",
        "Garden Elementary",
        "Wilson Elementary",
        "Cypress Elementary",
        "Lincoln Elementary",
        "Heritage Elementary",
        "Pleasant Elementary",
        "Cherry Avenue Middle",
        "Roosevelt Elementary",
        "Frank Kohn Elementary",
        "Mission Valley Elementary",
        "Tulare City Community Day"
    ]|
}
...
```

#### 3.5.4 Comparisons

In addition to testing for equality via `==` and testing containment via
`in`, the other common methods of comparison `!=`, `<`, `>`, `<=`, and
`>=` are also available.

For example, the following search finds the schools that reported the highest
math test scores,
```mdtest-command dir=testdata/edu
zq -z 'AvgScrMath > 690' testscores.zson
```
produces
```mdtest-output
{AvgScrMath:698(uint16),AvgScrRead:639(uint16),AvgScrWrite:664(uint16),cname:"Santa Clara",dname:"Fremont Union High",sname:"Lynbrook High"}
{AvgScrMath:699(uint16),AvgScrRead:653(uint16),AvgScrWrite:671(uint16),cname:"Alameda",dname:"Fremont Unified",sname:"Mission San Jose High"}
{AvgScrMath:691(uint16),AvgScrRead:638(uint16),AvgScrWrite:657(uint16),cname:"Santa Clara",dname:"Fremont Union High",sname:"Monta Vista High"}
```

The same approach can be used to compare characters in `string`-type values,
such as this search that finds school names at the end of the alphabet, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'School > "Z"' schools.zson
```
produces
```mdtest-output head
{School:"Zamora Elementary",District:"Woodland Joint Unified",City:"Woodland",County:"Yolo",Zip:"95695-5137",Latitude:38.658609,Longitude:-121.79355,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(530) 666-3641",StatusType:"Active",Website:null(string)}
{School:"Zamorano Elementary",District:"San Diego Unified",City:"San Diego",County:"San Diego",Zip:"92139-2989",Latitude:32.680338,Longitude:-117.03864,Magnet:true,OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(619) 430-1400",StatusType:"Active",Website:"http://new.sandi.net/schools/zamorano"}
{School:"Zane (Catherine L.) Junior High",District:"Eureka City High",City:"Eureka",County:"Humboldt",Zip:"95501-3140",Latitude:40.788118,Longitude:-124.14903,Magnet:false,OpenDate:1980-07-01T00:00:00Z,ClosedDate:1998-06-30T00:00:00Z,Phone:null(string),StatusType:"Merged",Website:null(string)}
...
```

### 3.6 Boolean Logic

Search terms can be combined with Boolean logic as detailed in
the [Zed language documentation](../language/overview.md#73-boolean-logic).

In particular, search terms separated by blank space implies
Boolean `and` between the concatenated terms.

Let's say we're earching for information about academies
that are flagged as being in a `Pending` status.  We can simply concatenate
the predicate for "Pending" and the keyword search for `academy`, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'StatusType=="Pending" academy' schools.zson
```
produces
```mdtest-output
{School:"Equitas Academy 4",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90015-2412",Latitude:34.044837,Longitude:-118.27844,Magnet:false,OpenDate:2017-09-01T00:00:00Z,ClosedDate:null(time),Phone:"(213) 201-0440",StatusType:"Pending",Website:"http://equitasacademy.org"}
{School:"Pinnacle Academy Charter - Independent Study",District:"South Monterey County Joint Union High",City:"King City",County:"Monterey",Zip:"93930-3311",Latitude:36.208934,Longitude:-121.13286,Magnet:false,OpenDate:2016-08-08T00:00:00Z,ClosedDate:null(time),Phone:"(831) 385-4661",StatusType:"Pending",Website:"www.smcjuhsd.org"}
{School:"Rocketship Futuro Academy",District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:false,OpenDate:2016-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
{School:"Sherman Thomas STEM Academy",District:"Madera Unified",City:"Madera",County:"Madera",Zip:"93638",Latitude:36.982843,Longitude:-120.06665,Magnet:false,OpenDate:2017-08-09T00:00:00Z,ClosedDate:null(time),Phone:"(559) 674-1192",StatusType:"Pending",Website:"www.stcs.k12.ca.us"}
{School:null(string),District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
```

Of course, the logical AND may also be explicit and the above query
can be written explicitly as
```
StatusType=="Pending" and academy
```

You can also combine predicates in a logical OR.
Let'a revisit two of our previous example searches that each only
returned a couple records, searching now with `or` to see them all at once,
e.g.,
```mdtest-command dir=testdata/edu
zq -z '"Defunct=" or ACE*Academy' schools.zson
```
produces
```mdtest-output
{School:"ACE Empower Academy",District:"Santa Clara County Office of Education",City:"San Jose",County:"Santa Clara",Zip:"95116-3423",Latitude:37.348601,Longitude:-121.8446,Magnet:false,OpenDate:2008-08-26T00:00:00Z,ClosedDate:null(time),Phone:"(408) 729-3920",StatusType:"Active",Website:"www.acecharter.org"}
{School:"ACE Inspire Academy",District:"San Jose Unified",City:"San Jose",County:"Santa Clara",Zip:"95112-6334",Latitude:37.350981,Longitude:-121.87205,Magnet:false,OpenDate:2015-08-03T00:00:00Z,ClosedDate:null(time),Phone:"(408) 295-6008",StatusType:"Active",Website:"www.acecharter.org"}
{School:"Lincoln Elem 'Defunct=",District:"Modesto City Elementary",City:null(string),County:"Stanislaus",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"Lovell Elem 'Defunct=",District:"Cutler-Orosi Joint Unified",City:null(string),County:"Tulare",Zip:null(string),Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1989-06-30T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
```

Use `not` to invert the matching logic in the term that comes to the right of
it in your search.

For example, to find schools in the `Dixon Unified` district _other than_
elementary schools, we invert the logic of a search term:
```mdtest-command dir=testdata/edu
zq -z 'not elementary District=="Dixon Unified"' schools.zson
```
produces
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

Note that `!` can also be used as alternative shorthand for `not`, e.g.,
```
! elementary District=="Dixon Unified"
```

#### 3.6.1 Logical Grouping

Unless wrapped in parentheses, a search is evaluated in _left-to-right order_.
Terms wrapped in parentheses will be evaluated _first_, overriding the default
left-to-right evaluation.

For example, we've noticed there are some test score records that have `null`
values for all three test scores:
```mdtest-command dir=testdata/edu
zq -z 'AvgScrMath==null AvgScrRead==null AvgScrWrite==null' testscores.zson
```
produces
```mdtest-output head
{AvgScrMath:null(uint16),AvgScrRead:null(uint16),AvgScrWrite:null(uint16),cname:"Riverside",dname:"Beaumont Unified",sname:"21st Century Learning Institute"}
{AvgScrMath:null(uint16),AvgScrRead:null(uint16),AvgScrWrite:null(uint16),cname:"Los Angeles",dname:"ABC Unified",sname:"ABC Secondary (Alternative)"}
...
```
We can easily filter these out by negating the search for these records, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'not (AvgScrMath==null AvgScrRead==null AvgScrWrite==null)' testscores.zson
```
produces
```mdtest-output head
{AvgScrMath:371(uint16),AvgScrRead:376(uint16),AvgScrWrite:368(uint16),cname:"Los Angeles",dname:"Los Angeles Unified",sname:"APEX Academy"}
{AvgScrMath:367(uint16),AvgScrRead:359(uint16),AvgScrWrite:369(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"ARISE High"}
{AvgScrMath:491(uint16),AvgScrRead:489(uint16),AvgScrWrite:484(uint16),cname:"Santa Clara",dname:"San Jose Unified",sname:"Abraham Lincoln High"}
...
```
Parentheses can also be nested, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'grep(*High*, sname) and (not (AvgScrMath==null AvgScrRead==null AvgScrWrite==null) and dname=="San Francisco Unified")' testscores.zson
```
produces
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

## 4. Record Operators

As with the data sets explored here, a very typical use case for Zed is
to operate over structured logs or events that are all represented as Zed records.
While Zed queries may operate over any sequence of values, the following operators
are designed specifically to work on sequences of records:
* [cut](../language/operators/cut.md) - extract subsets of record fields into new records
* [drop](../language/operators/drop.md) - drop fields from record values
* [fuse](../language/operators/fuse.md) - coerce all input values into a merged type
* [put](../language/operators/put.md) - add or modify fields of records
* [rename](../language/operators/rename.md) - change the name of record fields

### 4.1 [cut](../language/operators/cut.md)

`cut` produces output records from input records containing only
the specified named fields.

This example returns only the name and opening date from our school records:
```mdtest-command dir=testdata/edu
zq -Z 'cut School,OpenDate' schools.zson
```
produces
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
As long as some of the named fields are present, they are returned while absent
fields are `error("missing")`. For instance, the following
query is run against all three of our data sources and returns values from our
school data that includes fields for both `School` and `Website`, values from
our web address data that have the `Website` and `addr` fields, and the
missing value from the test score data since it has none of these fields:
```mdtest-command dir=testdata/edu
zq -z 'yosemiteuhsd | cut School,Website,addr' *.zson
```
produces
```mdtest-output
{School:null(string),Website:"www.yosemiteuhsd.com",addr:error("missing")}
{School:error("missing"),Website:"www.yosemiteuhsd.com",addr:104.253.209.210}
```
Here, we return only the `sname` and `dname` fields of the test scores while also
renaming the fields:
```mdtest-command dir=testdata/edu
zq -z 'cut School:=sname,District:=dname' testscores.zson
```
produces
```mdtest-output head
{School:"21st Century Learning Institute",District:"Beaumont Unified"}
{School:"ABC Secondary (Alternative)",District:"ABC Unified"}
...
```

### 4.2 [drop](../language/operators/drop.md)

`drop` produces output records from input records with the indicated
fields dropped from the output.

This example return all the fields _other than_ the score values in our test score data:
```mdtest-command dir=testdata/edu
zq -z 'drop AvgScrMath,AvgScrRead,AvgScrWrite' testscores.zson
```
produces
```mdtest-output head
{cname:"Riverside",dname:"Beaumont Unified",sname:"21st Century Learning Institute"}
{cname:"Los Angeles",dname:"ABC Unified",sname:"ABC Secondary (Alternative)"}
...
```

### 4.3 [fuse](../language/operators/fuse.md)

`fuse` produces output records from input records where the outputs
all have a uniform type consisting of a fusion of the input types.
Note that `fuse` operates in two passes: the first pass computes the
output type and the second pass tranforms the records.  Thus, all input
must be read before any output is produced.  If the input does not
fit in memory, it is spilled to temporary storage.

Let's say you'd started with table-formatted output of all records in our data
that reference the town of Geyserville, e.g.,

```mdtest-command dir=testdata/edu
zq -f table 'Geyserville' *.zson
```
produces
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
such as SQL. Indeed, `zq` halts its output in this case, e.g.,
```mdtest-command dir=testdata/edu fails
zq -f csv 'Geyserville' *.zson
```
produces
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
interruptions between the subsequent data rows, e.g.,
```mdtest-command dir=testdata/edu
zq -f csv 'Geyserville | fuse' *.zson
```
produces
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
In addition to the `csv` format, the `arrows`, `parquet`, `table`, and `zeek`
formats also benefit from fused records.

### 4.4 [put](../language/operators/drop.md)

`put` produces output records from input records and either mutates or
adds fields indicated by the expressions.

If multiple fields are written by `put`,  the new field values are computed first
and then they are all written simultaneously.  As a result, a computed value
cannot be referenced in another expression.  If you need to re-use a computed result,
this can be done by chaining multiple `put` operators.
For example, this will not work
```
put N:=len(somelist), isbig:=N>10
```
but it could be written instead as
```
put N:=len(somelist) | put isbig:=N>10
```
For example,
to add a field to our test score records representing the computed average of the math,
reading, and writing scores for each school that reported them, we could say:
```mdtest-command dir=testdata/edu
zq -Z 'AvgScrMath!=null | put AvgAll:=(AvgScrMath+AvgScrRead+AvgScrWrite)/3.0' testscores.zson
```
which produces
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
We can also use `put` to create derived tables and display them in tabular
form using `-f table`, e.g.,
```mdtest-command dir=testdata/edu
zq -f table 'AvgScrMath != null | put combined_scores:=AvgScrMath+AvgScrRead+AvgScrWrite | cut sname,combined_scores,AvgScrMath,AvgScrRead,AvgScrWrite | head 5' testscores.zson
```
produces
```mdtest-output
sname                       combined_scores AvgScrMath AvgScrRead AvgScrWrite
APEX Academy                1115            371        376        368
ARISE High                  1095            367        359        369
Abraham Lincoln High        1464            491        489        484
Abraham Lincoln Senior High 1319            462        432        425
Academia Avance Charter     1148            386        380        382
```
As noted above the `put` keyword is entirely optional. Here we omit
it and create a new field to hold the lowercase representation of
the school `District` field:
```mdtest-command dir=testdata/edu
zq -Z 'cut District | lower_district:=lower(District)' schools.zson
```
produces
```mdtest-output head
{
    District: "Nevada County Office of Education",
    lower_district: "nevada county office of education"
}
...
```

### 4.5 [rename](../language/operators/rename.md)

`rename` produces output records from input records where field mays are
change.  Note that a field's name can only be renamed as it exists inside
of the record and cannot be moved between sub-records in a nested value.

The rename steps are applied left-to-right.

Here is a simple example that renames some fields in our test score data
to match the field names from our school data:
```mdtest-command dir=testdata/edu
zq -Z 'rename School:=sname,District:=dname,City:=cname' testscores.zson
```
produces
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
The field `inner` can be renamed within that nested record, e.g.,
```mdtest-command
zq -Z 'rename outer.renamed:=outer.inner' nested.zson
```
produces
```mdtest-output
{
    outer: {
        renamed: "MyValue"
    }
}
```
However, an attempt to rename it to a top-level field will fail, e.g.,
```mdtest-command fails
zq -Z 'rename toplevel:=outer.inner' nested.zson
```
produces this compile-time error message and the query is not run:
```mdtest-output
cannot rename outer.inner to toplevel
```
This goal could instead be achieved by combining [`put`](#44-put) and [`drop`](#42-drop),
e.g.,
```mdtest-command
zq -Z 'put toplevel:=outer.inner | drop outer.inner' nested.zson
```
produces
```mdtest-output
{
    toplevel: "MyValue"
}
```

## 5. Aggregates

The [summarize operator](../language/operators/summarize.md)
performs zero or more aggregations with zero or more group-by expressions.
Each aggregation is performed by an
[aggregate function](../language/aggregates/README.md)
that operates on batches of records to carry out a running computation over
the values they contain.  The `summarize` keyword is optional as the operato
can be [inferred from context](../language/overview.md#26-implied-operators).

As with SQL, multiple aggregate functions may be invoked at the same time.
For example, to simultaneously calculate the minimum, maximum, and average of
the math test scores:
```mdtest-command dir=testdata/edu
zq -f table 'min(AvgScrMath),max(AvgScrMath),avg(AvgScrMath)' testscores.zson
```
produces
```mdtest-output
min max avg
289 699 484.99019042123484
```

### 5.1 Output Field Names

The output of an aggregation is a sequence of records that form a table,
and the field names are specified in the assignments of the aggregate
functions and the group-by assignemnts.  When an expression is given
without a field name, a name is derived from the expression.  If a name
cannot be derived, then a compile-time error is reported and the query
does not run.

As just shown, by default the result returned is placed in a field with the
same name as the aggregate function. You may instead use `:=` to specify an
explicit name for the generated field, e.g.,
```mdtest-command dir=testdata/edu
zq -f table 'lowest:=min(AvgScrMath),highest:=max(AvgScrMath),typical:=avg(AvgScrMath)' testscores.zson
```
produces
```mdtest-output
lowest highest typical
289    699     484.99019042123484
```

### 5.2 Grouping

All aggregate functions may be invoked with one or more
[group-by expressions](../language/operators/summarize.md), which forms one
or more group-by keys.  Each unique group-by set defines input values
upon which each aggregate function instance operates.
If no group-by expression is provided, the aggregate function
operates over all values in the input stream and a single record is the result.

### 5.3 Where Clause

A `where` clause may also be added to filter the values on which an aggregate
function will operate.
For example,
this query calculates average math test scores for the cities of Los Angeles
and San Francisco:
```mdtest-command dir=testdata/edu
zq -Z 'LA_Math:=avg(AvgScrMath) where cname=="Los Angeles", SF_Math:=avg(AvgScrMath) where cname=="San Francisco"' testscores.zson
```
produces
```mdtest-output
{
    LA_Math: 456.27341772151897,
    SF_Math: 485.3636363636364
}
```

### 5.4 Aggregate Functions

This section depicts examples of various
[aggregate functions](../language/overview.md#610-aggregate-function-calls)
operating over thes "schools data set".

#### 5.4.1 [and](../language/aggregates/and.md)

The `and` function accumulates a Boolean truth value based on the logical AND
of all of its input.

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities in which all schools have a website. e.g.,
```mdtest-command dir=testdata/edu
zq -Z 'all_schools_have_website:=and(Website!=null) by City | sort City' schools.zson
```
produces
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

#### 5.4.2 [any](../language/aggregates/any.md)

The `any` function produces one value from all of its input, chosen in
an undefined manner.

This query gives the name of one of the schools in our sample data:
```mdtest-command dir=testdata/edu
zq -z 'any(School)' schools.zson
```
For small inputs that fit in memory, this will typically be the first such
field in the stream, but in general you should not rely upon this.  In this
case, the output is:
```mdtest-output
{any:"'3R' Middle"}
```

#### 5.4.3 [avg](../language/aggregates/avg.md)

The `avg` function computes an arithmetic mean over all of all of its input.

This query calculates the average of the math test scores:
```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrMath)' testscores.zson
```
and produces
```mdtest-output
avg
484.99019042123484
```

#### 5.4.4 [collect](../language/aggregates/collect.md)

The `collect` function accumulates all of its input into an array.

For schools in Fresno county that include websites, the following query
constructs an ordered list per city of their websites along with a parallel
list of which school each website represents:
```mdtest-command dir=testdata/edu
zq -Z 'County=="Fresno" Website!=null | Websites:=collect(Website),Schools:=collect(School) by City | sort City' schools.zson
```
and produces
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

#### 5.4.5 [count](../language/aggregates/count.md)

The `count` function produces a count of all of its input values.

This query counts the number of records in each of our example data sources:
```mdtest-command dir=testdata/edu
zq -z 'count()' schools.zson
zq -z 'count()' testscores.zson
zq -z 'count()' webaddrs.zson
```
and produces
```mdtest-output
{count:17686(uint64)}
{count:2331(uint64)}
{count:2223(uint64)}
```
The `Website` field is known to be in our school and website address data
sources, but not in the test score data. To confirm this, we can count across
all data sources and specify the named field, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'count(Website)' *.zson
```
produces
```mdtest-output
{count:19909(uint64)}
```
Since `17686 + 2223 = 19909`, the count result is what we expected.

#### 5.4.6 [dcount](../language/aggregates/dcount.md)

The `dcount` function produces a distinct count of all of its input values,
i.e., the number of unique values in its input.

For large inputs, this value  is an approximation of the actual value.
The approcimation error is described in detail in the code and research linked
from the [HyperLogLog repository](https://github.com/axiomhq/hyperloglog).

This query generates an approcimate count the number of unique school names
in our sample data set:
```mdtest-command dir=testdata/edu
zq -Z 'dcount(School)' schools.zson
```
and produces
```mdtest-output
{
    dcount: 13804 (uint64)
}
```
To see the precise value, which may take longer to execute, this query
```mdtest-command dir=testdata/edu
zq -Z 'count() by School | count()' schools.zson
```
produces
```mdtest-output
{
    count: 13876 (uint64)
}
```
Here we saw the approximation was off by 0.3%.

#### 5.4.7 [max](../language/aggregates/max.md)

The `max` function computes the maximum numeric value over all of its input.

To see the highest reported math test score, this query:
```mdtest-command dir=testdata/edu
zq -f table 'max(AvgScrMath)' testscores.zson
```
produces
```mdtest-output
max
699
```

#### 5.4.8 [min](../language/aggregates/min.md)

The `min` function computes the minimum numeric value over all of its input.

To see the lowest reported math test score, this query
```mdtest-command dir=testdata/edu
zq -f table 'min(AvgScrMath)' testscores.zson
```
produces
```mdtest-output
min
289
```

#### 5.4.9 [or](../language/aggregates/or.md)

The `or` function accumulates a Boolean truth value based on the logical OR
of all of its input.

Many of the school records in our sample data include websites, but many do
not. The following query shows the cities for which at least one school has
a listed website:
```mdtest-command dir=testdata/edu
zq -Z 'has_at_least_one_school_website:=or(Website!=null) by City | sort City' schools.zson
```
and produces
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

#### 5.4.10 [sum](../language/aggregates/sum.md)

The `sum` function computes the minimum numeric value over all of its input.

This query calculates the total of all the math, reading, and writing test scores
across all schools:
```mdtest-command dir=testdata/edu
zq -Z 'AllMath:=sum(AvgScrMath),AllRead:=sum(AvgScrRead),AllWrite:=sum(AvgScrWrite)' testscores.zson
```
and produces
```mdtest-output
{
    AllMath: 840488 (uint64),
    AllRead: 832260 (uint64),
    AllWrite: 819632 (uint64)
}
```

#### 5.4.11 [union](../language/aggregates/union.md)

The `union` function computes a set union over all of this input.

For schools in Fresno county that include websites, the following query
constructs a set per city of all the unique websites for the schools in that
city:
```mdtest-command dir=testdata/edu
zq -Z 'County=="Fresno" Website!=null | Websites:=union(Website) by City | sort City' schools.zson
```
and produces
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

### 5.5 Group-by Examples

As mentioned above,
the `summarize` operator may include group-by expressions
that partitions the input sequence into groups that
are processed independently from on another.

The output order of values from each grouped aggregation is undefined.
To ensure a deterministic order,
a [`sort` operator](../language/operators/sort.md)
may be used downstream of the aggregation.

> In many of the examples, you will see a `sort` tacked onto the end of the
> computation.  This ensures a deterministic order and reliable testing since
> all of these examples are subject to automated testing.

The simplest group-by example summarizes the unique values of the named field(s),
which requires no aggregate function

For example, to see the different categories of status for the schools
in our example data, this query:

```mdtest-command dir=testdata/edu
zq -z 'by StatusType | sort' schools.zson
```
produces
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```
If you work a lot at the UNIX/Linux shell, you might have sought to accomplish
the same via a familiar idiom: `sort | uniq`.  This works in Zed, but the `by`
shorthand is preferable, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'cut StatusType | sort | uniq' schools.zson
```
produces
```mdtest-output
{StatusType:"Active"}
{StatusType:"Closed"}
{StatusType:"Merged"}
{StatusType:"Pending"}
```

When specifying multiple comma-separated field names, a group is formed for each
unique combination of values found in those fields.  To see the average reading
test scores and school count for each county/district pairing, this query:
```mdtest-command dir=testdata/edu
zq -f table 'avg(AvgScrRead),count() by cname,dname | sort -r count' testscores.zson
```
produces
```mdtest-output head
cname           dname                                              avg                count
Los Angeles     Los Angeles Unified                                416.83522727272725 202
San Diego       San Diego Unified                                  472                44
Alameda         Oakland Unified                                    414.95238095238096 27
San Francisco   San Francisco Unified                              454.36842105263156 26
...
```
Instead of a simple field name, any of the comma-separated group-by elements
can be any [Zed expression](../language/overview.md#6-expressions), which may
appear in the form of a field assignment `field:=expr`

To see a count of how many school names of a particular character length
appear in our example data, this query:
```mdtest-command dir=testdata/edu
zq -f table 'count() by Name_Length:=len(School) | sort -r' schools.zson
```
produces
```mdtest-output head
Name_Length count
89          2
85          2
84          2
83          1
...
```
The fields referenced in a `by` grouping may or may not be present, or may be
inconsistently present, in the given input records, in which case, the group-by
aggregation still proceeds but embeds any error conditions in the result,
When a value is missing for a specified field, it will appear as `error("missing")`.

For instance, if we'd made an typographical error in our
prior example when attempting to reference the `dname` field,
the misspelled column would appear as embedded missing errors, e.g.,
```mdtest-command dir=testdata/edu
zq -Z 'avg(AvgScrRead),count() by cname,dnmae | sort -r count' testscores.zson
```
produces
```mdtest-output head
{
    cname: "Los Angeles",
    dnmae: error("missing"),
    avg: 450.83037974683543,
    count: 469 (uint64)
}
{
    cname: "San Diego",
    dnmae: error("missing"),
    avg: 496.74789915966386,
    count: 168 (uint64)
}
...
```

## 6. Sorting

Zed provides a convenient way to sort data using the
[sort operator](../language/operators/sort.md).
All values in Zed have a well-defined sort order, even complex values
and values of different data types, so you can easily sort heterogenous
sequences of values.

This query sorts our test score records by average reading score:
```mdtest-command dir=testdata/edu
zq -z 'sort AvgScrRead' testscores.zson
```
and produces
```mdtest-output head
{AvgScrMath:352(uint16),AvgScrRead:308(uint16),AvgScrWrite:327(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"Oakland International High"}
{AvgScrMath:289(uint16),AvgScrRead:314(uint16),AvgScrWrite:312(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"Gompers (Samuel) Continuation"}
{AvgScrMath:450(uint16),AvgScrRead:321(uint16),AvgScrWrite:318(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"S.F. International High"}
{AvgScrMath:314(uint16),AvgScrRead:324(uint16),AvgScrWrite:321(uint16),cname:"Los Angeles",dname:"Norwalk-La Mirada Unified",sname:"El Camino High (Continuation)"}
{AvgScrMath:307(uint16),AvgScrRead:324(uint16),AvgScrWrite:328(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"North Campus Continuation"}
...
```
Now we'll sort the test score records first by average reading score and then
by average math score. Note how this changed the order of the bottom two
records in the result, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'sort AvgScrRead,AvgScrMath' testscores.zson
```
produces
```mdtest-output head
{AvgScrMath:352(uint16),AvgScrRead:308(uint16),AvgScrWrite:327(uint16),cname:"Alameda",dname:"Oakland Unified",sname:"Oakland International High"}
{AvgScrMath:289(uint16),AvgScrRead:314(uint16),AvgScrWrite:312(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"Gompers (Samuel) Continuation"}
{AvgScrMath:450(uint16),AvgScrRead:321(uint16),AvgScrWrite:318(uint16),cname:"San Francisco",dname:"San Francisco Unified",sname:"S.F. International High"}
{AvgScrMath:307(uint16),AvgScrRead:324(uint16),AvgScrWrite:328(uint16),cname:"Contra Costa",dname:"West Contra Costa Unified",sname:"North Campus Continuation"}
{AvgScrMath:314(uint16),AvgScrRead:324(uint16),AvgScrWrite:321(uint16),cname:"Los Angeles",dname:"Norwalk-La Mirada Unified",sname:"El Camino High (Continuation)"}
...
```
Here we'll find the counties with the most schools by using the
[`count()`](../language/aggregates/count.md) aggregate function and piping its
output to a `sort` in reverse order. Note that even though we didn't list a
field name as an explicit argument, the `sort` operator did what we wanted
because it found a field of the `uint64` [data type](../language/overview.md#5-data-types),
e.g.,
```mdtest-command dir=testdata/edu
zq -z 'count() by County | sort -r' schools.zson
```
produces
```mdtest-output head
{County:"Los Angeles",count:3636(uint64)}
{County:"San Diego",count:1139(uint64)}
{County:"Orange",count:886(uint64)}
...
```
Next we'll count the number of unique websites mentioned in our school
records. Since we know some of the records don't include a website, we'll
deliberately put the null values at the front of the list so we can see how
many there are, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'count() by Website | sort -nulls first Website' schools.zson
```
produces
```mdtest-output head
{Website:null(string),count:10722(uint64)}
{Website:"acornstooakscharter.org",count:1(uint64)}
{Website:"atlascharter.org",count:1(uint64)}
{Website:"bizweb.lightspeed.net/~leagles",count:1(uint64)}
...
```

## 7. Sequence Filters

Several Zed operators manipulate a sequence of values based on the order
in which they appear in the input:
* [head](../language/operators/head.md) - copy leading values of input sequence
* [tail](../language/operators/tail.md) - copy trailing values of input sequence
* [uniq](../language/operators/uniq.md) - deduplicate adjacent values

### 7.1 [head](../language/operators/head.md)

The `head` operator takes an integer argument `N` and copies the first N values
of its input to its output.

For example, this query selects the first school record:
```mdtest-command dir=testdata/edu
zq -Z 'head' schools.zson
```
and produces
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
To see the first five school records in Los Angeles county, this query
```mdtest-command dir=testdata/edu
zq -z 'County=="Los Angeles" | head 5' schools.zson
```
produces
```mdtest-output
{School:"ABC Adult",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90703-2801",Latitude:33.878924,Longitude:-118.07128,Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:null(time),Phone:"(562) 229-7960",StatusType:"Active",Website:"www.abcadultschool.com"}
{School:"ABC Charter Middle",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90017",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:2008-09-03T00:00:00Z,ClosedDate:2009-06-10T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:"www.abcsf.us"}
{School:"ABC Evening High School",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90701",Latitude:null(float64),Longitude:null(float64),Magnet:null(bool),OpenDate:1980-07-01T00:00:00Z,ClosedDate:1994-11-23T00:00:00Z,Phone:null(string),StatusType:"Closed",Website:null(string)}
{School:"ABC Secondary (Alternative)",District:"ABC Unified",City:"Cerritos",County:"Los Angeles",Zip:"90703-2301",Latitude:33.881547,Longitude:-118.04635,Magnet:false,OpenDate:1991-09-05T00:00:00Z,ClosedDate:null(time),Phone:"(562) 229-7768",StatusType:"Active",Website:null(string)}
{School:"APEX Academy",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90028-8526",Latitude:34.052234,Longitude:-118.24368,Magnet:false,OpenDate:2008-09-03T00:00:00Z,ClosedDate:null(time),Phone:"(323) 817-6550",StatusType:"Active",Website:null(string)}
```
### 7.2 [tail](../language/operators/tail.md)

The `tail` operator takes an integer argument `N` and copies the last N values
of its input to its output.

For example, this query selects the last school record:
```mdtest-command dir=testdata/edu
zq -Z 'tail' schools.zson
```
and produces
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
To see the last five school records in Los Angeles county, this query
```mdtest-command dir=testdata/edu
zq -z 'County=="Los Angeles" | tail 5' schools.zson
```
produces
```mdtest-output
{School:null(string),District:"Wiseburn Unified",City:"Hawthorne",County:"Los Angeles",Zip:"90250-6462",Latitude:33.920462,Longitude:-118.37839,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(310) 643-3025",StatusType:"Active",Website:"www.wiseburn.k12.ca.us"}
{School:null(string),District:"SBE - Anahuacalmecac International University Preparatory of North America",City:"Los Angeles",County:"Los Angeles",Zip:"90032-1942",Latitude:34.085085,Longitude:-118.18154,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 352-3148",StatusType:"Active",Website:"www.dignidad.org"}
{School:null(string),District:"SBE - Academia Avance Charter",City:"Highland Park",County:"Los Angeles",Zip:"90042-4005",Latitude:34.107313,Longitude:-118.19811,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 230-7270",StatusType:"Active",Website:"www.academiaavance.com"}
{School:null(string),District:"SBE - Prepa Tec Los Angeles High",City:"Huntington Park",County:"Los Angeles",Zip:"90255-4138",Latitude:33.983752,Longitude:-118.22344,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(323) 800-2741",StatusType:"Active",Website:"www.prepatechighschool.org"}
{School:null(string),District:"California Advancing Pathways for Students in Los Angeles County ROC/P",City:"Bellflower",County:"Los Angeles",Zip:"90706",Latitude:33.882509,Longitude:-118.13442,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(562) 866-9011",StatusType:"Active",Website:"www.CalAPS.org"}
```

### 7.3 [uniq](../language/operators/uniq.md)

The `uniq` operator copies input values that are different from the previous
input to the output.

Let's say you'd been looking at the contents of just the `District` and
`County` fields in the order they appear in the school data, e.g.,
```mdtest-command dir=testdata/edu
zq -z 'cut District,County' schools.zson
```
produces
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
To eliminate the adjacent lines that share the same field/value pairs,
this query
```mdtest-command dir=testdata/edu
zq -z 'cut District,County | uniq' schools.zson
```
produces
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

## 8. Value Construction

The [yield operator](../language/operators/yield.md) creates one or more output
values for each input value based on the one or more expressions provided
as arguments to yield.

This example produce two simpler records for every school record listing
the average math score with the school name and the county name:
```mdtest-command dir=testdata/edu
zq -Z 'AvgScrMath!=null | yield {school:sname,avg:AvgScrMath}, {county:cname,zvg:AvgScrMath}' testscores.zson
```
which produces
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
In earlier example, we used `put` to create a table using this query:
```mdtest-command dir=testdata/edu
zq -f table 'AvgScrMath != null | put combined_scores:=AvgScrMath+AvgScrRead+AvgScrWrite | cut sname,combined_scores,AvgScrMath,AvgScrRead,AvgScrWrite | head 5' testscores.zson
```
produces
```mdtest-output
sname                       combined_scores AvgScrMath AvgScrRead AvgScrWrite
APEX Academy                1115            371        376        368
ARISE High                  1095            367        359        369
Abraham Lincoln High        1464            491        489        484
Abraham Lincoln Senior High 1319            462        432        425
Academia Avance Charter     1148            386        380        382
```

The same result can be achieved by yielding a record literal,
sometimes with a more intuitive  structure, e.g.,
```mdtest-command dir=testdata/edu
zq -f table 'AvgScrMath != null | yield  {sname,combined_scores:AvgScrMath+AvgScrRead+AvgScrWrite,AvgScrMath,AvgScrRead,AvgScrWrite} | head 5' testscores.zson
```
produces
```mdtest-output
sname                       combined_scores AvgScrMath AvgScrRead AvgScrWrite
APEX Academy                1115            371        376        368
ARISE High                  1095            367        359        369
Abraham Lincoln High        1464            491        489        484
Abraham Lincoln Senior High 1319            462        432        425
Academia Avance Charter     1148            386        380        382
```
