package ranger_test

import (
	"math"
	"testing"

	"github.com/brimsec/zq/pkg/ranger"
	"github.com/stretchr/testify/assert"
)

func find(pts []ranger.Point, nbin int, r ranger.Range) ranger.Domain {
	e := ranger.NewEnvelope(pts, nbin)
	return e.FindSmallestDomain(r)
}

func TestEnvelope(t *testing.T) {
	t.Parallel()
	pts := []ranger.Point{
		{1, 0x100},
		{2, 0x120},
		{3, 0x110},
		{4, 0x130},
		{5, 0x150},
		{6, 0x150},
	}
	d := find(pts, 0, ranger.Range{0x151, 0x151})
	assert.Exactly(t, ranger.Domain{}, d)
	d = find(pts, 0, ranger.Range{0, 0x90})
	assert.Exactly(t, ranger.Domain{}, d)
	d = find(pts, 0, ranger.Range{0x90, 0x111})
	assert.Exactly(t, ranger.Domain{1, 4}, d)
	d = find(pts, 0, ranger.Range{0x115, 0x135})
	assert.Exactly(t, ranger.Domain{2, 5}, d)
	d = find(pts, 0, ranger.Range{0x150, 0x150})
	assert.Exactly(t, ranger.Domain{5, math.MaxUint64}, d)
	d = find(pts, 0, ranger.Range{0x151, 0x151})
	assert.Exactly(t, ranger.Domain{}, d)
	d = find(pts, 3, ranger.Range{0x100, 0x109})
	assert.Exactly(t, ranger.Domain{1, 3}, d)
	d = find(pts, 3, ranger.Range{0x100, 0x120})
	assert.Exactly(t, ranger.Domain{1, 5}, d)
	d = find(pts, 3, ranger.Range{0x100, 0x130})
	assert.Exactly(t, ranger.Domain{1, 5}, d)
	pts[5].Y = 0x149
	d = find(pts, 3, ranger.Range{0x100, 0x149})
	assert.Exactly(t, ranger.Domain{1, math.MaxUint64}, d)
}
