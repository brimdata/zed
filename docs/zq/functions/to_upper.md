### Function

&emsp; **to_upper** &mdash; convert a string to upper case

### Synopsis

```
to_upper(s: string) -> string
```
### Description

The _to_upper_ function converts all lower case Unicode characters in `s`
to upper case and returns the result.

### Examples

Split a semi-colon delimited list of fruits:
```mdtest-command
echo '"Zed"' | zq -z 'yield to_upper(this)' -
```
=>
```mdtest-output
"ZED"
```
