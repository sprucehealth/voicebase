package dbutil

// AppendStringsToInterfaceSlice appends the string slice to the interface slice.
func AppendStringsToInterfaceSlice(ifs []interface{}, ss []string) []interface{} {
	if cap(ifs) < len(ifs)+len(ss) {
		out := make([]interface{}, len(ifs)+len(ss))
		n := copy(out, ifs)
		for i, s := range ss {
			out[n+i] = s
		}
		return out
	}
	for _, s := range ss {
		ifs = append(ifs, s)
	}
	return ifs
}

// AppendInt64sToInterfaceSlice appends the int64 slice to the interface slice.
func AppendInt64sToInterfaceSlice(ifs []interface{}, is []int64) []interface{} {
	if cap(ifs) < len(ifs)+len(is) {
		out := make([]interface{}, len(ifs)+len(is))
		n := copy(out, ifs)
		for i, j := range is {
			out[n+i] = j
		}
		return out
	}
	for _, j := range is {
		ifs = append(ifs, j)
	}
	return ifs
}
