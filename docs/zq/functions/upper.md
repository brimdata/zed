### Function

&emsp; **upper** &mdash; convert a string to upper case

### Synopsis

```
upper(s: string) -> string
```
### Description

The _upper_ function converts all lower case Unicode characters in `s`
to upper case and returns the result.

### Examples

```mdtest-command
echo '"Zed"' | zq -z 'yield upper(this)' -
```
=>
```mdtest-output
"ZED"
```
