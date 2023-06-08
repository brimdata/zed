---
sidebar_position: 4
sidebar_label: Const, Func, Operator, and Type Statements
---

# Statements

## Const Statements

Constants may be defined and assigned to a symbolic name with the syntax
```
const <id> = <expr>
```
where `<id>` is an identifier and `<expr>` is a constant [expression](expressions.md)
that must evaluate to a constant at compile time and not reference any
runtime state such as `this`, e.g.,
```mdtest-command
echo '{r:5}{r:10}' | zq -z "const PI=3.14159 2*PI*r" -
```
produces
```mdtest-output
31.4159
62.8318
```

One or more `const` statements may appear only at the beginning of a scope
(i.e., the main scope at the start of a Zed program or a [lateral scope](lateral-subqueries.md/#lateral-scope))
defined by an [`over` operator](operators/over.md)
and binds the identifier to the value in the scope in which it appears in addition
to any contained scopes.

A `const` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`const` statements may appear intermixed with `func` and `type` statements.

## Func Statements

User-defined functions may be created with the syntax
```
func <id> ( [<param> [, <param> ...]] ) : ( <expr> )
```
where `<id>` and `<param>` are identifiers and `<expr>` is an
[expression](expressions.md) that may refer to parameters but not to runtime
state such as `this`.

For example,
```mdtest-command
echo 1 2 3 4 | zq -z 'func add1(n): (n+1) add1(this)' -
```
produces
```mdtest-output
2
3
4
5
```

One or more `func` statements may appear at the beginning of a scope
(i.e., the main scope at the start of a Zed program or a [lateral scope](lateral-subqueries.md#lateral-scope)
defined by an [`over` operator](operators/over.md))
and binds the identifier to the expression in the scope in which it appears in addition
to any contained scopes.

A `func` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`func` statements may appear intermixed with `const` and `type` statements.

## Operator Statements

User-defined operators may be created with the syntax

```
op <id> ( [<param> [, <param> ...]] ) : (
  <sequence>
)
```
where `<id>` is the operator identifier, `<param>` are the parameters for the
operator, and `<sequence>` is the chain of operators (e.g., `operator | ...`)
where the operator does its work.

A user-defined operator can then be called with using the familiar call syntax
```
<id> ( [<expr> [, <expr> ...]] )
```
where `<id>` is the identifier of the user-defined operator and `<expr>` is a list
of [expressions](expressions.md) matching the number of `<param>`s defined in
the operator's signature.

### Sequence `this` Value

The `this` value of a user-defined operator's sequence is a record value
comprised of the parameters provided in the operator's signature.

For instance the program in `myop.zed`
```mdtest-input myop.zed
op myop(foo, bar, baz): (
  pass
)
myop("foo", true, {pi: this})
```
run via
```mdtest-command
echo 3.14 | zq -z -I myop.zed -
```
produces
```mdtest-output
{foo:"foo",bar:true,baz:{pi:3.14}}
```

### Spread Parameters

In addition to the standard named parameter syntax, user-defined operators may
use the spread operator `...` to indicate that the operator expects a record
value whose key/values will be expanded as entries in the operator's `this`
record value.

The most common use of spread parameters will be to carry the `this` value of
the calling context into the operator's sequence.

For instance the program in `spread.zed`
```mdtest-input spread.zed
op stamp(...): (
  put ts := 2021-01-01T00:00:00Z
)
stamp(this)
```
run via
```mdtest-command
echo '{foo:"foo",bar:"bar"}' | zq -z -I spread.zed -
```
produces
```mdtest-output
{foo:"foo",bar:"bar",ts:2021-01-01T00:00:00Z}
```

### Const Parameters

User-defined operators may use the `const` keyword to indicate that a parameter
is expecting a constant value. Const parameters are different from standard named
parameters in that they are not included in the operator's `this` value but can
be accessed within the operator's sequence as a variable.

For instance the program in `const.zed`
```mdtest-input const.zed
op find_host(..., const p, const h): (
  _path==p
  | hostname==h
)
find_host(this, "http", "google.com")
```
run via
```mdtest-command
echo '{_path:"http",hostname:"google.com"} {_path:"http",hostname:"meta.com"}' | zq -z -I const.zed -
```
produces
```mdtest-output
{_path:"http",hostname:"google.com"}
```

### Nested Calls

User-defined operators can make calls to other user-defined operators that
are declared within the same scope or in a parent's scope. To illustrate, a program in `nested.zed`
```mdtest-input nested.zed
op add2(x): (
  x := x + 2
)

op add4(x): (
  add2(x) | add2(x)
)

add4(x)
```
run via
```mdtest-command
echo '{x:1}' | zq -z -I nested.zed -
```
produces
```mdtest-output
{x:5}
```

One caveat with nested calls is that calls to other user-defined operators must
not produce a cycle, i.e., recursive and mutually recursive operators are not
allowed and will produce an error.

## Type Statements

Named types may be created with the syntax
```
type <id> = <type>
```
where `<id>` is an identifier and `<type>` is a [Zed type](data-types.md#first-class-types).
This creates a new type with the given name in the Zed type system, e.g.,
```mdtest-command
echo 80 | zq -z 'type port=uint16 cast(this, <port>)' -
```
produces
```mdtest-output
80(port=uint16)
```

One or more `type` statements may appear at the beginning of a scope
(i.e., the main scope at the start of a Zed program or a [lateral scope](lateral-subqueries.md#lateral-scope)
defined by an [`over` operator](operators/over.md))
and binds the identifier to the type in the scope in which it appears in addition
to any contained scopes.

A `type` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`type` statements may appear intermixed with `const` and `func` statements.
