package utils

import "os"

func OpenFile(path string, flag int) (*os.File, error) {
	f, err := os.OpenFile(path, flag, 0644)
	if err != nil {
		return nil, err
	}
	return f, nil
}
