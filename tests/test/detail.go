package test

type Detail struct {
	Name     string
	Query    string
	Input    string
	Format   string
	Expected string
}

var Suite []Detail

func Add(d Detail) {
	Suite = append(Suite, d)
}
