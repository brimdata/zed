---
sidebar_position: 2
sidebar_label: Const Statements
---

# Const Statements

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
defined by an [`over` operator](../operators/over.md)
and binds the identifier to the value in the scope in which it appears in addition
to any contained scopes.

A `const` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`const` statements may appear intermixed with `func` and `type` statements.
