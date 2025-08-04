package utils

func SplitByteArray(data []byte, size int) [][]byte {
	var result [][]byte
	for i := 0; i < len(data); i += size {
		end := i + size
		if end > len(data) {
			end = len(data)
		}
		result = append(result, data[i:end])
	}
	return result
}
