---
sidebar_position: 2
sidebar_label: Conventions
---

# Type Conventions

Function arguments and operator input values are all dynamically typed,
yet certain functions expect certain data types or classes of data types.
To this end, the function and operator prototypes include a number
of type classes as follows:
* _any_ - any Zed data type
* _float_ - any floating point Zed type
* _int_ - any signed or unsigned Zed integer type
* _number_ - either float or int

Note that there is no "any" type in Zed as all super-structured data is
comprehensively typed; "any" here simply refers to a value that is allowed
to take on any Zed type.
