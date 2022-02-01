### Function

&emsp; **replace** &mdash; substitute one string for another

### Synopsis

```
replace(s: string, old: string, new: string) -> string
```
### Description

The _replace_ function substitutes all instances of the string `old`
that occur in string `s` with the string `new`.


#### Example:

```mdtest-command
echo '"oink oink oink"' | zq -z 'yield replace(this, "oink", "moo")' -
```
=>
```mdtest-output
"moo moo moo"
```
