### Operator

&emsp; **load** &mdash; automatically populates scratch pools

### Synopsis

```
load <pool> [@<branch>] [author <author>] [message <message>] [meta <meta>]
```
### Description

The `load` efficiently populates scratch pools based on data from other pools. A `branch` can be a
* Pool Identifier
* String

while `author`, `message`, and `meta` are strings
### Examples

_Given a data pool, `samples`, loaded with a schools.zson, grab all schools located in Orange county and
load into the empty pool, `Orange`_
```
zed query -z 'from samples | County=="Orange" | load Orange'
```

_Consider the above example, but operate under the branch, `test` with author `Steve`_
```
zed query -z 'from samples | County=="Orange" | load Orange@test author "Steve"'
```