package utils

func ReverseStrMap(originalMap map[string]string) map[string]string {
	reversedMap := make(map[string]string)
	for key, value := range originalMap {
		reversedMap[value] = key
	}
	return reversedMap
}
