package pkg

// generic function to remove objects from a slice
func RemoveFromSlice[T comparable](slice []T, target T) []T {
	result := slice[:0]
	for _, v := range slice {
		if v != target {
			result = append(result, v)
		}
	}
	return result

}
