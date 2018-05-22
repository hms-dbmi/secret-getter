package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/hms-dbmi/secret-getter/client"
	"github.com/hms-dbmi/secret-getter/util"
	"go.uber.org/zap"
)

var (
	logger, _ = zap.NewProduction()
	files     string
	suffixes  string
	prefixes  string
	order     string
	path      string
)

func main() {
	// parse command line arguments
	// get vault values
	// replace in files
	// run main executable

	var err error
	var cli client.Client

	var vaultCommand = flag.NewFlagSet("vault", flag.ExitOnError)
	vaultCommand.String("addr", "", "Vault address")
	vaultCommand.String("token", "", "Vault token")
	vaultCommand.String("path", "", "Vault path")
	vaultCommand.String("prefix", "{", "Front prefix")
	vaultCommand.String("suffix", "}", "End prefix")
	vaultCommand.String("files", "", "List of files to replace with Vault secrets")
	vaultCommand.String("order", "vault", "Order of precedence: vault, env, override")

	var fileCommand = flag.NewFlagSet("file", flag.ExitOnError)
	fileCommand.String("path", "", "File path")
	fileCommand.String("prefix", "{", "Front prefix")
	fileCommand.String("suffix", "}", "End prefix")
	fileCommand.String("files", "", "List of files to replace with Vault secrets")
	fileCommand.String("order", "vault", "Order of precedence: vault, env, override")

	switch os.Args[1] {
	case "vault":
		vaultCommand.Parse(os.Args[2:])
		path = vaultCommand.Lookup("path").Value.String()
		files = vaultCommand.Lookup("files").Value.String()
		prefixes = vaultCommand.Lookup("prefix").Value.String()
		suffixes = vaultCommand.Lookup("suffix").Value.String()
		order = vaultCommand.Lookup("order").Value.String()
		cli, err = client.CreateClient("vault", *vaultCommand)
		if err != nil {
			logger.Fatal("faled to initialize Vault client", zap.Error(err))
		}
	case "file":
		fileCommand.Parse(os.Args[2:])
		path = fileCommand.Lookup("path").Value.String()
		files = fileCommand.Lookup("files").Value.String()
		prefixes = fileCommand.Lookup("prefix").Value.String()
		suffixes = fileCommand.Lookup("suffix").Value.String()
		order = fileCommand.Lookup("order").Value.String()
		cli, err = client.CreateClient("file", *fileCommand)
		if err != nil {
			logger.Fatal("faled to initialize file client", zap.Error(err))
		}
	case "help":
		vaultCommand.Usage()
		fileCommand.Usage()
	default:
		fmt.Println("vault or file subcommand is required")
		os.Exit(1)
	}

	/*    if *version {
	          fmt.Printf("Version: %s\n", Version)
	      }
	*/

	// get secrets
	decryptedSecrets, err := readSecrets(cli)

	// variable replacement
	if err == nil {
		loadFiles(strings.Split(files, ","), decryptedSecrets, false)
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

func loadFiles(files []string, secrets *map[string]string, skipDir bool) {

	// exp = prefix (?P<var>[^suffix]*) suffix, e.g ${variable to index}
	// for now, expect delimited strings, e.g. \\$ must be defined by user,
	// should make sure to delimit all regex characters to prevent parsing fubar

	exp := regexp.MustCompile(prefixes + "(?P<var>[^" + suffixes + "]*)" + suffixes)
	logger.Debug("Searching for match.", zap.String("expression", exp.String()))
	for _, file := range files {

		// keep permissions the same
		info, err := os.Stat(file)
		if err != nil {
			logger.Fatal("Could not get stats on file", zap.Error(err))
		}

		// if this is a directory, load those files, then move through to next element
		if util.IsDirectory(info) {
			// prevent recursive (symlinks) and/or deep file loading.
			// sub dirctories need to be explicitly be in the files list
			// e.g. -files=/path/to/dir,/path/to/dir/subdir,
			if !skipDir {
				directoryFiles, _ := util.GetDirectoryFiles(file)
				loadFiles(directoryFiles, secrets, true)
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
				util.WriteLine(writer, &line)
				continue
			}

			// search through all found variable matches
			for j := range match {
				for i, name := range exp.SubexpNames() {
					if name != "var" {
						continue
					}
					logger.Debug("variable found in line", zap.String("match", match[j][i]))
					// replace
					variable := match[j][i]

					if (*secrets)[variable] != "" {
						// order==env will use environment variable non-empty value instead of vault value
						if order == "env" && os.Getenv(variable) != "" {
							(*secrets)[variable] = os.Getenv(variable)
						}

						line = strings.Replace(line, match[j][0], (*secrets)[variable], 1)

					} else {
						logger.Debug("unknown key", zap.String("variable", match[j][0]))
					}
				}
			}

			util.WriteLine(writer, &line)

		}
		writer.Flush()
		fi.Close()
		fo.Close()
		os.Rename(file+".tmp", file)
	}
}

// TODO: load files first, then load values into cache from Vault per key found - Andre
func readSecrets(cli client.Client) (*map[string]string, error) {

	// return list of secret keys
	secrets := cli.List(path)
	// get secret values
	secretsOut := make(map[string]string)
	if keys, ok := secrets.([]interface{}); ok {
		for _, key := range keys {

			key := key.(string)

			value := cli.Read(path + "/" + key)
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
				if _, ok := os.LookupEnv(std); order == "override" && ok {
					// note parent process env variables is not being updated
					// requires syscall
					os.Setenv(std, secretsOut[std])
				}
			}

			// legacy format
			secretsOut[key] = value
			// order=override will override environment variables with vault values
			if _, ok := os.LookupEnv(key); order == "override" && ok {
				os.Setenv(key, secretsOut[key])
			}
		}
	}

	return &secretsOut, nil
}
