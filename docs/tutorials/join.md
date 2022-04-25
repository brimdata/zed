---
sidebar_position: 2
sidebar_label: Join
---

# Join Overview

This is a brief primer on Zed's experimental [join operator](../language/operators/join.md).

Currently, join is limited in the following ways:
* the joined inputs both come from the parent so the query must be split before join,
* only merge join is implemented requiring inputs to be explicitly sorted, and
* only equ-join is supported.

A more comprehensive join design with easier-to-use syntax is forthcoming.

## Example Data

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

## Inner Join

We'll start by outputting only the fruits liked by at least one person.
The name of the matching person is copied into a field of a different name in
the joined results.

Because we're performing an inner join (the default), the
explicit `inner` is not strictly necessary, but including it clarifies our intention.

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
produces
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
```

## Left Join

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
produces
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie",age:30}
{name:"avocado",color:"green",flavor:"savory"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn",age:14}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn",age:14}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn",age:14}
{name:"apple",color:"red",flavor:"tart",eater:"morgan",age:61}
{name:"apple",color:"red",flavor:"tart",eater:"chris",age:47}
```

## Right join

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
produces
```mdtest-output
{name:"jessie",age:30,likes:"plain",fruit:"figs"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"banana"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"strawberry"}
{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets",fruit:"dates"}
{name:"morgan",age:61,likes:"tart",fruit:"apple"}
{name:"chris",age:47,likes:"tart",fruit:"apple"}
```

## Inputs from Pools

As our prior examples all used `zq`, we used `file` in our `from()` block to
pull our respective inputs from named file sources. However, if the inputs are
stored in pools in a Zed lake, the pool names would instead be specified in the
`from()` block.

Here we'll load our input data to pools in a temporary Zed Lake, then execute
our inner join using `zed query`.

Notice that because we happened to use `-orderby` to sort our pools by the same
keys that we reference in our `join`, we did not need to use any explicit
upstream `sort`.

The Zed script `inner-join-pools.zed`:

```mdtest-input inner-join-pools.zed
from (
  pool fruit
  pool people
) | inner join on flavor=likes eater:=name
```

Populating the pools, then executing the Zed script:

```mdtest-command
export ZED_LAKE=lake
zed init -q
zed create -q -orderby flavor:asc fruit
zed create -q -orderby likes:asc people
zed load -q -use fruit fruit.ndjson
zed load -q -use people people.ndjson
zed query -z -I inner-join-pools.zed
```
produces
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
```

## Self Joins

In addition to the named files and pools like we've used in the prior examples,
Zed is also intended to work on a single sequence of data that is split
and joined to itself.  Here we'll combine our file
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
produces
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eater:"jessie"}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn"}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn"}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn"}
{name:"apple",color:"red",flavor:"tart",eater:"morgan"}
{name:"apple",color:"red",flavor:"tart",eater:"chris"}
```

## Multi-value Joins

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
ripe fruit in our `fruit.ndjson`
records. In the Zed script `multi-value-join.zed`, we create the keys as
embedded records inside each input record, using the same field names and data
types in each. We'll leave the created `fruitkey` records intact to show what
they look like, but since it represents redundant data, in practice we'd
typically [`drop`](#drop) it after the `join` in our Zed pipeline.

```mdtest-input multi-value-join.zed
from (
  file fruit.ndjson => put fruitkey:={name,color} | sort fruitkey
  file inventory.ndjson => put invkey:={name,color} | sort invkey
) | inner join on fruitkey=invkey quantity
```

Executing the Zed script:
```mdtest-command
zq -z -I multi-value-join.zed
```
produces
```mdtest-output
{name:"banana",color:"yellow",flavor:"sweet",fruitkey:{name:"banana",color:"yellow"},quantity:1000}
{name:"strawberry",color:"red",flavor:"sweet",fruitkey:{name:"strawberry",color:"red"},quantity:3000}
```

#### Embedding the entire opposite record

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
produces
```mdtest-output
{name:"figs",color:"brown",flavor:"plain",eaterinfo:{name:"jessie",age:30,likes:"plain"}}
{name:"banana",color:"yellow",flavor:"sweet",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"strawberry",color:"red",flavor:"sweet",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eaterinfo:{name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}}
{name:"apple",color:"red",flavor:"tart",eaterinfo:{name:"morgan",age:61,likes:"tart"}}
{name:"apple",color:"red",flavor:"tart",eaterinfo:{name:"chris",age:47,likes:"tart"}}
```
