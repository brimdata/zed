package vector

// Apply applies eval to vecs. If any element of vecs is a Dynamic, Apply rips
// vecs accordingly, applies eval to the ripped vectors, and stitches the
// results together into a Dynamic. If ripUnions is true, Apply also rips
// Unions.
func Apply(ripUnions bool, eval func(...Any) Any, vecs ...Any) Any {
	if ripUnions {
		for k, vec := range vecs {
			if union, ok := Under(vec).(*Union); ok {
				vecs[k] = union.Dynamic
			}
		}
	}
	d, ok := findDynamic(vecs)
	if !ok {
		return eval(vecs...)
	}
	var results []Any
	for _, ripped := range rip(vecs, d) {
		results = append(results, Apply(ripUnions, eval, ripped...))
	}
	// Stitch results together in a new Dynamic.
	return NewDynamic(d.Tags, results)
}

func findDynamic(vecs []Any) (*Dynamic, bool) {
	for _, vec := range vecs {
		if d, ok := vec.(*Dynamic); ok {
			return d, true
		}
	}
	return nil, false
}

func rip(vecs []Any, d *Dynamic) [][]Any {
	var ripped [][]Any
	for j, rev := range d.TagMap.Reverse {
		var newVecs []Any
		for _, vec := range vecs {
			if vec == d {
				newVecs = append(newVecs, d.Values[j])
			} else {
				newVecs = append(newVecs, NewView(rev, vec))
			}
		}
		ripped = append(ripped, newVecs)
	}
	return ripped
}
