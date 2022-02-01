### Function

&emsp; **to_lower** &mdash; convert a string to lower case

### Synopsis

```
to_lower(s: string) -> string
```
### Description

The _to_lower_ function converts all upper case Unicode characters in `s`
to lower case and returns the result.

### Examples

Split a semi-colon delimited list of fruits:
```mdtest-command
echo '"Zed"' | zq -z 'yield to_lower(this)' -
```
=>
```mdtest-output
"zed"
```
