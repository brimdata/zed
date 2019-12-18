package zval

const (
	beginContainer = -1
	endContainer   = -2
)

type node struct {
	innerLen  int
	outerLen  int
	dfs       int
	container bool
}

// Builder implements an API for holding an intermediate representation
// of a hierarchical set of values arranged in a tree, e.g., structured
// values that can contain nested and recursive aggregate values.
// We encode a DFS traversal in a flat data structure that can be
// reused across invocations so we don't otherwise allocate a tree
// data structure for every record parsed that would then be GC'd.
type Builder struct {
	nodes  []node
	leaves [][]byte
}

func NewBuilder() *Builder {
	return &Builder{
		nodes:  make([]node, 0, 64),
		leaves: make([][]byte, 0, 64),
	}
}

func (b *Builder) Reset() {
	b.nodes = b.nodes[:0]
	b.leaves = b.leaves[:0]
}

func (b *Builder) BeginContainer() {
	b.nodes = append(b.nodes, node{dfs: beginContainer})
}

func (b *Builder) EndContainer() {
	b.nodes = append(b.nodes, node{dfs: endContainer})
}

func (b *Builder) AppendUnsetContainer() {
	b.Append(nil, true)
}

func (b *Builder) AppendUnsetValue() {
	b.Append(nil, false)
}

func (b *Builder) Append(leaf []byte, container bool) {
	k := len(b.leaves)
	b.leaves = append(b.leaves, leaf)
	b.nodes = append(b.nodes, node{dfs: k, container: container})
}

func (b *Builder) measure(off int) int {
	node := &b.nodes[off]
	dfs := node.dfs
	if dfs == beginContainer {
		// skip over start token
		off++
		for off < len(b.nodes) {
			next := b.measure(off)
			if next < 0 {
				// skip over end token
				off++
				break
			}
			node.innerLen += b.nodes[off].outerLen
			off = next
		}
		node.outerLen = sizeOfContainer(node.innerLen)
		return off
	}
	if dfs == endContainer {
		return -1
	}
	n := len(b.leaves[dfs])
	node.innerLen = n
	node.outerLen = sizeOfValue(n)
	return off + 1
}

func (b *Builder) encode(dst []byte, off int) (Encoding, int) {
	node := &b.nodes[off]
	dfs := node.dfs
	if dfs == beginContainer {
		// skip over start token
		off++
		dst = AppendUvarint(dst, containerTag(node.innerLen))
		for off < len(b.nodes) {
			var next int
			dst, next = b.encode(dst, off)
			if next < 0 {
				// skip over end token
				off++
				break
			}
			off = next
		}
		return dst, off
	}
	if dfs == endContainer {
		return dst, -1
	}
	if node.container {
		return AppendContainerValue(dst, b.leaves[dfs]), off + 1
	}
	return AppendValue(dst, b.leaves[dfs]), off + 1
}

func (b *Builder) Encode() Encoding {
	off := 0
	for off < len(b.nodes) {
		next := b.measure(off)
		if next < 0 {
			break
		}
		off = next
	}
	off = 0
	var zv []byte
	for off < len(b.nodes) {
		var next int
		zv, next = b.encode(zv, off)
		if next < 0 {
			break
		}
		off = next
	}
	return zv
}
