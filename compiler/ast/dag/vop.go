package dag

type Vop interface {
	vopNode()
}

type CountByStringHack struct {
	Kind  string `json:"kind" unpack:""`
	Field string `json:"field"`
}

func (*CountByStringHack) vopNode() {}
func (*CountByStringHack) OpNode()  {}

type SumHack struct {
	Kind  string `json:"kind" unpack:""`
	Field string `json:"field"`
}

func (*SumHack) vopNode() {}
func (*SumHack) OpNode()  {}
