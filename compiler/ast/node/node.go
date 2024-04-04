package node

type Node interface {
	Pos() (int, int)
}

type Base struct {
	Start, End int
}

func (b Base) Pos() (int, int) { return b.Start, b.End }
