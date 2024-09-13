---
sidebar_position: 3
sidebar_label: Join
---

# Join Overview

This is a brief primer on Zed's [`join` operator](../language/operators/join.md).

Currently, `join` is limited in that only equi-join (i.e., a join predicate
containing `=`) is supported.

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

The Zed script `inner-join.zed`:
```mdtest-input inner-join.zed
file fruit.ndjson
| inner join (
  file people.ndjson
) on flavor=likes eater:=name
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

:::tip note
In some databases a left join is called a _left outer join_.
:::

By performing a left join that targets the same key fields, now all of our
fruits will be shown in the results even if no one likes them (e.g., `avocado`).

As another variation, we'll also copy over the age of the matching person. By
referencing only the field name rather than using `:=` for assignment, the
original field name `age` is maintained in the results.

The Zed script `left-join.zed`:
```mdtest-input left-join.zed
file fruit.ndjson
| left join (
  file people.ndjson
) on flavor=likes eater:=name,age
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

:::tip note
In some databases a right join is called a _right outer join_.
:::

Next we'll change the join type from `left` to `right`. Notice that this causes
the `note` field from the right-hand input to appear in the joined results.

The Zed script `right-join.zed`:
```mdtest-input right-join.zed
file fruit.ndjson
| right join (
  file people.ndjson
) on flavor=likes fruit:=name
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

As our prior examples all used `zq`, we used the
[`file` operator](../language/operators/file.md) to pull our respective inputs
from named file sources.  However, if the inputs are stored in pools in a Zed
lake, we would instead specify those pools using the
[`from` operator](../language/operators/from.md).

Here we'll load our input data to pools in a temporary Zed lake, then execute
our inner join using `zed query`.

The Zed script `inner-join-pools.zed`:

```mdtest-input inner-join-pools.zed
from fruit
| inner join (
  from people
) on flavor=likes eater:=name
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

## Alternate Syntax

In addition to the syntax shown so far, `join` supports an alternate syntax in
which left and right inputs are specified by the two branches of a preceding
[`fork` operator](../language/operators/fork.md),
[`from` operator](../language/operators/from.md), or
[`switch` operator](../language/operators/switch.md).

Here we'll use the alternate syntax to perform the same inner join shown earlier
in the [Inner Join section](#inner-join).

The Zed script `inner-join-alternate.zed`:
```mdtest-input inner-join-alternate.zed
from (
  file fruit.ndjson
  file people.ndjson
) | inner join on flavor=likes eater:=name
```

Executing the Zed script:
```mdtest-command
zq -z -I inner-join-alternate.zed
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

## Self Joins

In addition to the named files and pools like we've used in the prior examples,
Zed is also intended to work on a single sequence of data that is split and
joined to itself.  Here we'll combine our file sources into a stream that we'll
pipe into `zq` via stdin.  Because `join` requires two separate inputs, here
we'll use the `has()` function inside a `switch` operator to identify the
records in the stream that will be treated as the left and right sides.  Then
we'll use the [alternate syntax for `join`](#alternate-syntax) to read those two
inputs.

The Zed script `inner-join-streamed.zed`:

```mdtest-input inner-join-streamed.zed
switch (
  case has(color) => pass
  case has(age) => pass
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

The equality test in a Zed `join` accepts only one named key from each input.
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
typically [`drop`](../language/operators/drop.md) it after the `join` in our Zed pipeline.

```mdtest-input multi-value-join.zed
file fruit.ndjson | put fruitkey:={name,color}
| inner join (
  file inventory.ndjson | put invkey:={name,color}
) on fruitkey=invkey quantity
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

## Joining More Than Two Inputs

While the `join` operator takes only two inputs, more inputs can be joined by
extending the Zed pipeline.

To illustrate this, we'll introduce some new input data in `prices.ndjson`.

```mdtest-input prices.ndjson
{"name":"apple","price":3.15}
{"name":"banana","price":4.01}
{"name":"avocado","price":2.50}
{"name":"strawberry","price":1.05}
{"name":"dates","price":6.70}
{"name":"figs","price": 1.60}
```

In our Zed script `three-way-join.zed` we'll extend the pipeline we used
previously for our inner join by piping its output to an additional join
against the price list.

```mdtest-input three-way-join.zed
file fruit.ndjson
| inner join (
  file people.ndjson
) on flavor=likes eater:=name
| inner join (
  file prices.ndjson
) on name=name price:=price
```

Executing the Zed script:

```mdtest-command
zq -z -I three-way-join.zed
```

produces

```mdtest-output
{name:"apple",color:"red",flavor:"tart",eater:"morgan",price:3.15}
{name:"apple",color:"red",flavor:"tart",eater:"chris",price:3.15}
{name:"banana",color:"yellow",flavor:"sweet",eater:"quinn",price:4.01}
{name:"dates",color:"brown",flavor:"sweet",note:"in season",eater:"quinn",price:6.7}
{name:"figs",color:"brown",flavor:"plain",eater:"jessie",price:1.6}
{name:"strawberry",color:"red",flavor:"sweet",eater:"quinn",price:1.05}
```

## Including the entire opposite record

In the current `join` implementation, explicit entries must be provided in the
`[field-list]` in order to copy values from the opposite input into the joined
results (a possible future enhancement [zed/2815](https://github.com/brimdata/zed/issues/2815)
may improve upon this). This can be cumbersome if your goal is to copy over many
fields or you don't know the names of all desired fields.

One way to work around this limitation is to specify `this` in the field list
to copy the contents of the _entire_ opposite record into an embedded record
in the result.

The Zed script `embed-opposite.zed`:

```mdtest-input embed-opposite.zed
file fruit.ndjson
| inner join (
  file people.ndjson
) on flavor=likes eaterinfo:=this
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

If embedding the opposite record is undesirable, the left and right
records can easily be merged with the
[spread operator](../language/expressions.md#record-expressions). Additional
processing may be necessary to handle conflicting field names, such as
in the example just shown where the `name` field is used differently in the
left and right inputs. We'll demonstrate this by augmenting `embed-opposite.zed`
to produce `merge-opposite.zed`.

```mdtest-input merge-opposite.zed
file fruit.ndjson
| inner join (
  file people.ndjson
) on flavor=likes eaterinfo:=this
| rename fruit:=name
| yield {...this,...eaterinfo}
| drop eaterinfo
```

Executing the Zed script:

```mdtest-command
zq -z -I merge-opposite.zed
```

produces

```mdtest-output
{fruit:"figs",color:"brown",flavor:"plain",name:"jessie",age:30,likes:"plain"}
{fruit:"banana",color:"yellow",flavor:"sweet",name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}
{fruit:"strawberry",color:"red",flavor:"sweet",name:"quinn",age:14,likes:"sweet",note:"many kids enjoy sweets"}
{fruit:"dates",color:"brown",flavor:"sweet",note:"many kids enjoy sweets",name:"quinn",age:14,likes:"sweet"}
{fruit:"apple",color:"red",flavor:"tart",name:"morgan",age:61,likes:"tart"}
{fruit:"apple",color:"red",flavor:"tart",name:"chris",age:47,likes:"tart"}
```
