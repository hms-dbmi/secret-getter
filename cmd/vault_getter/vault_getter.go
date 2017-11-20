package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/hms-dbmi/vault-getter/client"
	"go.uber.org/zap"
)

var (
	addr      = flag.String("addr", "", "Vault address")
	token     = flag.String("token", "", "Vault token")
	path      = flag.String("path", "", "Vault path")
	prefixes  = flag.String("prefix", "{", "Front prefix")
	suffixes  = flag.String("suffix", "}", "End prefix")
	files     = flag.String("files", "", "List of files to replace with Vault secrets")
	order     = flag.String("order", "vault", "Order of precedence: vault, env, override")
	logger, _ = zap.NewProduction()
)

func main() {
	// parse command line arguments
	// get vault values
	// replace in files
	// run main executable

	flag.Parse()

	/*    if *version {
	          fmt.Printf("Version: %s\n", Version)
	      }
	*/

	vault := initClient()

	// get secrets
	decryptedSecrets, err := readSecrets(vault)

	// variable replacement
	if err == nil {
		loadFiles(strings.Split(*files, ","), decryptedSecrets, false)
	}
	// run next command

	/*logger.Info("output: ", zap.Object("client", client), zap.Object("response", secret))
	if err != nil {
		logger.Fatal("failed to send request", zap.Error(err))
	}*/

	// unset vault token variable
	os.Setenv("VAULT_TOKEN", "")

	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			if err := execute(args); err != nil {
				logger.Fatal("failed to execute command",
					zap.Strings("args", args),
					zap.Error(err))
			}
			break
		}
	}

	defer logger.Sync()

}

func execute(argv []string) error {
	if len(argv) == 0 {
		return nil
	}
	argv0, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return execFunc(argv0, argv, os.Environ())
}

var execFunc = syscall.Exec

func initClient() *client.Vault {

	// specific environment variable. VAULT_ADDR, VAULT_TOKEN, VAULT_PATH
	// Vault address
	// Vault path
	// Vault access token are encrypted variables
	vault := &client.Vault{}
	vault.NewClient()
	//if err != nil {
	//	logger.Fatal("faled to initialize Vault client", zap.Error(err))
	//}

	if vaultAddr := os.Getenv("VAULT_ADDR"); *addr == "" {
		*addr = vaultAddr
	}
	vault.Client.SetAddress(*addr)

	if vaultToken := os.Getenv("VAULT_TOKEN"); *token == "" {
		*token = vaultToken
	}

	vault.Client.SetToken(*token)

	if vaultPath := os.Getenv("VAULT_PATH"); *path == "" {
		*path = vaultPath
	}

	if *path == "" {
		logger.Fatal("Vault path must be defined")
	}

	return vault

}

