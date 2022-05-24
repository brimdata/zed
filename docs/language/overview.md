---
sidebar_position: 1
sidebar_label: Overview
---

# Zed Language Overview

## 1. Introduction

The Zed language is a query language for search, analytics,
and transformation inspired by the
[pipeline pattern](https://en.wikipedia.org/wiki/Tacit_programming)
of the traditional Unix shell.
Like a Unix pipeline, a query is expressed as a data source followed
by a number of commands:
```
command | command | command | ...
```
However, in Zed, the entities that transform data are called
"operators" instead of "commands" and unlike Unix pipelines,
the streams of data in a Zed query
are typed data sequences that adhere to the
[Zed data model](../formats/zed.md).
Moreover, Zed sequences can be forked and joined:
```
operator
| operator
| fork (
  => operator | ...
  => operator | ...
)
| join | ...
```
Here, Zed programs can include multiple data sources and splitting operations
where multiple paths run in parallel and paths can be combined (in an
undefined order), merged (in a defined order) by one or more sort keys,
or joined using relational-style join logic.

Generally speaking, a [flow graph](https://en.wikipedia.org/wiki/Directed_acyclic_graph)
defines a directed acyclic graph (DAG) composed
of data sources and operator nodes.  The Zed syntax leverages "fat arrows",
i.e., `=>`, to indicate the start of a parallel leg of the data flow.

That said, the Zed language is
[declarative](https://en.wikipedia.org/wiki/Declarative_programming)
and the Zed compiler optimizes the data flow computation
&mdash; e.g., often implementing a Zed program differently than
the flow implied by the pipeline yet reaching the same result &mdash;
much as a modern SQL engine optimizes a declarative SQL query.

Zed is also intended to provide a seamless transition from a simple search experience
(e.g., typed into a search bar or as the query argument of the [`zq`](../commands/zq.md) command-line
tool) to more a complex analytics experience composed of complex joins and aggregations
where the Zed language source text would typically be authored in a editor and
managed under source-code control.

Like an email or Web search, a simple keyword search is just the word itself,
e.g.,
```
example.com
```
is a search for the string "example.com" and
```
example.com urgent
```
is a search for values with both the strings "example.com" and "urgent" present.

Unlike typical log search systems, the Zed language operators are uniform:
you can specify an operator including keyword search terms, Boolean predicates,
etc. using the same syntax at any point in the pipeline as
[described below](#7-search-expressions).

For example,
the predicate `message_length > 100` can simply be tacked onto the keyword search
from above, e.g.,
```
example.com urgent message_length > 100
```
finds all values containing the string "example.com" and "urgent" somewhere in them
provided further that the field `message_length` is a numeric value greater than 100.
A related query that performs an aggregation could be more formally
written as follows:
```
search "example.com" AND "urgent"
| where message_length > 100
| summarize kinds:=union(type) by net:=network_of(srcip)
```
which computes an aggregation table of different message types (e.g.,
from a hypothetical field called `type`) into a new, aggregated field
called `kinds` and grouped by the network of all the source IP addresses
in the input
(e.g., from a hypothetical field called `srcip`) as a derived field called `net`.

The short-hand query from above might be typed into a search box while the
latter query might be composed in a query editor or in Zed source files
maintained in GitHub.  Both forms are valid Zed queries.

## 2. The Dataflow Model

In Zed, each operator takes its input from the output of its upstream operator beginning
either with a data source or with an implied source.

All available operators are listed on the [reference page](operators/README.md).

### 2.1 Dataflow Sources

In addition to the data sources specified as files on the `zq` command line,
a source may also be specified with the [`from` operator](operators/from.md).

When running on the command-line, `from` may refer to a file, an HTTP
endpoint, or an S3 URI.  When running in a [Zed lake](../commands/zed.md), `from` typically
refers to a collection of data called a "data pool" and is referenced using
the pool's name much as SQL references database tables by their name.

For more detail, see the reference page of the [`from` operator](operators/from.md),
but as an example, you might use the `get` form of `from` to fetch data from an
HTTP endpoint and process it with Zed, in this case, to extract the description
and license of a GitHub repository:
```
zq -f text "get https://api.github.com/repos/brimdata/zed | yield description,license.name"
```
When a Zed query is run on the command-line with `zq`, the `from` source is
typically omitted and implied instead by the command-line file arguments.
The input may be stdin via `-` as in
```
echo '"hello, world"' | zq  -
```
The examples throughout the language documentation use this "echo pattern"
to standard input of `zq -` to illustrate language semantics.
Note that in these examples, the input values are expressed as Zed values serialized
in the [ZSON text format](../formats/zson.md)
and the `zq` query text expressed as the first argument of the `zq` command
is expressed in the syntax of the Zed language described here.

### 2.2 Dataflow Operators

Each operator is identified by name and performs a specific operation
on a stream of records.

Some operators, like
[`summarize`](operators/summarize.md) or [`sort`](operators/sort.md)
read all of their input before producing output, though
`summarize` can produce incremental results when the group-by key is
aligned with the order of the input.

For large queries that process all of their input, time may pass before
seeing any output.

On the other hand, most operators produce incremental output by operating
on values as they are produced.  For example, a long running query that
produces incremental output will stream results as they are produced, i.e.,
running `zq` to standard output will display results incrementally.

The [`search`](operators/search.md) and [`where`](operators/where.md)
operators "find" values in their input and drop
the ones that do not match what is being looked for.

The [`yield` operator](operators/yield.md) emits one or more output values
for each input value based on arbitrary [expressions](#6-expressions),
providing a convenient means to derive arbitrary output values as a function
of each input value, much like the map concept in the MapReduce framework.

The [`fork` operator](operators/fork.md) copies its input to parallel
legs of a query.  The output of these parallel paths can be combined
in a number of ways:
* merged in sorted order using the [`merge` operator](operators/merge.md),
* joined using the [`join` operator](operators/join.md), or
* combined in an undefined order using the implied [`combine` operator](operators/combine.md).

A path can also be split to multiple query legs using the
[`switch` operator](operators/switch.md), in which data is routed to only one
corresponding leg (or dropped) based on the switch clauses.

Switch operators typically
involve multiline Zed programs, which are easiest to edit in a file.  For example,
suppose this text is in a file called `switch.zed`:
```mdtest-input switch.zed
switch this (
  case 1 => yield {val:this,message:"one"}
  case 2 => yield {val:this,message:"two"}
  default => yield {val:this,message:"many"}
) | merge val
```
Then, running `zq` with `-I switch.zed` like so:
```mdtest-command
echo '1 2 3 4' | zq -z -I switch.zed -
```
produces
```mdtest-output
{val:1,message:"one"}
{val:2,message:"two"}
{val:3,message:"many"}
{val:4,message:"many"}
```
Note that the output order of the switch legs is undefined (indeed they run
in parallel on multiple threads).  To establish a consistent sequence order,
a [`merge` operator](operators/merge.md)
may be applied at the output of the switch specifying a sort key upon which
to order the upstream data.  Often such order does not matter (e.g., when the output
of the switch hits an [aggregator](aggregates/README.md)), in which case it is typically more performant
to omit the merge (though the Zed system will often delete such unnecessary
operations automatically as part optimizing queries when they are compiled).

If no `merge` or `join` is indicated downstream of a `fork` or `switch`,
then the implied `combine` operator is presumed.  In this case, values are
forwarded from the switch to the downstream operator in an undefined order.

### 2.3 The Special Value `this`

In Zed, there are no looping constructs and variables are limited to binding
values between [lateral scopes](#81-lateral-scope) as described below.
Instead, the input sequence
to an operator is produced continuously and any output values are derived
from input values.

In contrast to SQL, where a query may refer to input tables by name,
there are no explicit tables and a Zed operator instead refers
to its input values using the special identifier `this`.

For example, sorting the following input
```mdtest-command
echo '"foo" "bar" "BAZ"' | zq -z sort -
```
produces this case-sensitive output:
```mdtest-output
"BAZ"
"bar"
"foo"
```
But we can make the sort case-insensitive by applying a [function](functions/README.md) to the
input values with the expression `lower(this)`, which converts
each value to lower-case for use in in the sort without actually modifying
the input value, e.g.,
```
echo '"foo" "bar" "BAZ"' | zq -z 'sort lower(this)' -
```
produces
```
"bar"
"BAZ"
"foo"
```

### 2.4 Implied Field References

A common use case for Zed is to process sequences of record-oriented data
(e.g., arising from formats like JSON or Avro) in the form of events
or structured logs.  In this case, the input values to the operators
are Zed [records](../formats/zed.md#21-record) and the fields of a record are referenced with the dot operator.

For example, if the input above were a sequence of records instead of strings
and perhaps contained a second field, e.g.,
```
{s:"foo",x:1}
{s:"bar",x:2}
{s:"BAZ",x:3}
```
Then we could refer to the field `s` using `this.s` and sort the records
as above with `sort this.s`, which would give
```
{s:"BAZ",x:3}
{s:"bar",x:2}
{s:"foo",x:1}
```
This pattern is so common that field references to `this` may be shortened
by simply referring to the field by name wherever a Zed expression is expected,
e.g.,
```
sort s
```
is shorthand for `sort this.s`

### 2.5 Field Assignments

A typical operation in records involves
adding or changing the fields of a record using the [`put` operator](operators/put.md)
or extracting a subset of fields using the [`cut` operator](operators/cut.md).
Also, when aggregating data using group-by keys, the group-by assignments
create new named record fields.

In all of these cases, the Zed language uses the token `:=` to denote
field assignment.  For example,
```
put x:=y+1
```
or
```
summarize salary:=sum(income) by address:=lower(address)
```
This style of "assignment" to a record value is distinguished from the `=`
token which is used in [constant](#3-const-statements) and [variable](#8-lateral-subqueries) assignments.

### 2.6 Implied Operators

When Zed is run in an application like [Brim](https://github.com/brimdata/brim),
queries are often composed interactively in a "search bar" experience.
The language design here attempts to support both this "lean forward" pattern of usage
along with a "coding style" of query writing where the queries might be large
and complex, e.g., to perform transformations in a data pipeline, where
the Zed queries are stored under source-code control perhaps in GitHub or
in Brim's query library.

To facilitate both a programming-like model as well as an ad hoc search
experience, Zed has a canonical, long form that can be abbreviated
using syntax that supports an agile, interactive query workflow.
To this end, Zed allows certain operator names to be optionally omitted when
they can be inferred from context.  For example, the expression following
the [`summarize` operator](operators/summarize.md)
```
summarize count() by id
```
is unambiguously an aggregation and can be shortened to
```
count() by id
```
Likewise, a very common lean-forward use pattern is "searching" so by default,
expressions are interpreted as keyword searches, e.g.,
```
search foo bar or x > 100
```
is abbreviated
```
foo bar or x > 100
```
Furthermore, if an operator-free expression is not valid syntax for
a search expression but is a valid [Zed expression](#6-expressions),
then the abbreviation is treated as having an implied `yield` operator, e.g.,
```
{s:lower(s)}
```
is shorthand for
```
yield {s:lower(s)}
```
When operator names are omitted, `search` has precedence over `yield`, so
```
foo
```
is interpreted as a search for the string "foo" rather than a yield of
the implied record field named `foo`.

Another common query pattern involves adding or mutating fields of records
where the input is presumed to be a sequence of records.
The [`put` operator](operators/put.md) provides this mechanism and the `put`
keyword is implied by the [field assignment](#25-field-assignments) syntax `:=`.

For example, the operation
```
put y:=2*x+1
```
can be expressed simply as
```
y:=2*x+1
```
When composing long-form queries that are shared via Brim or managed in GitHub,
it is best practice to include all operator names in the Zed source text.

In summary, if no operator name is given, the implied operator is determined
from the operator-less source text, in the order given, as follows:
* If the text can be interpreted as a search expression, then the operator is `search`.
* If the text can be interpreted as a boolean expression, then the operator is `where`.
* If the text can be interpreted as one or more field assignments, then the operator is `put`.
* If the text can be interpreted as an aggregation, then the operator is `summarize`.
* If the text can be interpreted as an expression, then the operator is `yield`.
* Otherwise, the text causes a compile-time error.

When in doubt, you can always check what the compiler is doing under the hood
by running `zq` with the `-C` flag to print the parsed query in "canonical form", e.g.,
```mdtest-command
zq -C foo
zq -C 'is(<foo>)'
zq -C 'count()'
zq -C '{a:x+1,b:y-1}'
zq -C 'a:=x+1,b:=y-1'
```
produces
```mdtest-output
search foo
where is(<foo>)
summarize
    count()
yield {a:x+1,b:y-1}
put a:=x+1,b:=y-1
```

## 3. Const Statements

Constants may be defined and assigned to a symbolic name with the syntax
```
const <id> = <expr>
```
where `<id>` is an identifier and `<expr>` is a constant [expression](#6-expressions)
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
(i.e., the main scope at the start of a Zed program or a [lateral scope](#81-lateral-scope)
defined by an [`over` operator](operators/over.md))
and binds the identifier to the value in the scope in which it appears in addition
to any contained scopes.

A `const` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`const` statements may appear intermixed with `type` statements.

## 4. Type Statements

Named types may be created with the syntax
```
type <id> = <type>
```
where `<id>` is an identifier and `<type>` is a [Zed type](#51-first-class-types).
This creates a new type with the given name in the Zed type system, e.g.,
```mdtest-command
echo 80 | zq -z 'type port=uint16 cast(this, <port>)' -
```
produces
```mdtest-output
80(port=uint16)
```

One or more `type` statements may appear at the beginning of a scope
(i.e., the main scope at the start of a Zed program or a [lateral scope](#81-lateral-scope)
defined by an [`over` operator](operators/over.md))
and binds the identifier to the type in the scope in which it appears in addition
to any contained scopes.

A `type` statement cannot redefine an identifier that was previously defined in the same
scope but can override identifiers defined in ancestor scopes.

`type` statements may appear intermixed with `const` statements.

## 5. Data Types

The Zed language includes most data types of a typical programming language
as defined in the [Zed data model](../formats/zed.md).

The syntax of individual literal values generally follows
the [ZSON syntax](../formats/zson.md) with the exception that
[type decorators](../formats/zson.md#22-type-decorators)
are not included in the language.  Instead, a
[type cast](#612-casts) may be used in any expression for explicit
type conversion.

In particular, the syntax of primitive types follows the
[primitive-value definitions](../formats/zson.md#23-primitive-values) in ZSON
as well as the various [complex value definitions](../formats/zson.md#24-complex-values)
like records, arrays, sets, and so forth.  However, complex values are not limited to
constant values like ZSON and can be composed from literal expressions as
[defined below](#611-literals).

### 5.1 First-class Types

Like the Zed data model, the Zed language has first-class types:
any Zed type may be used as a value.

The primitive types are listed in the
[data model specification](../formats/zed.md#1-primitive-types)
and have the same syntax in the Zed language.  Complex types also follow
the ZSON syntax.  Note that the type of a type value is simply `type`.

As in ZSON, _when types are used as values_, e.g., in a Zed expression,
they must be referenced within angle brackets.  That is, the integer type
`int64` is expressed as a type value using the syntax `<int64>`.

Complex types in the Zed language follow the ZSON syntax as well.  Here are
a few examples:
* a simple record type - `{x:int64,y:int64}`
* an array of integers - `[int64]`
* a set of strings - `|[string]|`
* a map of strings keys to integer values - `{[string,int64]}`
* a union of string and integer  - `(string,int64)`

Complex types may be composed, as in `[({s:string},{x:int64})]` which is
an array of type `union` of two types of records.

The [`typeof` function](functions/typeof.md) returns a value's type as
a value, e.g., `typeof(1)` is `<int64>` and `typeof(<int64>)` is `<type>`.

First-class types are quite powerful because types can
serve as group-by keys or be used in ["data shaping"](#9-shaping) logic.
A common workflow for data introspection is to first perform a search of
exploratory data and then count the shapes of each type of data as follows:
```
search ... | count() by typeof(this)
```
For example,
```mdtest-command
echo '1 2 "foo" 10.0.0.1 <string>' | zq -z 'count() by typeof(this) | sort this' -
```
produces
```mdtest-output
{typeof:<int64>,count:2(uint64)}
{typeof:<string>,count:1(uint64)}
{typeof:<ip>,count:1(uint64)}
{typeof:<type>,count:1(uint64)}
```
When running such a query over complex, semi-structured data, the results can
be quite illuminating and can inform the design of "data shaping" Zed queries
to transform raw, messy data into clean data for downstream tooling.

Note the somewhat subtle difference between a record value with a field `t` of
type `type` whose value is type `string`
```
{t:<string>}
```
and a record type used as a value
```
<{t:string}>
```

### 5.2 Named Types

As in any modern programming language, types can be named and the type names
persist into the data model and thus into the serialized input and output.

Named types may be defined in three ways:
* with a [`type` statement as described above](#4-type-statements),
* with a definition inside of another type, or
* by the input data itself.

Type names that are embedded in another type have the form
```
name=type
```
and create a binding between the indicated string `name` and the specified type.
For example,
```
type socket {addr:ip,port:port=uint16}
```
defines a named type `socket` that is a record with field `addr` of type `ip`
and field `port` of type "port", where type "port" is a named type for type `uint16` .

Named types may also be defined by the input data itself, as Zed data is
comprehensively self describing.
When named types are defined in the input data, there is no need to declare their
type in a query.
In this case, a Zed expression may refer to the type by the name that simply
appears to the runtime as a side effect of operating upon the data.  If the type
name referred to in this way does not exist, then the type value reference
results in `error("missing")`.  For example,
```mdtest-command
echo '1(=foo) 2(=bar) 3(=foo)' | zq -z 'typeof(this)==<foo>' -
```
results in
```mdtest-output
1(=foo)
3(=foo)
```
and
```mdtest-command
echo '1(=foo)' | zq -z 'yield <foo>' -
```
results in
```mdtest-output
<foo=int64>
```
but
```mdtest-command
zq -z 'yield <foo>'
```
gives
```mdtest-output
error("missing")
```
Each instance of a named type definition overrides any earlier definition.
In this way, types are local in scope.

Each value that references a named type retains its local definition of the
named type retaining the proper type binding while accommodating changes in a
particular named type.  For example,
```mdtest-command
echo '1(=foo) 2(=bar) "hello"(=foo) 3(=foo)' | zq -z 'count() by typeof(this) | sort this' -
```
results in
```mdtest-output
{typeof:<bar=int64>,count:1(uint64)}
{typeof:<foo=int64>,count:2(uint64)}
{typeof:<foo=string>,count:1(uint64)}
```
Here, the two versions of type "foo" are retained in the group-by results.

In general, it is bad practice to define multiple versions of a single named type,
though the Zed system and Zed data model accommodate such dynamic bindings.
Managing and enforcing the relationship between type names and their type definitions
on a global basis (e.g., across many different data pools in a Zed lake) is outside
the scope of the Zed data model and language.  That said, Zed provides flexible
building blocks so systems can define their own schema versioning and schema
management policies on top of these Zed primitives.

Zed's [super-structured data model](../formats/README.md#2-zed-a-super-structured-pattern)
is a superset of relational tables and
the Zed language's type system can easily make this connection.
As an example, consider this type definition for "employee":
```
type employee {id:int64,first:string,last:string,job:string,salary:float64}
```
In SQL, you might find the top five salaries by last name with
```
SELECT last,salary
FROM employee
ORDER BY salary
LIMIT 5
```
In Zed, you would say
```
from anywhere | typeof(this)==<employee> | cut last,salary | sort salary | head 5
```
and since type comparisons are so useful and common, the [`is` function](functions/is.md)
can be used to perform the type match:
```
from anywhere | is(<employee>) | cut last,salary | sort salary | head 5
```
The power of Zed is that you can interpret data on the fly as belonging to
a certain schema, in this case "employee", and those records can be intermixed
with other relevant data.  There is no need to create a table called "employee"
and put the data into the table before that data can be queried as an "employee".
And if the schema or type name for "employee" changes, queries still continue
to work.

### 5.3 First-class Errors

As with types, errors in Zed are first-class: any value can be transformed
into an error by wrapping it in the Zed [`error` type](../formats/zed.md#27-error).

In general, expressions and functions that result in errors simply return
a value of type `error` as a result.  This encourages a powerful flow-style
of error handling where errors simply propagate from one operation to the
next and land in the output alongside non-error values to provide a very helpful
context and rich information for tracking down the source of errors.  There is
no need to check for error conditions everywhere or look through auxiliary
logs to find out what happened.

For example,
input values can be transformed to errors as follows:
```mdtest-command
echo '0 "foo" 10.0.0.1' | zq -z 'error(this)' -
```
produces
```mdtest-output
error(0)
error("foo")
error(10.0.0.1)
```
More practically, errors from the runtime show up as error values.
For example,
```mdtest-command
echo 0 | zq -z '1/this' -
```
produces
```mdtest-output
error("divide by zero")
```
And since errors are first-class and just values, they have a type.
In particular, they are a complex type where the error value's type is the
complex type `error` containing the type of the value.  For example,
```mdtest-command
echo 0 | zq -z 'typeof(1/this)' -
```
produces
```mdtest-output
<error(string)>
```
First-class errors are particularly useful for creating structured errors.
When a Zed query encounters a problematic condition,
instead of silently dropping the problematic error
and logging an error obscurely into some hard-to-find system log as so many
ETL pipelines do, the Zed logic can
preferably wrap the offending value as an error and propagate it to its output.

For example, suppose a bad value shows up:
```
{kind:"bad", stuff:{foo:1,bar:2}}
```
A Zed [shaper](#9-shaping) could catch the bad value (e.g., as a default
case in a [`switch`](operators/switch.md) topology) and propagate it as
an error using the Zed expression:
```
yield error({message:"unrecognized input",input:this})
```
then such errors could be detected and searched for downstream with the
[`is_error` function](functions/is_error.md).
For example,
```
is_error(this)
```
on the wrapped error from above produces
```
error({message:"unrecognized input",input:{kind:"bad", stuff:{foo:1,bar:2}}})
```
There is no need to create special tables in a complex warehouse-style ETL
to land such errors as they can simply land next to the output values themselves.

And when transformations cascade one into the next as different stages of
an ETL pipeline, errors can be wrapped one by one forming a "stack trace"
or lineage of where the error started and what stages it traversed before
landing at the final output stage.

Errors will unfortunately and inevitably occur even in production,
but having a first-class data type to manage them all while allowing them to
peacefully coexist with valid production data is a novel and
useful approach that Zed enables.

#### 5.3.1 Missing and Quiet

Zed's heterogeneous data model allows for queries
that operate over different types of data whose structure and type
may not be known ahead of time, e.g., different
types of records with different field names and varying structure.
Thus, a reference to a field, e.g., `this.x` may be valid for some values
that include a field called `x` but not valid for those that do not.

What is the value of `x` when the field `x` does not exist?

A similar question faced SQL when it was adapted in various different forms
to operate on semi-structured data like JSON or XML.  SQL already had the `NULL` value
so perhaps a reference to a missing value could simply be `NULL`.

But JSON also has `null`, so a reference to `x` in the JSON value
```
{"x":null}
```
and a reference to `x` in the JSON value
```
{}
```
would have the same value of `NULL`.  Furthermore, an expression like `x==NULL`
could not differentiate between these two cases.

To solve this problem, the `MISSING` value was proposed to represent the value that
results from accessing a field that is not present.  Thus, `x==NULL` and
`x==MISSING` could disambiguate the two cases above.

Zed, instead, recognizes that the SQL value is `MISSING` is a paradox:
I'm here but I'm not.  

In reality, a `MISSING` value is not a value.  It's an error condition
that resulted from trying to reference something that didn't exist.

So why should we pretend that this is a bona fide value?  SQL adopted this
approach because it lacks first-class errors.

But Zed has first-class errors so
a reference to something that does not exist is an error of type
`error<string>` whose value is `error("missing")`.  For example,
```mdtest-command
echo "{x:1} {y:2}" | zq -z 'yield x' -
```
produces
```mdtest-output
1
error("missing")
```
Sometimes you want missing errors to show up and sometimes you don't.
The [`quiet` function](functions/quiet.md) transforms missing errors into
"quiet errors".  A quiet error is the value `error("quiet")` and is ignored
by most operators, in particular `yield`.  For example,
```mdtest-command
echo "{x:1} {y:2}" | zq -z "yield quiet(x)" -
```
produces
```mdtest-output
1
```

## 6. Expressions

Zed expressions follow the typical patterns in programming languages.
Expressions are typically used within data flow operators
to perform computations on input values and are typically evaluated once per each
input value [`this`](#23-the-special-value-this).

For example, `yield`, `where`, `cut`, `put`, `sort` and so forth all take
various expressions as part of their operation.

### 6.1 Arithmetic

Arithmetic operations (`*`, `/`, `%`, `+`, `-`) follow customary syntax
and semantics and are left-associative with multiplication and division having
precedence over addition and subtraction.  `%` is the modulo operator.

For example,
```mdtest-command
zq -z 'yield 2*3+1, 11%5, 1/0, "foo"+"bar"'
```
produces
```mdtest-output
7
1
error("divide by zero")
"foobar"
```

### 6.2 Comparisons

Comparison operations (`<`, `<=`, `==`, `!=`, `>`, `>=`) follow customary syntax
and semantics and result in a truth value of type `bool` or an error.
A comparison expression is any valid Zed expression compared to any other
valid Zed expression using a comparison operator.

When the operands are coercible to like types, the result is the truth value
of the comparison.  Otherwise, the result is `false`.

If either operand to a comparison
is `error("missing")`, then the result is `error("missing")`.

For example,
```mdtest-command
zq -z 'yield 1 > 2, 1 < 2, "b" > "a", 1 > "a", 1 > x'

```
produces
```mdtest-output
false
true
true
false
error("missing")
```

### 6.3 Containment

The `in` operator has the form
```
<item-expr> in <container-expr>
```
and is true if the `<item-expr>` expression results in a value that
appears somewhere in the `<container-expr>` as an exact match of the item.
The right-hand side value can be any Zed value and complex values are
recursively traversed to determine if the item is present anywhere within them.

For example,
```mdtest-command
echo '{a:[1,2]}{b:{c:3}}{d:{e:1}}' | zq -z '1 in this' -
```
produces
```mdtest-output
{a:[1,2]}
{d:{e:1}}
```
You can also use this operator with a static array:
```mdtest-command
echo '{accounts:[{id:1},{id:2},{id:3}]}' | zq -z 'over accounts | where id in [1,2]' -
```
produces
```mdtest-output
{id:1}
{id:2}
```

### 6.4 Logic

The keywords `and`, `or`, and `not` perform logic on operands of type `bool`.
The binary operators `and` and `or` operate on Boolean values and result in
an error value if either operand is not a Boolean.  Likewise, `not` operates
on its unary operand and results in an error if its operand is not type `bool`.
Unlike many other languages, non-Boolean values are not automatically converted to
Boolean type using "truthiness" heuristics.

### 6.5 Field Dereference

Record fields are dereferenced with the dot operator `.` as is customary
in other languages and have the form
```
<value> . <id>
```
where `<id>` is an identifier representing the field name referenced.
If a field name is not representable as an identifier, then [indexing](#66-indexing)
may be used with a quoted string to represent any valid field name.
Such field names can be accessed using [`this`](#23-the-special-value-this) and an array-style
reference, e.g., `this["field with spaces"]`.

If the dot operator is applied to a value that is not a record
or if the record does not have the given field, then the result is
`error("missing")`.

### 6.6 Indexing

The index operation can be applied to various data types and has the form:
```
<value> [ <index> ]
```
If the `<value>` expression is a record, then the `<index>` operand
must be coercible to a string and the result is the record's field
of that name.

If the `<value>` expression is an array, then the `<index>` operand
must be coercible to an integer and the result is the
value in the array of that index.

If the `<value>` expression is a set, then the `<index>` operand
must be coercible to an integer and the result is the
value in the set of that index ordered by total order of Zed values.

If the `<value>` expression is a map, then the `<index>` operand
is presumed to be a key and the corresponding value for that key is
the result of the operation.  If no such key exists in the map, then
the result is `error("missing")`.

If the `<value>` expression is a string, then the `<index>` operand
must be coercible to an integer and the result is an integer representing
the unicode code point at that offset in the string.

If the `<value>` expression is type `bytes`, then the `<index>` operand
must be coercible to an integer and the result is an unsigned 8-bit integer
representing the byte value at that offset in the bytes sequence.

### 6.7 Slices

The slice operation can be applied to various data types and has the form:
```
<value> [ <from> : <to> ]
```
The `<from>` and `<to>` terms must be expressions that are coercible
to integers and represent a range of index values to form a subset of elements
from the `<value>` term provided.  The range begins at the `<from>` position
and ends one before the `<to>` position.  A negative
value of `<from>` or `<to>` represents a position relative to the
end of the value being sliced.

If the `<value>` expression is an array, then the result is an array of
elements comprising the indicated range.

If the `<value>` expression is a set, then the result is a set of
elements comprising the indicated range ordered by total order of Zed values.

If the `<value>` expression is a string, then the result is a substring
consisting of unicode code points comprising the given range.

If the `<value>` expression is type `bytes`, then the result is a bytes sequence
consisting of bytes comprising the given range.

### 6.8 Conditional

A conditional expression has the form
```
<boolean> ? <expr> : <expr>
```
The `<boolean>` expression is evaluated and must have a result of type `bool`.
If not, an error results.

If the result is true, then the first `<expr>` expression is evaluated and becomes
the result.  Otherwise, the second `<expr>` expression is evaluated and
becomes the result.

For example,
```mdtest-command
echo '{s:"foo",v:1}{s:"bar",v:2}' | zq -z 'yield (s=="foo") ? v : -v' -
```
produces
```mdtest-output
1
-2
```

Note that if the expression has side effects,
as with [aggregate function calls](#610-aggregate-function-calls), only the selected expression
will be evaluated.

For example,
```mdtest-command
echo '"foo" "bar" "foo"' | zq -z 'yield this=="foo" ? {foocount:count()} : {barcount:count()}' -
```
produces
```mdtest-output
{foocount:1(uint64)}
{barcount:1(uint64)}
{foocount:2(uint64)}
```

### 6.9 Function Calls

[Functions](functions/README.md) perform stateless transformations of their input value to their
return value and utilize call-by value semantics with positional and unnamed
arguments.  Some functions take a variable number of arguments.

> The only functions currently available are built-in, but user-defined functions and
> library package management will be added to the Zed language soon.

For example,
```mdtest-command
zq -z 'yield pow(2,3), lower("ABC")+upper("def"), typeof(1)'
```
produces
```mdtest-output
8.
"abcDEF"
<int64>
```

### 6.10 Aggregate Function Calls

[Aggregate functions](aggregates/README.md) may be called within an expression.
Unlike the aggregation context provided by a [summarizing group-by](operators/summarize.md), such calls
in expression context yield an output value for each input value.

Note that because aggregate functions carry state which is typically
dependent on the order of input values, their use can prevent the runtime
optimizer from parallelizing a query.

That said, aggregate function calls can be quite useful in a number of contexts.
For example, a unique ID can be assigned to the input quite easily:
```mdtest-command
echo '"foo" "bar" "baz"' | zq -z 'yield {id:count(),value:this}' -
```
produces
```mdtest-output
{id:1(uint64),value:"foo"}
{id:2(uint64),value:"bar"}
{id:3(uint64),value:"baz"}
```
In contrast, calling aggregate functions with the [`summarize` operator](operators/summarize.md)
```mdtest-command
echo '"foo" "bar" "baz"' | zq -z 'summarize count(),union(this)' -
```
produces just one output value
```mdtest-output
{count:3(uint64),union:|["bar","baz","foo"]|}
```

### 6.11 Literals

Any of the [data types listed above](#5-data-types) may be used in expressions
as long as it is compatible with the semantics of the expression.

String literals are enclosed in either single quotes or double quotes and
must conform to UTF-8 encoding and follow the JavaScript escaping
conventions and unicode escape syntax.  Also, if the sequence `${` appears
in a string the `$` character must be escaped, i.e., `\$`.

#### 6.11.1 String Interpolation

Strings may include interpolation expressions, which has the form
```
${ <expr> }
```
In this case, the characters starting with `$` and ending at `}` are substituted
with the result of evaluating the expression `<expr>`.  If this result is not
a string, it is implicitly cast to a string.

For example,
```mdtest-command
echo '{numerator:22.0, denominator:7.0}' | zq -z 'yield "approximate pi = ${numerator / denominator}"' -
```

produces
```mdtest-output
"approximate pi = 3.142857142857143"
```

If any template expression results in an error, then the value of the template
literal is the first error encountered in left-to-right order.

> TBD: we could improve an error result here by creating a structured error
> containing the string template text along with a list of values/errors of
> the expressions.

String interpolation may be nested, where `<expr>` contains additional strings
with interpolated expressions.

For example,
```mdtest-command
echo '{foo:"hello", bar:"world", HELLOWORLD:"hi!"}' | zq -z 'yield "oh ${this[upper("${foo + bar}")]}"' -
```

produces
```mdtest-output
"oh hi!"
```

#### 6.11.2 Record Expressions

Record literals have the form
```
{ <spec>, <spec>, ... }
```
where a `<spec>` has one of three forms:
```
<field> : <expr>
<field>
...<expr>
```
The first form is a customary colon-separated field and value similar to JavaScript,
where `<field>` may be an identifier or quoted string.
The second form is an [implied field reference](#24-implied-field-references)
`<ref>`, which is shorthand for `<field>:<ref>`.  The third form is the `...`
spread operator which expects a record value as the result of `<expr>` and
inserts all of the fields from the resulting record.
If a spread expression results in a non-record type (e.g., errors), then that
part of the record is simply elided.

The fields of a record expression are evaluated left to right and when
field names collide the rightmost instance of the name determines that
field's value.

For example,
```mdtest-command
echo '{x:1,y:2,r:{a:1,b:2}}' | zq -z 'yield {a:0},{x}, {...r}, {a:0,...r,b:3}' -
```
produces
```mdtest-output
{a:0}
{x:1}
{a:1,b:2}
{a:1,b:3}
```

#### 6.11.3 Array Expressions

Array literals have the form
```
[ <expr>, <expr>, ... ]
```
When the expressions result in values of non-uniform type, then the implied
type of the array is an array of type `union` of the types that appear.

For example,
```mdtest-command
zq -z 'yield [1,2,3],["hello","world"]'
```
produces
```mdtest-output
[1,2,3]
["hello","world"]
```

#### 6.11.4 Set Expressions

Set literals have the form
```
|[ <expr>, <expr>, ... ]|
```
When the expressions result in values of non-uniform type, then the implied
type of the set is a set of type `union` of the types that appear.

Set values are always organized in their "natural order" independent of the order
they appear in the set literal.

For example,
```mdtest-command
zq -z 'yield |[3,1,2]|,|["hello","world","hello"]|'
```
produces
```mdtest-output
|[1,2,3]|
|["hello","world"]|
```

#### 6.11.5 Map Expressions

Map literals have the form
```
|{ <expr>:<expr>, <expr>:<expr>, ... }|
```
where the first expression of each colon-separated entry is the key value
and the second expression is the value.
When the key and/or value expressions result in values of non-uniform type,
then the implied type of the map has a key type and/or value type that is
a union of the types that appear in each respective category.

For example,
```mdtest-command
zq -z 'yield |{"foo":1,"bar"+"baz":2+3}|'
```
produces
```mdtest-output
|{"foo":1,"barbaz":5}|
```

#### 6.11.6 Union Values

A union value can be created with a [cast](#612-casts).  For example, a union of types `int64`
and `string` is expressed as `(int64,string)` and any value that has a type
that appears in the union type may be cast to that union type.
Since 1 is an `int64` and "foo" is a `string`, they both can be
values of type `(int64,string)`, e.g.,
```mdtest-command
echo '1 "foo"' | zq -z 'yield cast(this,<(int64,string)>)' -
```
produces
```mdtest-output
1((int64,string))
"foo"((int64,string))
```
The value underlying a union-tagged value is accessed with the
[`under` function](functions/under.md):
```mdtest-command
echo '1((int64,string))' | zq -z 'yield under(this)' -
```
produces
```mdtest-output
1
```
Union values are powerful because they provide a mechanism to precisely
describe the type of any nested, semi-structured value composed of elements
of different types.  For example, the type of the value `[1,"foo"]` in JavaScript
is simply a generic JavaScript "object".  But in Zed, the type of this
value is an array of union of string and integer, e.g.,
```mdtest-command
echo '[1,"foo"]' | zq -z 'typeof(this)' -
```
produces
```mdtest-output
<[(int64,string)]>
```

### 6.12 Casts

Type conversion is performed with casts and the built-in [`cast` function](functions/cast.md).

Casts for primitive types have a function-style syntax of the form
```
<type> ( <expr> )
```
where `<type>` is a [Zed type](#51-first-class-types) and `<expr>` is any Zed expression.
In the case of primitive types, the type-value angle brackets
may be omitted, e.g., `<string>(1)` is equivalent to `string(1)`.
If the result of `<expr>` cannot be converted
to the indicated type, then the cast's result is an error value.

For example,
```mdtest-command
echo '1 200 "123" "200"' | zq -z 'yield int8(this)' -
```
produces
```mdtest-output
1(int8)
error("cannot cast 200 to type int8")
123(int8)
error("cannot cast \"200\" to type int8")
```

Casting attempts to be fairly liberal in conversions.  For example, values
of type `time` can be created from a diverse set of date/time input strings
based on the [Go Date Parser library](https://github.com/araddon/dateparse).

```mdtest-command
echo '"May 8, 2009 5:57:51 PM" "oct 7, 1970"' | zq -z 'yield time(this)' -
```
produces
```mdtest-output
2009-05-08T17:57:51Z
1970-10-07T00:00:00Z
```

Casts of complex or [named types](#52-named-types) may be performed using type values
either in functional form or with `cast`:
```
<type-value> ( <expr> )
cast(<expr>, <type-value>)
```
For example
```mdtest-command
echo '80 8080' | zq -z 'type port = uint16 yield <port>(this)' -
```
produces
```mdtest-output
80(port=uint16)
8080(port=uint16)
```

Casts may be used with complex types as well.  As long as the target type can
accommodate the value, the case will be recursively applied to the components
of a nested value.  For example,
```mdtest-command
echo '["10.0.0.1","10.0.0.2"]' | zq -z 'cast(this,<[ip]>)' -
```
produces
```mdtest-output
[10.0.0.1,10.0.0.2]
```
and
```mdtest-command
echo '{ts:"1/1/2022",r:{x:"1",y:"2"}} {ts:"1/2/2022",r:{x:3,y:4}}' | zq -z 'cast(this,<{ts:time,r:{x:float64,y:float64}}>)' -
```
produces
```mdtest-output
{ts:2022-01-01T00:00:00Z,r:{x:1.,y:2.}}
{ts:2022-01-02T00:00:00Z,r:{x:3.,y:4.}}
```

## 7. Search Expressions

Search expressions provide a hybrid syntax between keyword search
and boolean expressions.  In this way, a search is a shorthand for
a "lean forward" style activity where one is interactively exploring
data with ad hoc searches.  All shorthand searches have a corresponding
long form built from the expression syntax above in combination with the
[search term syntax](#721-search-terms) described below.

### 7.1 Search Patterns

Several styles of string search can be performed with a search expression
(as well as the [`grep` function](functions/grep.md)) using "patterns",
where a pattern is a regular expression, glob, or simple string.

#### 7.1.1 Regular Expressions

A regular expression is specified in the familiar slash syntax where the
expression begins with a `/` character and ends with a terminating `/` character.
The string between the slashes (exclusive of those characters) is the
regular expression.

The format of Zed regular expressions follows the syntax of the
[RE2 regular expression library](https://github.com/google/re2)
and is documented in the
[RE2 Wiki](https://github.com/google/re2/wiki/Syntax).

Regular expressions may be used freely in search expressions, e.g.,
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {foo:1}' | zq -z '/(foo|bar)/' -
```
produces
```mdtest-output
"foo"
{s:"bar"}
{foo:1}
```
Regular expressions may also appear in the `grep` function:
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {foo:1}' | zq -z 'yield grep(/ba.*/, s)' -
```
produces
```mdtest-output
false
true
true
false
```

#### 7.1.2 Globs

Globs provide a convenient short-hand for regular expressions and follow
the familiar pattern of "file globbing" supported by Unix shells.
Zed globs are a simple, special case that utilize only the `*` wildcard.

Valid glob characters include `a` through `z`, `A` through `Z`,
any valid string escape sequence
(along with escapes for `*`, `=`, `+`, `-`), and the unescaped characters:
```
_ . : / % # @ ~
```
A glob must begin with one of these characters or `*` then may be
followed by any of these characters, `*`, or digits `0` through `9`.

> Note that these rules do not allow for a leading digit.

For example, a prefix match is easily accomplished via `prefix*`, e.g.,
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {foo:1}' | zq -z 'b*' -
```
produces
```mdtest-output
{s:"bar"}
{s:"baz"}
```
Likewise, a suffix match may be performed as follows:
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {foo:1}' | zq -z '*z' -
```
produces
```mdtest-output
{s:"baz"}
```
and
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {a:1}' | zq -z '*a*' -
```
produces
```mdtest-output
{s:"bar"}
{s:"baz"}
{a:1}
```

Globs may also appear in the `grep` function:
```mdtest-command
echo '"foo" {s:"bar"} {s:"baz"} {foo:1}' | zq -z 'yield grep(ba*, s)' -
```
produces
```mdtest-output
false
true
true
false
```

Note that a glob may look like multiplication but context disambiguates
these conditions, e.g.,
```
a*b
```
is a glob match for any matching string value in the input, but
```
a*b==c
```
is a Boolean comparison between the product `a*b` and `c`.

### 7.2 Search Logic

The search patterns described above can be combined with other elements
of a search expression comprised of "search terms" that may be combined
using Boolean logic.

> Note that when processing [ZNG](../formats/zng.md) data, the Zed runtime performs a multi-threaded
> Boyer-Moore scan over decompressed data buffers before parsing any data.
> This allows large buffers of data to be efficiently discarded and skipped when
> searching for rarely occurring values.  For a [Zed lake](../lake/format.md), search indexes
> may also be configured to further accelerate searches.
> In a forthcoming release, Zed will also offer an approach for locating
> delimited words within string fields, which will allow accelerated
> search using a full-text search index.  Currently, search indexes may be built
> for exact value match as text segmentation is in the works.

#### 7.2.1 Search Terms

A "search term" is one of the following;
* a regular expression as described above,
* a glob as described above,
* a keyword,
* any literal of a primitive type, or
* expression predicates.

##### 7.2.1.1 Regular Expression Search Term

A regular expression `/re/` is equivalent to
```
grep(/re/, this)
```
but shorter and easier to type in a search expression.

For example,
```
/(foo|bar.*baz.*\.com)/
```
Searches for any string that begins with `foo` or `bar` has the string
`baz` in it and ends with `.com`.

##### 7.2.1.2 Glob Search Term

A glob search term `<glob>` is equivalent to
```
grep(<glob>, this)
```
but shorter and easier to type in a search expression.

For example,
```
foo*baz*.com
```
Searches for any string that begins with `foo` has the string
`baz` in it and ends with `.com`.

##### 7.2.1.3 Keyword Search Term

Keywords and string literals are equivalent search terms so it is often
easier to quote a string search term instead of using escapes in a keyword.
Keywords are useful in interactive contexts where searches can be issued
and modified quickly without having to type matching quotes.

Keyword search has the look and feel of Web search or email search.

Valid keyword characters include `a` through `z`, `A` through `Z`,
any valid string escape sequence
(along with escapes for `*`, `=`, `+`, `-`), and the unescaped characters:
```
_ . : / % # @ ~
```
A keyword must begin with one of these characters then may be
followed by any of these characters or digits `0` through `9`.

A keyword search is equivalent to
```
grep(<keyword>, this)
```
where `<keyword>` is the quoted string-literal of the unquoted string.
For example,
```
search foo
```
is equivalent to
```
where grep("foo", this)
```

Note that the "search" keyword may be omitted.
For example, the simplest Zed program is perhaps a single keyword search, e.g.,
```
foo
```
As above, this program searches the implied input for values that
contain the string "foo".

##### 7.2.1.4 String Literal Search Term

A string literal as a search term is simply a search for that string and is
equivalent to
```
grep(<string>, this)
```
For example,
```
search "foo"
```
is equivalent to
```
where grep("foo", this)
```

> Note that this equivalency between keyword search terms and grep semantics
> will change in the near future when we add support for full-text search.
> In this case, grep will still support substring match but keyword search
> will match segmented words from string fields so that they can be efficiently
> queried in search indexes.

##### 7.2.1.5 Non-String Literal Search Term

Search terms representing non-string Zed values search for both an exact
match for the given value as well as a string search for the term exactly
as it appears as typed.  Such values include:
* integers,
* floating point numbers,
* time values,
* durations,
* IP addresses,
* networks,
* bytes values, and
* type values.

A search for a Zed value `<value>` represented as the string `<string>` is
equivalent to
```
<value> in this or grep(<string>, this)
```
For example,
```
search 123 and 10.0.0.1
```
which can be abbreviated
```
123 10.0.0.1
```
is equivalent to
```
where (123 in this or grep("123", this)) and (10.0.0.1 in this or grep("10.0.0.1", this))
```

Complex values are not supported as search terms but may be queried with
the "in" operator, e.g.,
```
{s:"foo"} in this
```

##### 7.2.1.6 Predicate Search Term

Any Boolean-valued [function](functions/README.md) like `is`, `has`,
`grep` etc. and any [comparison expression](#62-comparisons)
may be used as a search term and mixed into a search expression.

For example,
```
is(<foo>) has(bar) baz x==y+z timestamp > 2018-03-24T17:17:55Z
```
is a valid search expression but
```
/foo.*/ x+1
```
is not.

#### 7.3 Boolean Logic

Search terms may be combined into boolean expressions using logical operators
`and`, `or` and `not`.  `and` may be elided; i.e., concatenation of search terms
is a logical `and`.  `not` has highest precedence and `and` has precedence over
`or`.  Parentheses may be used to override natural precedence.

Note that the concatenation form of `and` is not valid in standard expressions and
is available only in search expressions.
Concatenation is convenient in interactive sessions but it is best practice to
explicitly include the `and` operator when editing Zed source files.

For example,
```
not foo bar or baz
```
means
```
((not grep("foo")) and grep("bar)) or grep("baz")
```
while
```
foo (bar or baz)
```
means
```
grep("foo") and (grep("bar)) or grep("baz"))
```

## 8. Lateral Subqueries

Lateral subqueries provide a powerful means to apply a Zed query
to each subsequence of values generated from an outer sequence of values.
The inner query may be _any Zed query_ and may refer to values from
the outer sequence.

Lateral subqueries are created using the scoped form of the
[`over` operator](operators/over.md) and may be nested to arbitrary depth.

For example,
```mdtest-command
echo '{s:"foo",a:[1,2]} {s:"bar",a:[3]}' | zq -z 'over a with name=s => (yield {name,elem:this})' -
```
produces
```mdtest-output
{name:"foo",elem:1}
{name:"foo",elem:2}
{name:"bar",elem:3}
```
Here the lateral scope, described below, creates a subquery
```
yield {name,elem:this}
```
for each subsequence of values derived from each outer input value.
In the example above, there are two input values:
```
{s:"foo",a:[1,2]}
{s:"bar",a:[3]}
```
which imply two subqueries derived from the `over` operator traversing `a`.
The first subquery thus operates on the input values `1, 2` with the variable
`name` set to "foo" assigning `1` and then `2` to `this`, thereby emitting
```
{name:"foo",elem:1}
{name:"foo",elem:2}
```
and the second subquery operators on the input value `3` with the variable
`name` set to "bar", emitting
```
{name:"bar",elem:3}
```

You can also import a parent-scope field reference into the inner scope by
simply referring to its name without assignment, e.g.,
```mdtest-command
echo '{s:"foo",a:[1,2]} {s:"bar",a:[3]}' | zq -z 'over a with s => (yield {s,elem:this})' -
```
produces
```mdtest-output
{s:"foo",elem:1}
{s:"foo",elem:2}
{s:"bar",elem:3}
```

### 8.1 Lateral Scope

A lateral scope has the form `=> ( <query> )` and currently appears
only the context of an [`over` operator](operators/over.md),
as illustrated above, and has the form:
```
over ... with <elem> [, <elem> ...] => ( <query> )
```
where `<elem>` has either an assignment form
```
<var>=<expr>
```
or a field reference form
```
<field>
```
For each input value to the outer scope, the assignment form creates a binding
between each `<expr>` evaluated in the outer scope and each `<var>`, which
represents a new symbol in the inner scope of the `<query>`.
In the field reference form, a single identifier `<field>` refers to a field
in the parent scope and makes that field's value available in the lateral scope
with the same name.

The `<query>`, which may be any Zed query, is evaluated once per outer value
on the sequence generated by the `over` expression.  In the lateral scope,
the value `this` refers to the inner sequence generated from the `over` expressions.
This query runs to completion for each inner sequence and emits
each subquery result as each inner sequence traversal completes.

This structure is powerful because _any_ Zed query can appear in the body of
the lateral scope.  In contrast to the `yield` example, a sort could be
applied to each subsequence in the subquery, where sort
reads all of the values of the subsequence, sorts them, emits them, then
repeats the process for the next subsequence.  For example,
```mdtest-command
echo '[3,2,1] [4,1,7] [1,2,3]' | zq -z 'over this => (sort this | collect(this))' -
```
produces
```mdtest-output
{collect:[1,2,3]}
{collect:[1,4,7]}
{collect:[1,2,3]}
```

### 8.2 Lateral Expressions

Lateral subqueries can also appear in expression context using the
parenthesized form:
```
( over <expr> [, <expr>...] [with <var>=<expr> [, ... <var>[=<expr>]] | <lateral> )
```
> Note that the parentheses disambiguate a lateral expression from a lateral
> dataflow operator.

This form must always include a lateral scope as indicated by `<lateral>`,
which can be any dataflow operator sequence excluding [`from` operators](operators/from.md).
As with the `over` operator, values from the outer scope can be brought into
the lateral scope using the `with` clause.

The lateral expression is evaluated by evaluating each `<expr>` and feeding
the results as inputs to the `<lateral>` dataflow operators.  Each time the
lateral expression is evaluated, the lateral operators are run to completion,
e.g.,
```mdtest-command
echo '[3,2,1] [4,1,7] [1,2,3]' | zq -z 'yield (over this | sum(this))' -
```
produces
```mdtest-output
{sum:6}
{sum:12}
{sum:6}
```
This structure generalizes to any more complicated expression context,
e.g., we can embed multiple lateral expressions inside of a record literal
and use the spread operator to tighten up the output:
```mdtest-command
echo '[3,2,1] [4,1,7] [1,2,3]' | zq -z '{...(over this | sort this | sorted:=collect(this)),...(over this | sum(this))}' -
```
produces
```mdtest-output
{sorted:[1,2,3],sum:6}
{sorted:[1,4,7],sum:12}
{sorted:[1,2,3],sum:6}
```

## 9. Shaping

Data that originates from heterogeneous sources typically has
inconsistent structure and is thus difficult to reason about or query.
To unify disparate data sources, data is often cleaned up to fit into
a well-defined set of schemas, which combines the data into a unified
store like a data warehouse.

In Zed, this cleansing process is called "shaping" the data, and Zed leverages
its rich, [super-structured](../formats/README.md#2-zed-a-super-structured-pattern)
type system to perform core aspects of data transformation.
In a data model with nesting and multiple scalar types (such as Zed or JSON),
shaping includes converting the type of leaf fields, adding or removing fields
to "fit" a given shape, and reordering fields.

While shaping remains an active area of development, the core functions in Zed
that currently perform shaping are:

* [`cast`](functions/cast.md) - coerce a value to a different type
* [`crop`](functions/crop.md) - remove fields from a value that are missing in a specified type
* [`fill`](functions/fill.md) - add null values for missing fields
* [`order`](functions/order.md) - reorder record fields
* [`shape`](functions/shape.md) - apply `cast`, `fill`, and `order`

They all have the same signature, taking two parameters: the value to be
transformed and a type value for the target type.

> Another type of transformation that's needed for shaping is renaming fields,
> which is supported by the [`rename` operator](operators/rename.md).
> Also, the [`yield` operator](operators/yield.md)
> is handy for simply emitting new, arbitrary record literals based on
> input values and mixing in these shaping functions in an embedded record literal.
> The [`fuse` aggregate function](aggregates/fuse.md) is also useful for fusing
> values into a common schema, though a type is returned rather than values.

In the examples below, we will use the following named type `connection`
that is stored in a file `connection.zed`
and is included in the example Zed queries with the `-I` option of `zq`:
```mdtest-input connection.zed
type socket = { addr:ip, port:port=uint16 }
type connection = {
    kind: string,
    client: socket,
    server: socket,
    vlan: uint16
}
```
We also use this sample JSON input in a file called `sample.json`:
```mdtest-input sample.json
{
  "kind": "dns",
  "server": {
    "addr": "10.0.0.100",
    "port": 53
  },
  "client": {
    "addr": "10.47.1.100",
    "port": 41772
  },
  "uid": "C2zK5f13SbCtKcyiW5"
}
```

### 9.1 Cast

The `cast` function applies a cast operation to each leaf value that matches the
field path in the specified type, e.g.,
```mdtest-command
zq -Z -I connection.zed "cast(this, <connection>)" sample.json
```
casts the address fields to type `ip`, the port fields to type `port`
(which is a [named type](#52-named-types) for type `uint16`) and the address port pairs to
type `socket` without modifying the `uid` field or changing the
order of the `server` and `client` fields:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: 10.0.0.100,
        port: 53 (port=uint16)
    } (=socket),
    client: {
        addr: 10.47.1.100,
        port: 41772
    } (socket),
    uid: "C2zK5f13SbCtKcyiW5"
}
```

### 9.2 Crop

Cropping is useful when you want records to "fit" a schema tightly, e.g.,
```mdtest-command
zq -Z -I connection.zed "crop(this, <connection>)" sample.json
```
removes the `uid` field since it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    client: {
        addr: "10.47.1.100",
        port: 41772
    }
}
```

### 9.3 Fill

Use `fill` when you want to fill out missing fields with nulls, e.g.,
```mdtest-command
zq -Z -I connection.zed "fill(this, <connection>)" sample.json
```
adds a null-valued `vlan` field since the input value is missing it and
the `connection` type has it:
```mdtest-output
{
    kind: "dns",
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    client: {
        addr: "10.47.1.100",
        port: 41772
    },
    uid: "C2zK5f13SbCtKcyiW5",
    vlan: null (uint16)
}
```

### 9.4 Order

The `order` function changes the order of fields in its input to match the
order in the specified type, as field order is significant in Zed records, e.g.,
```mdtest-command
zq -Z -I connection.zed "order(this, <connection>)" sample.json
```
reorders the `client` and `server` fields to match the input but does nothing
about the `uid` field as it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: "10.47.1.100",
        port: 41772
    },
    server: {
        addr: "10.0.0.100",
        port: 53
    },
    uid: "C2zK5f13SbCtKcyiW5"
}
```

### 9.5 Shape

The `shape` function brings everything together by applying `cast`,
`fill`, and `order` all in one step, e.g.,
```mdtest-command
zq -Z -I connection.zed "shape(this, <connection>)" sample.json
```
reorders the `client` and `server` fields to match the input but does nothing
about the `uid` field as it is not in the `connection` type:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: 10.47.1.100,
        port: 41772 (port=uint16)
    } (=socket),
    server: {
        addr: 10.0.0.100,
        port: 53
    } (socket),
    vlan: null (uint16),
    uid: "C2zK5f13SbCtKcyiW5"
}
```
To get a tight shape of the target type,
apply `crop` to the output of `shape`, e.g.,
```mdtest-command
zq -Z -I connection.zed "shape(this, <connection>) | crop(this, <connection>)" sample.json
```
drops the `uid` field after shaping:
```mdtest-output
{
    kind: "dns",
    client: {
        addr: 10.47.1.100,
        port: 41772 (port=uint16)
    } (=socket),
    server: {
        addr: 10.0.0.100,
        port: 53
    } (socket),
    vlan: null (uint16)
}
```
## 10. Type Fusion

Type fusion is another important building block of data shaping.
Here, types are operated upon by fusing them together, where the
result is a single fused type.
Some systems call a related process "schema inference" where a set
of values, typically JSON, is analyzed to determine a relational schema
that all the data will fit into.  However, this is just a special case of
type fusion as fusion is fine-grained and based on Zed's type system rather
than having the narrower goal of computing a schema for representations
like relational tables, Parquet, Avro, etc.

Type fusion utilizes two key techniques.

The first technique is to simply combine types with a type union.
For example, an `int64` and a `string` can be merged into a common
type of union `(int64,string)`, e.g., the value sequence `1 "foo"`
can be fused into the single-type sequence:
```
1((int64,string))
"foo"((int64,string))
```
The second technique is to merge fields of records, analogous to a spread
expression.  Here, the value sequence `{a:1}{b:"foo"}` may be
fused into the single-type sequence:
```
{a:1,b:null(string)}
{a:null(int64),b:"foo"}
```

Of course, these two techniques can be powerfully combined,
e.g., where the value sequence `{a:1}{a:"foo",b:2}` may be
fused into the single-type sequence:
```
{a:1((int64,string)),b:null(int64)}
{a:"foo"((int64,string)),b:2}
```

To perform fusion, Zed currently includes two key mechanisms
(though this is an active area of development):
* the [`fuse` operator](operators/fuse.md), and
* the [`fuse` aggregate function](aggregates/fuse.md).

### 10.1 Fuse Operator

The `fuse` operator reads all of its input, computes a fused type using
the techniques above, and outputs the result, e.g.,
```mdtest-command
echo '{x:1} {y:"foo"} {x:2,y:"bar"}' | zq -z fuse -
```
produces
```mdtest-output
{x:1,y:null(string)}
{x:null(int64),y:"foo"}
{x:2,y:"bar"}
```
whereas
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"}{x:2,y:"bar"}' | zq -z fuse -
```
requires a type union for field `x` and produces:
```mdtest-output
{x:1((int64,string)),y:null(string)}
{x:"foo"((int64,string)),y:"foo"}
{x:2((int64,string)),y:"bar"}
```

### 10.2 Fuse Aggregate Function

The `fuse` aggregate function is most often useful during data exploration and discovery
where you might interactively run queries to determine the shapes of some new
or unknown input data and how those various shapes relate to one another.

For example, in the example sequence above, we can use the `fuse` aggregate function to determine
the fused type rather than transforming the values, e.g.,
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"}' | zq -z 'fuse(this)' -
```
results in
```mdtest-output
{fuse:<{x:(int64,string),y:string}>}
```
Since the `fuse` here is an aggregate function, it can also be used with
group-by keys.  Supposing we want to divide records into categories and fuse
the records in each category, we can use a group-by. In this simple example, we
will fuse records based on their number of fields using the
[`len` function:](functions/len.md)
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"}' | zq -z 'fuse(this) by len(this) | sort len' -
```
which produces
```mdtest-output
{len:1,fuse:<{x:int64}>}
{len:2,fuse:<{x:(int64,string),y:string}>}
```
Now, we can turn around and write a "shaper" for data that has the patterns
we "discovered" above, e.g., if this Zed source text is in `shape.zed`
```mdtest-input shape.zed
switch len(this) (
    case 1 => pass
    case 2 => yield shape(this, <{x:(int64,string),y:string}>)
    default => yield error({kind:"unrecognized shape",value:this})
)
```
when we run
```mdtest-command
echo '{x:1} {x:"foo",y:"foo"} {x:2,y:"bar"} {a:1,b:2,c:3}' | zq -z -I shape.zed '| sort -r this' -
```
we get
```mdtest-output
{x:1}
{x:"foo"((int64,string)),y:"foo"}
{x:2((int64,string)),y:"bar"}
error({kind:"unrecognized shape",value:{a:1,b:2,c:3}})
```
