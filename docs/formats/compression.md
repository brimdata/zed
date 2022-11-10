---
sidebar_position: 6
sidebar_label: Compression
---

# ZNG Compression Types

This document specifies values for the `<format>` byte of a
[ZNG compressed value message block](zng.md#2-the-zng-format)
and the corresponding algorithms for the `<compressed payload>` byte sequence.

As new compression algorithms are specified, they will be documented
here without any need to change the ZNG specification.

Of the 256 possible values for the `<format>` byte, only type `0` is currently
defined and specifies that `<compressed payload>` contains an
[LZ4 block](https://github.com/lz4/lz4/blob/master/doc/lz4_Block_format.md).
