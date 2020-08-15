package ranger

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func find(pts []Point, nbin int, r Range) Domain {
	e := NewEnvelope(pts, nbin)
	return e.FindSmallestDomain(r)
}

func TestEnvelope(t *testing.T) {
	t.Parallel()
	pts := []Point{
		{1, 0x100},
		{2, 0x120},
		{3, 0x110},
		{4, 0x130},
		{5, 0x150},
		{6, 0x150},
	}
	d := find(pts, 0, Range{0x151, 0x151})
	assert.Exactly(t, Domain{}, d)
	d = find(pts, 0, Range{0, 0x90})
	assert.Exactly(t, Domain{}, d)
	d = find(pts, 0, Range{0x90, 0x111})
	assert.Exactly(t, Domain{1, 4}, d)
	d = find(pts, 0, Range{0x115, 0x135})
	assert.Exactly(t, Domain{2, 5}, d)
	d = find(pts, 0, Range{0x150, 0x150})
	assert.Exactly(t, Domain{5, math.MaxUint64}, d)
	d = find(pts, 0, Range{0x151, 0x151})
	assert.Exactly(t, Domain{}, d)
	d = find(pts, 3, Range{0x100, 0x109})
	assert.Exactly(t, Domain{1, 3}, d)
	d = find(pts, 3, Range{0x100, 0x120})
	assert.Exactly(t, Domain{1, 5}, d)
	d = find(pts, 3, Range{0x100, 0x130})
	assert.Exactly(t, Domain{1, 5}, d)
	pts[5].Y = 0x149
	d = find(pts, 3, Range{0x100, 0x149})
	assert.Exactly(t, Domain{1, math.MaxUint64}, d)
}

func TestUnion(t *testing.T) {
	env1 := Envelope{
		{1, Range{0x100, 0x120}},
		{3, Range{0x110, 0x130}},
		{5, Range{0x150, 0x150}},
	}
	env2 := Envelope{
		{7, Range{0x110, 0x160}},
		{9, Range{0x170, 0x180}},
		{11, Range{0x190, 0x190}},
		{13, Range{0x180, 0x230}},
	}
	assert.Exactly(t, env1.Merge(env2), Envelope{
		{1, Range{0x100, 0x130}},
		{5, Range{0x110, 0x160}},
		{9, Range{0x170, 0x190}},
		{13, Range{0x180, 0x230}},
	})
	assert.Exactly(t, Envelope{}.Merge(env1), env1)
}
