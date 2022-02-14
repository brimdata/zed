### Operator

&emsp; **switch** &mdash; route values based on cases

### Synopsis

```
switch <expr> {
  case <const> => <leg>
  case <const> => <leg>
  ...
  [ default => <leg> ]
}

switch {
  case <bool-expr> => <leg>
  case <bool-expr> => <leg>
  ...
  [ default => <leg> ]
}
```
### Description

The `switch` operator routes input values to multiple, parallel legs of
the dataflow path based on case matching.

In this first form, the expression `<expr>` is evaluated for each input value
and its result is
compared with all of the case values, which must be distinct, compile-time constant
expressions.  The value is propagated to the matching leg.

In the second form, each case is evaluated for each input value
in the order that the cases appear.
The first case to match causes the input value to propagate to the corresponding leg.
Even if later cases match, only the first leg receives the value.

In either form, if no case matches, but a default is present,
then the value is routed to the default leg.  Otherwise, the value is dropped.

Only one default case is allowed and it may appear anywhere in the list of cases;
where it appears does not influence the result.

The output of a switch consists of multiple legs that must be merged.
If the downstream operator expects a single input, then the output legs are
merged with an automatically inserted [combine operator](combine.md).

### Examples

_Split input into even and odds_
```mdtest-command
echo '1 2 3 4' | zq -z 'switch ( case this%2==0 => {even:this} case this%2==1 => {odd:this}) | sort odd,even' -
```
=>
```mdtest-output
{odd:1}
{odd:3}
{even:2}
{even:4}
```
_Switch on this with a constant case_
```mdtest-command
echo '1 2 3 4' | zq -z 'switch this ( case 1 => yield "1!" default => yield string(this) ) | sort' -
```
=>
```mdtest-output
"1!"
"2"
"3"
"4"
```
