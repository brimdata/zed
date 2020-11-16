package zbuf

type Order bool

const (
	OrderAsc  = Order(false)
	OrderDesc = Order(true)
)

func (o Order) Int() int {
	if o {
		return -1
	}
	return 1
}

func (o Order) String() string {
	if o {
		return "descending"
	}
	return "ascending"
}
