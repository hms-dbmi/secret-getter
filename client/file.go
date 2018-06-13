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
	secret map[string]interface{}
}

// NewFileClient ... create new Vault client
func NewFileClient(conf flag.FlagSet) (Client, error) {

	// File struct logger
	var err error
	clientLogger, err := zap.NewProduction()
	defer clientLogger.Sync()

	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})

	// open the file
	filePath := conf.Lookup("path").Value.String()

	if filePath == "" {
		conf.Usage()
		return nil, errors.New("File path must be defined (--path)")
	}

	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// if this is a directory
	if util.IsDirectory(info) {
		return nil, errors.New("File path must be a file (--path)")
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

	fileClient := &File{
		logger: clientLogger,
		secret: data,
	}

	return fileClient, nil
}

// Name ... Type of client
func (f *File) Name() string {
	return "file"
}

// List returns list of keys
func (f *File) List(path string) interface{} {
	// TODO: force clients to map their data into maps
	// We are expecting data to be treated similarly to Vault client
	// Vault client needs to be updated - Andre
	if f.secret["__internal_keys"] != nil {
		return f.secret["__internal_keys"]
	}

	keys := make([]interface{}, 0)
	for key := range f.secret {
		keys = append(keys, key)
	}
	f.secret["__internal_keys"] = keys
	f.secret["__internal_path"] = path

	return f.secret["__internal_keys"]
}

// Read returns value for path/key
func (f *File) Read(path string) string {
	// TODO: need to ignore extended path
	// lots of legacy dealing with Vault client expectations
	// this will break if List() not called first. - Andre
	path = strings.TrimPrefix(path, f.secret["__internal_path"].(string)+"/")
	if val, ok := (f.secret)[path]; ok {
		return val.(string)
	}
	return ""
}
