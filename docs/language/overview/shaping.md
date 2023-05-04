---
sidebar_position: 9
sidebar_label: Shaping
---

# Shaping

Data that originates from heterogeneous sources typically has
inconsistent structure and is thus difficult to reason about or query.
To unify disparate data sources, data is often cleaned up to fit into
a well-defined set of schemas, which combines the data into a unified
store like a data warehouse.

In Zed, this cleansing process is called "shaping" the data, and Zed leverages
its rich, [super-structured](../../formats/README.md#2-zed-a-super-structured-pattern)
type system to perform core aspects of data transformation.
In a data model with nesting and multiple scalar types (such as Zed or JSON),
shaping includes converting the type of leaf fields, adding or removing fields
to "fit" a given shape, and reordering fields.

While shaping remains an active area of development, the core functions in Zed
that currently perform shaping are:

* [`cast`](../functions/cast.md) - coerce a value to a different type
* [`crop`](../functions/crop.md) - remove fields from a value that are missing in a specified type
* [`fill`](../functions/fill.md) - add null values for missing fields
* [`order`](../functions/order.md) - reorder record fields
* [`shape`](../functions/shape.md) - apply `cast`, `fill`, and `order`

They all have the same signature, taking two parameters: the value to be
transformed and a type value for the target type.

> Another type of transformation that's needed for shaping is renaming fields,
> which is supported by the [`rename` operator](../operators/rename.md).
> Also, the [`yield` operator](../operators/yield.md)
> is handy for simply emitting new, arbitrary record literals based on
> input values and mixing in these shaping functions in an embedded record literal.
> The [`fuse` aggregate function](../aggregates/fuse.md) is also useful for fusing
> values into a common schema, though a type is returned rather than values.

In the examples below, we will use the following named type `connection`
that is stored in a file `connection.zed`
and is included in the example Zed queries with the `-I` option of `zq`:
```mdtest-input connection.zed
type socket = { addr:ip, port:port=uint16 }
type connection = {
    kind: string,
    client: socket,
    server: socket,
    vlan: uint16
}
```
We also use this sample JSON input in a file called `sample.json`:
```mdtest-input sample.json
{
  "kind": "dns",
  "server": {
    "addr": "10.0.0.100",
    "port": 53
  },
  "client": {
    "addr": "10.47.1.100",
    "port": 41772
  },
  "uid": "C2zK5f13SbCtKcyiW5"
}
```

## Cast

The `cast` function applies a cast operation to each leaf value that matches the
field path in the specified type, e.g.,
```mdtest-command
zq -Z -I connection.zed "cast(this, <connection>)" sample.json
```
casts the address fields to type `ip`, the port fields to type `port`
(which is a [named type](data-types.md#named-types) for type `uint16`) and the address port pairs to
type `socket` without modifying the `uid` field or changing the
order of the `server` and `client` fields:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: 10.0.0.100,
        port: 53 (port=uint16)
    } (=socket),
    client: {
        addr: 10.47.1.100,
        port: 41772
    } (socket),
    uid: "C2zK5f13SbCtKcyiW5"
}
```

## Crop

Cropping is useful when you want records to "fit" a schema tightly, e.g.,
```mdtest-command
zq -Z -I connection.zed "crop(this, <connection>)" sample.json
```
removes the `uid` field since it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    client: {
        addr: "10.47.1.100",
        port: 41772
    }
}
```

## Fill

Use `fill` when you want to fill out missing fields with nulls, e.g.,
```mdtest-command
zq -Z -I connection.zed "fill(this, <connection>)" sample.json
```
adds a null-valued `vlan` field since the input value is missing it and
the `connection` type has it:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    client: {
        addr: "10.47.1.100",
        port: 41772
    },
    uid: "C2zK5f13SbCtKcyiW5",
    vlan: null (uint16)
}
```

## Order

The `order` function changes the order of fields in its input to match the
order in the specified type, as field order is significant in Zed records, e.g.,
```mdtest-command
zq -Z -I connection.zed "order(this, <connection>)" sample.json
```
reorders the `client` and `server` fields to match the input but does nothing
about the `uid` field as it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: "10.47.1.100",
        port: 41772
    },
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    uid: "C2zK5f13SbCtKcyiW5"
}
```

## Shape

The `shape` function brings everything together by applying `cast`,
`fill`, and `order` all in one step, e.g.,
```mdtest-command
zq -Z -I connection.zed "shape(this, <connection>)" sample.json
```
reorders the `client` and `server` fields to match the input but does nothing
about the `uid` field as it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: 10.47.1.100,
        port: 41772 (port=uint16)
    } (=socket),
    server: {
        addr: 10.0.0.100,
        port: 53
    } (socket),
    vlan: null (uint16),
    uid: "C2zK5f13SbCtKcyiW5"
}
```
To get a tight shape of the target type,
apply `crop` to the output of `shape`, e.g.,
```mdtest-command
zq -Z -I connection.zed "shape(this, <connection>) | crop(this, <connection>)" sample.json
```
drops the `uid` field after shaping:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: 10.47.1.100,
        port: 41772 (port=uint16)
    } (=socket),
    server: {
        addr: 10.0.0.100,
        port: 53
    } (socket),
    vlan: null (uint16)
}
```
