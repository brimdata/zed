package vector

// Apply applies eval to vecs. If any element of vecs is a Variant, Apply rips
// vecs accordingly, applies eval to the ripped vectors, and stitches the
// results together into a Variant. If ripUnions is true, Apply also rips
// Unions.
func Apply(ripUnions bool, eval func(...Any) Any, vecs ...Any) Any {
	if ripUnions {
		for k, vec := range vecs {
			if union, ok := Under(vec).(*Union); ok {
				vecs[k] = union.Variant
			}
		}
	}
	variant, ok := findVariant(vecs)
	if !ok {
		return eval(vecs...)
	}
	var results []Any
	for _, ripped := range rip(vecs, variant) {
		results = append(results, Apply(ripUnions, eval, ripped...))
	}
	// Stitch results together by creating a Variant.
	return NewVariant(variant.Tags, results)
}

func findVariant(vecs []Any) (*Variant, bool) {
	for _, vec := range vecs {
		if variant, ok := vec.(*Variant); ok {
			return variant, true
		}
	}
	return nil, false
}

func rip(vecs []Any, variant *Variant) [][]Any {
	var ripped [][]Any
	for j, rev := range variant.TagMap.Reverse {
		var newVecs []Any
		for _, vec := range vecs {
			if vec == variant {
				newVecs = append(newVecs, variant.Values[j])
			} else {
				newVecs = append(newVecs, NewView(rev, vec))
			}
		}
		ripped = append(ripped, newVecs)
	}
	return ripped
}
