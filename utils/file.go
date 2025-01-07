package utils

import "path/filepath"

func GetServerStoragePathByNames(packageNames ...string) string {
	if len(packageNames) == 0 {
		return ""
	}
	return filepath.Join(packageNames...)
}
