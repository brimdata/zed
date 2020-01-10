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
	// the tag, it arrange for additional space if required.
	b.bytes = append(b.bytes, 0)
	// Push the offset of the container body onto the stack.
	b.containers = append(b.containers, len(b.bytes))
}

// EndContainer closes the most recently opened container.  It panics if there
// is no open container.
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

// AppendUnsetContainer appends an unset container.
func (b *Builder) AppendUnsetContainer() {
	b.Append(nil, true)
}

// AppendUnsetContainer appends an unset value.
func (b *Builder) AppendUnsetValue() {
	b.Append(nil, false)
}

// Append appends leaf as a container if the container boolean is true or as a
// value otherwise.
func (b *Builder) Append(leaf []byte, container bool) {
	b.bytes = Append(b.bytes, leaf, container)
}

// Encode returns the constructed value.  It panics if the Builder has an open
// container.
func (b *Builder) Encode() Bytes {
	if len(b.containers) > 0 {
		panic("open container")
	}
	return b.bytes
}
