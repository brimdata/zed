---
sidebar_position: 4
sidebar_label: Type Statements
---

# Type Statements

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
defined by an [`over` operator](../operators/over.md))
and binds the identifier to the type in the scope in which it appears in addition
to any contained scopes.

A `type` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`type` statements may appear intermixed with `const` and `func` statements.
