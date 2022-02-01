### Function

&emsp; **typename** &mdash; lookup and return a named type

### Synopsis

```
typename(s: string) -> type
```
### Description

The _typename_ function returns the [type](../../formats/zson.md#357-type-type) of the
named type give by `name` if it exists.  Otherwise, `error("missing")` is returned.

### Examples

Return a simple named type with a string constant argument:
```mdtest-command
echo  '80(port=int16)' | zq -z 'yield typename("port")' -
```
=>
```mdtest-output
<port=int16>
```
Return a named type using an expression:
```mdtest-command
echo  '{name:"port",p:80(port=int16)}' | zq -z 'yield typename(name)' -
```
=>
```mdtest-output
<port=int16>
```
The result is `error("missing")` if the type name does not exist:
```mdtest-command
echo  '80' | zq -z 'yield typename("port")' -
```
=>
```mdtest-output
error("missing")
```
