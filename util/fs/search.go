package fsutil

func SearchForFile(files []string, file string) bool {
	for _, b := range files {
		if b == file {
			return true
		}
	}
	return false
}
