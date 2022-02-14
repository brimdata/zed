### Operator

&emsp; **where** &mdash; select values based on a Boolean expression

### Synopsis
```
[where] <expr>
```
### Description

The `where` operator filters its input by applying a Boolean expression `<expr>`
to each input value and dropping each value for which the expression evaluates
to `false` or to an error.

The "where" keyword may be omitted in which case `<expr>` follows
the [search expression](../language.md#search-expressions) syntax.

When Zed queries are run interactively, it is highly convenient to be able to omit
the "where" keyword, but when filters appear in Zed source files, it is good practice
to include the optional keyword.

### Examples

_A simple keyword search for "world"_
```mdtest-command
echo '"hello, world" "say hello" "goodbye, world"' | zq -z 'where world' -
```
=>
```mdtest-output
"hello, world"
"goodbye, world"
```
_An arithmetic comparison_
```mdtest-command
echo '1 2 3' | zq -z 'where this >= 2' -
```
=>
```mdtest-output
2
3
```
_The "where" keyword may be dropped_
```mdtest-command
echo '1 2 3' | zq -z 'this >= 2' -
```
=>
```mdtest-output
2
3
```
_A filter with Boolean logic_
```mdtest-command
echo '1 2 3' | zq -z 'where this >= 2 AND this <= 2' -
```
=>
```mdtest-output
2
```
_The AND operator may be omitted through predicate concatenation_
```mdtest-command
echo '1 2 3' | zq -z 'where this >= 2 this <= 2' -
```
=>
```mdtest-output
2
```
_Concatenation for keyword search_
```mdtest-command
echo '"foo" "foo bar" "foo bar baz" "baz"' | zq -z 'foo bar' -
```
=>
```mdtest-output
"foo bar"
"foo bar baz"
```
_Search expressions match fields names_
```mdtest-command
echo '{foo:1} {bar:2} {foo:3}' | zq -z foo -
```
=>
```mdtest-output
{foo:1}
{foo:3}
```
_Boolean functions may be called_
```mdtest-command
echo '1 "foo" 10.0.0.1' | zq -z 'is(<int64>)' -
```
=>
```mdtest-output
1
```
_Boolean functions with Boolean logic_
```mdtest-command
echo '1 "foo" 10.0.0.1' | zq -z 'is(<int64>) or is(<ip>)' -
```
=>
```mdtest-output
1
10.0.0.1
```
