# Operators

> **Note:** Many examples below use the
> [educational sample data](../../testdata/edu).

A pipeline may contain one or more _operators_ to transform or filter records.
You can imagine the data flowing left-to-right through an operator, with its
functionality further determined by arguments you may set. Operator names are
case-insensitive.

The following available operators are documented in detail below:

* [`cut`](#cut)
* [`drop`](#drop)
* [`filter`](#filter)
* [`fuse`](#fuse)
* [`head`](#head)
* [`join`](#join)
* [`pick`](#pick)
* [`put`](#put)
* [`rename`](#rename)
* [`sort`](#sort)
* [`tail`](#tail)
* [`traverse`](#traverse)
* [`uniq`](#uniq)

---

# Available Operators

## `cut`

|                           |                                                   |
| ------------------------- | ------------------------------------------------- |
| **Description**           | Return the data only from the specified named fields, where available. Contrast with [`pick`](#pick), which is stricter. |
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

As long as some of the named fields are present, these will be returned. No
warning is generated regarding absent fields. For instance, the following
query is run against all three of our data sources and returns values from our
school data that includes fields for both `School` and `Website`, values from
our web address data that have the `Website` and `addr` fields, and nothing
from the test score data since it has none of these fields.

```mdtest-command dir=testdata/edu
zq -z 'yosemiteuhsd | cut School,Website,addr' *.zson
```

#### Output:
```mdtest-output
{School:null(string),Website:"www.yosemiteuhsd.com"}
{Website:"www.yosemiteuhsd.com",addr:104.253.209.210}
```

Contrast this with a [similar example](#example-2-3) that shows how
[`pick`](#pick)'s stricter behavior only returns results when _all_ of the
named fields are present.

#### Example #3:

If no records are found that contain any of the named fields, `cut` returns a
warning.

```mdtest-command dir=testdata/edu
zq -z 'cut nothere,alsoabsent' testscores.zson
```

#### Output:
```mdtest-output
cut: no record found with columns nothere,alsoabsent
```

#### Example #4:

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

---

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

---

## `filter`

|                           |                                                                       |
| ------------------------- | --------------------------------------------------------------------- |
| **Description**           | Apply a search to potentially trim data from the pipeline.            |
| **Syntax**                | `filter <search>`                                                     |
| **Required<br>arguments** | `<search>`<br>Any valid Zed [search syntax](search-syntax.md) |
| **Optional<br>arguments** | None                                                                  |

> **Note:** As searches may appear anywhere in a Zed pipeline, it is not
> strictly necessary to enter the explicit `filter` operator name before your
> search. However, you may find it useful to include it to help express the
> intent of your query.

#### Example #1:

To further trim the data returned in our [`cut`](#cut) example:

```mdtest-command dir=testdata/edu
zq -Z 'cut School,OpenDate | filter School=="Breeze Hill Elementary"' schools.zson
```

#### Output:
```mdtest-output
{
    School: "Breeze Hill Elementary",
    OpenDate: 1992-07-06T00:00:00Z
}
```

#### Example #2:

An alternative syntax for our [`and` example](search-syntax.md#and):

```mdtest-command dir=testdata/edu
zq -z 'filter StatusType=="Pending" academy' schools.zson
```

#### Output:
```mdtest-output
{School:"Equitas Academy 4",District:"Los Angeles Unified",City:"Los Angeles",County:"Los Angeles",Zip:"90015-2412",Latitude:34.044837,Longitude:-118.27844,Magnet:false,OpenDate:2017-09-01T00:00:00Z,ClosedDate:null(time),Phone:"(213) 201-0440",StatusType:"Pending",Website:"http://equitasacademy.org"}
{School:"Pinnacle Academy Charter - Independent Study",District:"South Monterey County Joint Union High",City:"King City",County:"Monterey",Zip:"93930-3311",Latitude:36.208934,Longitude:-121.13286,Magnet:false,OpenDate:2016-08-08T00:00:00Z,ClosedDate:null(time),Phone:"(831) 385-4661",StatusType:"Pending",Website:"www.smcjuhsd.org"}
{School:"Rocketship Futuro Academy",District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:false,OpenDate:2016-08-15T00:00:00Z,ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
{School:"Sherman Thomas STEM Academy",District:"Madera Unified",City:"Madera",County:"Madera",Zip:"93638",Latitude:36.982843,Longitude:-120.06665,Magnet:false,OpenDate:2017-08-09T00:00:00Z,ClosedDate:null(time),Phone:"(559) 674-1192",StatusType:"Pending",Website:"www.stcs.k12.ca.us"}
{School:null(string),District:"SBE - Rocketship Futuro Academy",City:"Concord",County:"Contra Costa",Zip:"94521-1522",Latitude:37.965658,Longitude:-121.96106,Magnet:null(bool),OpenDate:null(time),ClosedDate:null(time),Phone:"(301) 789-5469",StatusType:"Pending",Website:"www.rsed.org"}
```

---

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

---

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

---

## `join`

|                           |                                               |
| ------------------------- | --------------------------------------------- |
| **Description**           | Return records derived from two inputs when particular values match between them.<br><br>The inputs must be sorted in the same order by their respective join keys. If an input source is already known to be sorted appropriately (either in an input file/object/stream, or if the data is pulled from a [Zed Lake](../lake/README.md) that's ordered by this key) an explicit upstream [`sort`](#sort) is not required. ||
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
  file fruit.ndjson => sort flavor;
  file people.ndjson => sort likes;
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
  file fruit.ndjson => sort flavor;
  file people.ndjson => sort likes;
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
  file fruit.ndjson => sort flavor;
  file people.ndjson => sort likes;
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
our inner join using `zed lake query`. If the Zed Lake had been fronted by a
`zed lake serve` process, the equivalent operations would be performed over the
network via `zed api`.

Notice that because we happened to use `-orderby` to sort our Pools by the same
keys that we reference in our `join`, we did not need to use any explicit
upstream `sort`.

The Zed script `inner-join-pools.zed`:

```mdtest-input inner-join-pools.zed
from (
  fruit => pass;
  people => pass;
) | inner join on flavor=likes eater:=name
```

Populating the Pools, then executing the Zed script:

```mdtest-command
mkdir lake
export ZED_LAKE_ROOT=lake
zed lake init -q
zed lake create -q -orderby flavor:asc fruit
zed lake create -q -orderby likes:asc people
zed lake load -q -use fruit@main fruit.ndjson
zed lake load -q -use people@main people.ndjson
zed lake query -z -I inner-join-pools.zed
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
  has(color) => sort flavor;
  has(age) => sort likes;
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
  file fruit.ndjson => put fruitkey:={name:string(name),color:string(color)} | sort fruitkey;
  file inventory.ndjson => put invkey:={name:string(name),color:string(color)} | sort invkey;
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
  file fruit.ndjson => sort flavor;
  file people.ndjson => sort likes;
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

---

## `pick`

|                           |                                               |
| ------------------------- | --------------------------------------------- |
| **Description**           | Return the data from the named fields in records that contain _all_ of the specified fields. Contrast with [`cut`](#cut), which is more relaxed. |
| **Syntax**                | `pick <field-list>`                           |
| **Required<br>arguments** | `<field-list>`<br>One or more comma-separated field names or assignments.  |

#### Example #1:

To return only the name and opening date from our school records:

```mdtest-command dir=testdata/edu
zq -Z 'pick School,OpenDate' schools.zson
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

All of the named fields must be present in a record for `pick` to return a
result for it. For instance, since only our school data has _both_ `School`
and `Website` fields, the following query of all three example data sources
only returns a result from the school data.

```mdtest-command dir=testdata/edu
zq -z 'yosemiteuhsd | pick School,Website' *.zson
```

#### Output:
```mdtest-output
{School:null(string),Website:"www.yosemiteuhsd.com"}
```

Contrast this with a [similar example](#example-2) that shows how
[`cut`](#cut)'s relaxed behavior returns a result whenever _any_ of the named
fields are present.

#### Example #3:

If no records are found that contain any of the named fields, `pick` returns a
warning.

```mdtest-command dir=testdata/edu
zq -z 'pick nothere,alsoabsent' testscores.zson
```

#### Output:
```mdtest-output
pick: no record found with columns nothere,alsoabsent
```

#### Example #4:

To return only the `sname` and `dname` fields of the test scores while also
renaming the fields:

```mdtest-command dir=testdata/edu
zq -z 'pick School:=sname,District:=dname' testscores.zson
```

#### Output:
```mdtest-output head
{School:"21st Century Learning Institute",District:"Beaumont Unified"}
{School:"ABC Secondary (Alternative)",District:"ABC Unified"}
...
```

---

## `put`

|                           |                                                 |
| ------------------------- | ----------------------------------------------- |
| **Description**           | Add/update fields based on the results of an expression.<br><br>If evaluation of any expression fails, a warning is emitted and the original record is passed through unchanged.<br><br>As this operation is very common, the `put` keyword is optional. |
| **Syntax**                | `[put] <field> := <expression> [, (<field> := <expression>)...]` |
| **Required arguments**    | One or more of:<br><br>`<field> := <expression>`<br>Any valid Zed [expression](expressions.md), preceded by the assignment operator `:=` and the name of a field in which to store the result. |
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

---

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

---

## `sort`

|                           |                                                                           |
| ------------------------- | ------------------------------------------------------------------------- |
| **Description**           | Sort records based on the order of values in the specified named field(s).|
| **Syntax**                | `sort [-r] [-nulls first\|last] [field-list]`                             |
| **Required<br>arguments** | None                                                                      |
| **Optional<br>arguments** | `[-r]`<br>If specified, results will be sorted in reverse order.<br><br>`[-nulls first\|last]`<br>Specifies where null values (i.e., values that are unset or that are not present at all in an incoming record) should be placed in the output.<br><br>`[field-list]`<br>One or more comma-separated field names by which to sort. Results will be sorted based on the values of the first field named in the list, then based on values in the second field named in the list, and so on.<br><br>If no field list is provided, `sort` will automatically pick a field by which to sort. It does so by examining the first input record and finding the first field in left-to-right order that is of a Zed integer [data type](data-types.md) (`int8`, `uint8`, `int16`, `uint16`, `int32`, `uint32`, `int64`, `uint64`) or, if no integer field is found, the first field that is of a floating point data type (`float16`, `float32`, `float64`). If no such numeric field is found, `sort` finds the first field in left-to-right order that is _not_ of the `time` data type. Note that there are some cases (such as the output of a [grouped aggregation](grouping.md#note-undefined-order) performed on heterogeneous data) where the first input record to `sort` may vary even when the same query is executed repeatedly against the same data. If you require a query to show deterministic output on repeated execution, an explicit field list must be provided. |

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
[`count()`](aggregate-functions.md#count) aggregate function and piping its
output to a `sort` in reverse order. Note that even though we didn't list a
field name as an explicit argument, the `sort` operator did what we wanted
because it found a field of the `uint64` [data type](data-types.md).

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
deliberately put the unset values at the front of the list so we can see how
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

---

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

---

## `traverse`

```
traverse foo <collection> [=> (seq <sequence>)]
```

`traverse` emits each element in collection (i.e., values of type `array`,
`set`, `record`\*, `map`\*) `foo` as a separate value to child
nodes in the flowgraph. If the optional sequence `seq`\*\* argument is provided the
elements of `foo` are read into `seq` as a separate stream sending an `EOS` when
the last element is traversed. The each element can be accessed inside `seq` by
referencing the `this` root record.

The ability to have sub sequence with traverse is a powerful feature: it allows
users to leverage the full power of the Zed language on an single collection
value. For instance the sum of elements in an array can be computed with
`traverse a => (sum(this))`.

\* `traverse` is currently in beta and does not currently support iterating over
records and maps. This will be added shortly.

\*\* `traverse` is currently in beta and works on a subset of the available 
operators. It has been tested with `filter`, `cut` and `pick` but users who use
`traverse` with `summarize` and `sort` will get weird results.

#### Example (basic):

```mdtest-command
echo '{a:[3,2,1]}' | zq -z 'traverse a' -
```

#### Output:
```mdtest-output
3
2
1
```

#### Example (filter)

```mdtest-command
echo '{a:[6,5,4]} {a:[3,2,1]}' | zq -z 'traverse a => (mod(this, 2) == 0)' -
```

#### Output:
```mdtest-output
6
4
2
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
