package state_utils

import "sort"

func SliceOrNil[T any](slice []T) []T {
	if len(slice) == 0 {
		return nil
	}
	return slice
}

func ReadSliceState(slice interface{}) []string {
	var stringSlice []string
	switch sl := slice.(type) {
	default:
		return []string{}
	case []string:
		stringSlice = slice.([]string)
	case []interface{}:
		stringSlice = make([]string, len(sl))
		for i, v := range sl {
			stringSlice[i] = v.(string)
		}
	}

	sort.Strings(stringSlice) //to ensure consistency with TF state

	return stringSlice
}
