### Operator

&emsp; **filter** &mdash; select values based on boolean search expression

### Synopsis
```
[filter] <search-expr>
```
### Description

The `filter` operator copies a filtered version of its input to its output by
applying a [search expression](../language.md#search-expressions) to each
input value and dropping each value not matched by the expression.

The "filter" keyword is optional since filters are
[implied operators](../language.md#implied-operators).
When Zed queries are run from a search, it is highly convenient to be able to omit
the "filter" keyword, but when filters appear in Zed source files, it is good practice
to include the optional keyword.

See the [search expression](../language.md#search-expressions) syntax for
a detailed description of the filter syntax.

### Examples

_A simple keyword search for "world"_
```mdtest-command
echo '"hello, world" "say hello" "goodbye, world"' | zq -z 'filter world' -
```
=>
```mdtest-output
"hello, world"
"goodbye, world"
```
_An arithmetic comparison_
```mdtest-command
echo '1 2 3' | zq -z 'filter this >= 2' -
```
=>
```mdtest-output
2
3
```
_The "filter" keyword may be dropped_
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
echo '1 2 3' | zq -z 'this >= 2 AND this <= 2' -
```
=>
```mdtest-output
2
```
_The AND operator may be omitted through predicate concatenation_
```mdtest-command
echo '1 2 3' | zq -z 'this >= 2 this <= 2' -
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
