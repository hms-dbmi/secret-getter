package util

import (
	"bufio"
	"io/ioutil"
	"os"
)

// IsDirectory ... todo
func IsDirectory(info os.FileInfo) bool {
	// if this file is a directory,
	// get files from directory, and append to files Object
	switch mode := info.Mode(); {
	case mode.IsDir():
		return true
	}
	return false
}

// GetDirectoryFiles ... todo
func GetDirectoryFiles(path string) ([]string, error) {
	dirfiles, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, dirfile := range dirfiles {
		files = append(files, path+"/"+dirfile.Name())
	}
	return files, nil
}

// WriteLine ... todo
func WriteLine(writer *bufio.Writer, line *string) {
	if line == nil {
		return
	}
	if writer.Available()-len(*line) < 0 {
		writer.Flush()
	}
	writer.WriteString(*line + "\n")
}
