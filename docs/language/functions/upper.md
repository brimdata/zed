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

[Slices](../expressions.md#slices) can be used to uppercase a subset of a string as well.

```mdtest-command
echo '"zed"' |
  zq -z 'func upper_first_char(str): (
           upper(str[0:1]) + str[1:]
         )
         yield upper_first_char(this)
  ' -
```
=>
```mdtest-output
"Zed"
```
