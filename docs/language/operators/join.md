### Operator

&emsp; **join** &mdash; combine data from two inputs using a join predicate

### Synopsis

```
( => left path => right path )
| [anti|inner|left|right] join on <left-key>=<right-key> [<field>:=<right-expr>, ...]
```
### Description

The `join` operator combines records from two inputs based on whether
the `<left-key>` expression (evaluated in the context of the left input)
is equal to the `<right-key>` expression (evaluated in the context of
the right input) omitting values where there is no match (or including them
in the case of anti join).

The available join types are:
* _inner_ - output only values that match
* _left_ - output all left values with merged components from `<right-expr>`
* _right_ - output as a left join but with the roles of the inputs and `<right-expr>` reversed
* _anti_ - output left values whose left key does not have a matching right key

For anti join, the `<right-expr>` is undefined and thus cannot be specified.

> Currently, only exact equi-join is supported and the inputs must be sorted
> in ascending order by their respective keys.  Also, the join keys must
> be field expressions.  A future version of join will not require sorted inputs
> and will have more flexible join expressions.

### Examples

The [join tutorial](../../tutorials/join.md) includes several examples.