func loadFiles(files []string, secrets *map[string]string, skipDir bool) {

	// exp = prefix (?P<var>[^suffix]*) suffix, e.g ${variable to index}
	// for now, expect delimited strings, e.g. \\$ must be defined by user,
	// should make sure to delimit all regex characters to prevent parsing fubar

	exp := regexp.MustCompile(*prefixes + "(?P<var>[^" + *suffixes + "]*)" + *suffixes)
	logger.Info("Searching for match.", zap.String("expression", exp.String()))
	for _, file := range files {

		// keep permissions the same
		info, err := os.Stat(file)
		if err != nil {
			logger.Fatal("Could not get stats on file", zap.Error(err))
		}

		// if this is a directory, load those files, then move through to next element
		if _isDirectory(info) {
			// prevent recursive (symlinks) and/or deep file loading.
			// sub dirctories need to be explicitly be in the files list
			// e.g. -files=/path/to/dir,/path/to/dir/subdir,
			if !skipDir {
				loadFiles(_getDirectoryFiles(file), secrets, true)
			}
			continue
		}

		// open file and start reading it line-by-line
		fi, err := os.OpenFile(file, os.O_RDONLY, info.Mode())
		if err != nil {
			logger.Fatal("Could not read file", zap.Error(err))
		}
		scanner := bufio.NewScanner(fi)

		fo, err := os.OpenFile(file+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			logger.Fatal("Could not create file", zap.Error(err))
		}
		writer := bufio.NewWriter(fo)

		for scanner.Scan() {
			line := scanner.Text()
			logger.Debug("", zap.String("line", line))

			match := exp.FindAllStringSubmatch(line, -1)
			if match == nil || len(match) == 0 {
				logger.Debug("no variables in line found matching pattern", zap.String("regex", exp.String()))
				_writeline(writer, &line)
				continue
			}

			// search through all found variable matches
			for j := range match {
				for i, name := range exp.SubexpNames() {
					if name != "var" {
						continue
					}
					logger.Info("variable found in line", zap.String("match", match[j][i]))
					// replace
					variable := match[j][i]

					if (*secrets)[variable] != "" {
						// order==env will use environment variable non-empty value instead of vault value
						if *order == "env" && os.Getenv(variable) != "" {
							(*secrets)[variable] = os.Getenv(variable)
						}

						line = strings.Replace(line, match[j][0], (*secrets)[variable], 1)

					} else {
						logger.Info("unknown key", zap.String("variable", match[j][0]))
					}
				}
			}

			_writeline(writer, &line)

		}
		writer.Flush()
		fi.Close()
		fo.Close()
		os.Rename(file+".tmp", file)
	}
}

func _isDirectory(info os.FileInfo) bool {
	// if this file is a directory,
	// get files from directory, and append to files Object
	switch mode := info.Mode(); {
	case mode.IsDir():
		logger.Info(info.Name() + " is a directory.")
		return true
	}
	return false
}

func _getDirectoryFiles(path string) []string {
	dirfiles, err := ioutil.ReadDir(path)
	if err != nil {
		logger.Fatal("Could not read directory", zap.Error(err))
	}

	var files []string
	for _, dirfile := range dirfiles {
		logger.Info("appending ", zap.String("file", path+"/"+dirfile.Name()))
		files = append(files, path+"/"+dirfile.Name())
	}
	return files
}

func _writeline(writer *bufio.Writer, line *string) {
	if line == nil {
		return
	}
	//logger.Info("buffer remaining", zap.Object("buffer", writer.Available()))
	if writer.Available()-len(*line) < 0 {
		writer.Flush()
	}
	writer.WriteString(*line + "\n")
}

func readSecrets(cli client.Client) (*map[string]string, error) {

	// return list of secret keys
	secrets := cli.List(*path)
	// get secret values
	secretsOut := make(map[string]string)
	if keys, ok := secrets.([]interface{}); ok {
		for _, key := range keys {

			key := key.(string)

			value := cli.Read(*path + "/" + key)
			logger.Debug("", zap.String("key", key))

			// HACK TODO: FIX THIS. This limits our secrets options to stack/stack_key
			// If stack/stack_key exists, THEN split, and keep legacy and new format
			// We *could* keep original format AND uppercase .... something to think about
			// Ideally, we should *not* be making guesses on the format of the key
			// e.g nhanes-prod_secret -> nhanes-prod_secret (legacy) && SECRET - Andre

			// standard format
			if std := (strings.SplitN(key, "_", 2))[1]; std != "" {
				std = strings.ToUpper(std)
				secretsOut[std] = value
				logger.Debug("", zap.String("key", std))

				// order=override will override environment variables with vault values
				if _, ok := os.LookupEnv(std); *order == "override" && ok {
					// note parent process env variables is not being updated
					// requires syscall
					os.Setenv(std, secretsOut[std])
				}
			}

			// legacy format
			secretsOut[key] = value
			// order=override will override environment variables with vault values
			if _, ok := os.LookupEnv(key); *order == "override" && ok {
				os.Setenv(key, secretsOut[key])
			}
		}
	}

	return &secretsOut, nil
}
