package zbuf

type Control struct {
	Message interface{}
}

var _ error = (*Control)(nil)

func (c *Control) Error() string {
	return "control"
}
