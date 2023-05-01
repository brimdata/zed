package vam

// XXX for now this is a list of slots, but it probably should be a roaring bitmap
type Index []int32

func (i Index) And(with Index) Index {
	var head, tail, from int
	for {
		for i[tail] < with[from] {
			tail++
			if tail >= len(i) {
				break
			}
		}
		if i[tail] == with[from] {
			i[head] = i[tail]
			head++
		} else {
			from++
			if from >= len(with) {
				break
			}
		}
	}
	return i[:head]
}

func (i Index) Or(with Index) Index {
	panic("Index.Or TBD")
}
