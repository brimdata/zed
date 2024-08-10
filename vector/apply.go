func apply(vecs []vector.Any, eval func([]vector.Any) vector.Any) vector.Any {
       pos := firstVariant(vecs)
       if pos < 0 {
               return eval(vecs)
       }
       var results []vector.Any
       for v := range rip(vecs[pos]) {
               vecs[pos] = v
               results = append(results, apply(vecs, eval))
       }
       return stitch(results)
}
