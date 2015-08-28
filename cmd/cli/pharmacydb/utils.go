package main

func strSliceToInterfaceSlice(strSlice []string) []interface{} {
	interfaceSlice := make([]interface{}, len(strSlice))
	for i, item := range strSlice {
		interfaceSlice[i] = item
	}

	return interfaceSlice
}
