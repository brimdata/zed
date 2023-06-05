### Aggregate Function

&emsp; **union** &mdash; set union of input values

### Synopsis
```
union(any) -> |[any]|
```

### Description

The _union_ aggregate function computes a set union of its input values.
If the values are of uniform type, then the output is a set of that type.
If the values are of mixed typs, the the output is a set of union of the
types encountered.

### Examples

Average value of simple sequence:
```mdtest-command
echo '1 2 3 3' | zq -z 'union(this)' -
```
=>
```mdtest-output
|[1,2,3]|
```

Continuous average of simple sequence:
```mdtest-command
echo '1 2 3 3' | zq -z 'yield union(this)' -
```
=>
```mdtest-output
|[1]|
|[1,2]|
|[1,2,3]|
|[1,2,3]|
```
Mixed types create a union type for the set elements:
```mdtest-command-issue-3610
echo '1 2 3 "foo"' | zq -z 'set:=union(this) | yield this,typeof(set)' -
```
=>
```mdtest-output-issue-3610
{set:|[1,2,3,"foo"]|}
<|[(int64,string)]|>
```
