### Aggregate Function

&emsp; **map** &mdash; aggregate map values into a single map

### Synopsis
```
map(|{any:any}|) -> |{any:any}|
```

### Description

The _map_ aggregate function combines map inputs into a single map output.
If _map_ receives multiple values for the same key, the last value received is
retained. If the input keys or values vary in type, the return type will be a map
of union of those types.

### Examples

Combine a sequence of records into a map:
```mdtest-command
echo '{stock:"APPL",price:145.03} {stock:"GOOG",price:87.07}' | zq -z 'map(|{stock:price}|)' -
```
=>
```mdtest-output
|{"APPL":145.03,"GOOG":87.07}|
```

Continuous collection over a simple sequence:
```mdtest-command
echo '|{"APPL":145.03}| |{"GOOG":87.07}| |{"APPL":150.13}|' | zq -z 'yield map(this)' -
```
=>
```mdtest-output
|{"APPL":145.03}|
|{"APPL":145.03,"GOOG":87.07}|
|{"APPL":150.13,"GOOG":87.07}|
```
