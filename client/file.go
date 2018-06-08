package client

import (
	"bufio"
	"errors"
	"flag"
	"os"
	"regexp"
	"strings"

	"github.com/hms-dbmi/secret-getter/util"
	"go.uber.org/zap"
)

// File client implemented
type File struct {
	logger *zap.Logger
	secret map[string]string
}

// NewFileClient ... create new Vault client
func NewFileClient(conf flag.FlagSet) (Client, error) {
	// NewFileClient method logger
	var logger, _ = zap.NewProduction()
	defer logger.Sync()

	// File struct logger
	var err error
	clientLogger, err := zap.NewProduction()
	defer clientLogger.Sync()

	if err != nil {
		return nil, err
	}

	data := make(map[string]string)

	fileClient := &File{
		logger: clientLogger,
		secret: data,
	}

	// open the file
	filePath := conf.Lookup("path").Value.String()

	if filePath == "" {
		conf.Usage()
		return nil, errors.New("File path must be defined.")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// if this is a directory
	if util.IsDirectory(info) {
		return nil, errors.New("File path must be a file")
	}

	// open file and start reading it line-by-line
	fi, err := os.OpenFile(filePath, os.O_RDONLY, info.Mode())
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	scanner := bufio.NewScanner(fi)
	// keys cannot have spaces
	// values may have spaces within quotes
	exp := regexp.MustCompile("^(?P<key>[^[:space:]|=]+)(\\s|\\t)*=(\\s|\\t)*(?P<value>.*)$")

	for scanner.Scan() {

		match := exp.FindAllStringSubmatch(scanner.Text(), -1)
		if match == nil || len(match) == 0 {
			continue
		}
		// search through all found variable matches
		key := ""
		for j := range match {
			for i, name := range exp.SubexpNames() {
				if name == "key" {
					key = strings.TrimSpace(match[j][i])
				} else if name == "value" && key != "" {
					data[key] = match[j][i]
					key = ""
				}
			}
		}
	}
	return fileClient, nil
}

// Name ... Type of client
func (f *File) Name() string {
	return "file"
}

// List returns list of keys
func (f *File) List(path string) interface{} {
	return f.secret
}

// Read returns value for path/key
func (f *File) Read(path string) string {
	if val, ok := f.secret[path]; ok {
		return val
	}
	return ""
}
