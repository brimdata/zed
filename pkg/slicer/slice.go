package slicer

type Slice struct {
	Offset uint64
	Length uint64
}

func (s Slice) Overlaps(x Slice) bool {
	return x.Offset >= s.Offset && x.Offset < s.Offset+x.Length
}
