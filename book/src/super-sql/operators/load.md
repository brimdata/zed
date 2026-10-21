# load

[✅](../intro.md#data-order)&ensp; add and commit data to a table

## Synopsis

```
load <table>[@<branch>] [ ( [author <author>] [message <message>] [meta <meta>] ) ]
```

>[!NOTE]
> The `load` operator is exclusively for working with a
>[database](../../command/db.md) and is not available when running
>[`super`](../../command/super.md) detached from a database.

## Description

The `load` operator populates the specified `<table>` with the values it
receives as input. Much like how [`super db load`](../../command/db.md#super-db-load)
is used at the command line to populate a table with data from files, streams,
and URIs, the `load` operator is used to save query results from your SuperSQL
query to a table in the same database. `<table>` is a string indicating the
name or ID of the destination table.
If the optional `@<branch>` string is included then the data will be committed
to an existing branch of that name, otherwise the `main` branch is assumed.
The `author`, `message`, and `meta` strings may also be provided to further
describe the committed data, similar to the same `super db load` options.

## Input Data

Examples below assume the existence of a database created and populated
by the following commands:

```mdtest-command
super db -q init example
super db -attach example -q create -orderby flip:asc coinflips
super db -attach example branch -q -use coinflips onlytails
echo '{flip:1,result:"heads"} {flip:2,result:"tails"}' |
  super db -attach example load -q -use coinflips -
super db -attach example -q create -orderby flip:asc bigflips
super db -attach example -f line -c '
  from :branches
  | values table.name || "@" || branch.name
  | sort'
```

The database then contains the two tables:

```mdtest-output
bigflips@main
coinflips@main
coinflips@onlytails
```

## Examples

_Modify some values, load them into the `main` branch of our empty `bigflips` table, and see what was loaded_
```mdtest-command
super db -attach example -c '
  from coinflips
  | result:=upper(result)
  | load bigflips
' > /dev/null

super db -attach example -s -c 'from bigflips'
```
=>
```mdtest-output
{flip:1,result:"HEADS"}
{flip:2,result:"TAILS"}
```

_Add a filtered subset of records to our `onlytails` branch, while also adding metadata_
```mdtest-command
super db -attach example -c '
  from coinflips
  | result=="tails"
  | load coinflips@onlytails (
      author "Steve"
      message "A subset"
      meta "\"Additional metadata\""
    )
' > /dev/null

super db -attach example -s -c 'from coinflips@onlytails'
```
=>
```mdtest-output
{flip:2,result:"tails"}
```
