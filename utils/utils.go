package utils

import (
	"path"
	"strings"
)

// DeriveFilePath creates a new path for the given path by exchanging the
// file extension with the given one.
func DeriveFilePath(filePath, newExt string) string {
	dir, file := path.Split(filePath)
	ext := path.Ext(file)
	basename := file[:len(file)-len(ext)]
	if strings.HasPrefix(newExt, ".") {
		newExt = newExt[1:]
	}
	return path.Join(dir, basename+"."+newExt)
}
