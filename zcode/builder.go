package zcode

import (
	"encoding/binary"
)

// Builder provides an efficient API for constructing nested ZNG values.
type Builder struct {
	bytes      Bytes
	containers []int // Stack of open containers (as body offsets within bytes).
}

// NewBuilder returns a new Builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Reset resets the Builder to be empty.
func (b *Builder) Reset() {
	b.bytes = nil
	b.containers = b.containers[:0]
}

// BeginContainer opens a new container.
func (b *Builder) BeginContainer() {
	// Allocate one byte for the container tag.  When EndContainer writes
	// the tag, it will arrange for additional space if required.
	b.bytes = append(b.bytes, 0)
	// Push the offset of the container body onto the stack.
	b.containers = append(b.containers, len(b.bytes))
}

// EndContainer closes the most recently opened container.  It panics if the
// receiver has no open container.
func (b *Builder) EndContainer() {
	// Pop the container body offset off the stack.
	bodyOff := b.containers[len(b.containers)-1]
	b.containers = b.containers[:len(b.containers)-1]
	tag := containerTag(len(b.bytes) - bodyOff)
	tagSize := sizeOfUvarint(tag)
	// BeginContainer allocated one byte for the container tag.
	tagOff := bodyOff - 1
	if tagSize > 1 {
		// Need additional space for the tag, so move body over.
		b.bytes = append(b.bytes[:tagOff+tagSize], b.bytes[bodyOff:]...)
	}
	if binary.PutUvarint(b.bytes[tagOff:], tag) != tagSize {
		panic("bad container tag size")
	}
}

// AppendContainer appends val as a container value.
func (b *Builder) AppendContainer(val []byte) {
	b.bytes = AppendContainer(b.bytes, val)
}

// AppendPrimitive appends val as a primitive value.
func (b *Builder) AppendPrimitive(val []byte) {
	b.bytes = AppendPrimitive(b.bytes, val)
}

// Bytes returns the constructed value.  It panics if the receiver has an open
// container.
func (b *Builder) Bytes() Bytes {
	if len(b.containers) > 0 {
		panic("open container")
	}
	return b.bytes
}
