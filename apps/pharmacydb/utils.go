package main

func StrSliceToInterfaceSlice(strSlice []string) []interface{} {
	interfaceSlice := make([]interface{}, len(strSlice))
	for i, item := range strSlice {
		interfaceSlice[i] = item
	}

	return interfaceSlice
}

func StrPtr(str string) *string {
	return &str
}
