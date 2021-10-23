# Table of Contents

- [from_base64](#from-base64)
- [from_hex](#from-hex)
- [to_base64](#to-base64)
- [to_hex](#to-hex)

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
foo := from_hex(foo)
```

**Input:**
```
{foo:"68656c6c6f20776f726c64"}
```

**Output:**
```
{foo:0x68656c6c6f20776f726c64}
```
## to_base64

```
to_base64(bytes) -> any
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
to_hex(bytes) -> any
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
