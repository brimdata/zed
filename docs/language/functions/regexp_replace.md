### Function

&emsp; **regexp_replace** &mdash; replace regular expression matches in a string

### Synopsis

```
regexp_replace(s: string, re: string|regexp, new: string) -> string
```
### Description

The _regexp_replace_ function substitutes all characters matching the regular
expression `re` in string `s` with the string `new`.

#### Example:

```mdtest-command
echo '"-ab-axxb-"' | zq -z 'yield regexp_replace(this, /a(x*)b/, "T")' -
```
=>
```mdtest-output
"-T-T-"
```
