# Table of Contents

- [from_base64](#from_base64)
- [from_hex](#from_hex)
- [to_base64](#to_base64)
- [to_hex](#to_hex)

## from_base64

```
from_base64(string) -> bytes
```

Decode a base64 encoded value into a byte array.

### Example:

```
foo := from_base64(foo)
```

**Input:**
```
{foo:"aGVsbG8gd29ybGQ="}
```

**Output:**
```
{foo:0x68656c6c6f20776f726c64}
```
## from_hex

```
from_hex(string) -> bytes
```

Decode a hex encoded value into a byte array.

### Example:

```
foo := string(from_hex(foo))
```

**Input:**
```
{foo:"68656c6c6f20776f726c64"}
```

**Output:**
```
{foo:"hello world"}
```
## to_base64

```
to_base64(bytes) -> string
```

Base64 encode a value.

### Example:

```
foo := to_base64(foo)
```

**Input:**
```
{foo:"hello word"}
```

**Output:**
```
{foo:"aGVsbG8gd29ybGQ="}
```
## to_hex

```
to_hex(bytes) -> string
```

Hex encode a value.

### Example:

```
foo := to_hex(foo)
```

**Input:**
```
{foo:0x68656c6c6f20776f726c64}
```

**Output:**
```
{foo:"hello world"}
```
