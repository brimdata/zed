package plural

func Slice[S ~[]E, E any](s S, suffix string) string {
	if len(s) == 1 {
		return ""
	}
	return suffix
}
