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

	// TODO: options available per flagset
	// options := map[string][]flag.NewFlagSet

	files         string
	suffixes      string
	prefixes      string
	order         string
	path          string
	mainLogger, _ = zap.NewProduction()
)

const (
	// SgCommand env variable for secret-getter
	SgCommand = "SG_COMMAND"
	// SgOptions env variable for secret-getter
	SgOptions = "SG_OPTIONS"
)

func main() {
	// parse command line arguments
	// get vault values
	// replace in files
	// run main executable
	defer mainLogger.Sync()

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

	// available SG_COMMAND env variable
	// SG_COMMAND overrides command line option
	sgCmd, _ := os.LookupEnv(SgCommand)

	if len(os.Args) > 0 {
		separate := strings.Split(os.Args[0], " ")
		// command line argument will override
		for _, avail := range client.Available() {
			// first commmand line argument matches available client
			// overrides SG_COMMAND env variable
			if separate[0] == avail {
				sgCmd = separate[0]
				os.Args[0] = strings.Join(separate[1:], " ")
				break
			}
		}
	}

	sgEnvOptions, _ := os.LookupEnv(SgOptions)
	var options []string
	// loop through commannd line arguments and SG_OPTIONS for secret-getter options
	// command line options override SG_OPTIONS
	for _, option := range append(strings.Split(sgEnvOptions, " "), os.Args...) {
		// flags.Parse() do not like empty strings :/ -Andre
		if option != "" {
			options = append(options, option)
		}
	}

	mainLogger.Info("command", zap.String("command", sgCmd))
	mainLogger.Info("options", zap.Strings("options", options))

	// set values
	switch sgCmd {
	case "vault":
		vaultCommand.Parse(options)
		path = vaultCommand.Lookup("path").Value.String()
		files = vaultCommand.Lookup("files").Value.String()
		prefixes = vaultCommand.Lookup("prefix").Value.String()
		suffixes = vaultCommand.Lookup("suffix").Value.String()
		order = vaultCommand.Lookup("order").Value.String()
		cli, err = client.CreateClient("vault", *vaultCommand)
		if err != nil {
			mainLogger.Fatal("failed to initialize Vault client", zap.Error(err))
		}
	case "file":
		fileCommand.Parse(options)
		path = fileCommand.Lookup("path").Value.String()
		files = fileCommand.Lookup("files").Value.String()
		prefixes = fileCommand.Lookup("prefix").Value.String()
		suffixes = fileCommand.Lookup("suffix").Value.String()
		order = fileCommand.Lookup("order").Value.String()
		cli, err = client.CreateClient("file", *fileCommand)
		if err != nil {
			mainLogger.Fatal("failed to initialize file client", zap.Error(err))
		}
	case "help":
		vaultCommand.Usage()
		fileCommand.Usage()
	default:
		fmt.Println("required secret-getter subcommand. Available: %v", client.Available())
		os.Exit(1)
	}

	// get secrets
	decryptedSecrets, err := readSecrets(cli)

	// variable replacement
	if err == nil {
		loadFiles(strings.Split(files, ","), decryptedSecrets, false)
	}

	// unset vault token variable
	os.Setenv("VAULT_TOKEN", "")

	// run next command
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			if err := execute(args); err != nil {
				mainLogger.Fatal("failed to execute command",
					zap.Strings("args", args),
					zap.Error(err))
			}
			break
		}
	}

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
	mainLogger.Debug("Searching for match.", zap.String("expression", exp.String()))
	for _, file := range files {

		// keep permissions the same
		info, err := os.Stat(file)
		if err != nil {
			mainLogger.Error("Could not get stats on file", zap.Error(err))
			continue
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
			mainLogger.Error("Could not read file", zap.Error(err))
			continue
		}
		defer fi.Close()
		scanner := bufio.NewScanner(fi)

		// create temp file for writing
		fo, err := os.OpenFile(file+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			mainLogger.Error("Could not create file", zap.Error(err))
			continue
		}
		defer fo.Close()
		writer := bufio.NewWriter(fo)

		// search for regex matches
		for scanner.Scan() {
			line := scanner.Text()
			mainLogger.Debug("", zap.String("line", line))

			match := exp.FindAllStringSubmatch(line, -1)
			if match == nil || len(match) == 0 {
				mainLogger.Debug("no variables in line found matching pattern", zap.String("regex", exp.String()))
				util.WriteLine(writer, &line)
				continue
			}

			// search through all found variable matches
			for j := range match {
				for i, name := range exp.SubexpNames() {
					if name != "var" {
						continue
					}
					mainLogger.Debug("variable found in line", zap.String("match", match[j][i]))

					// replace with non-empty secret
					variable := match[j][i]
					if (*secrets)[variable] != "" {
						// order==env will use environment variable non-empty value instead of vault value
						if order == "env" && os.Getenv(variable) != "" {
							(*secrets)[variable] = os.Getenv(variable)
						}

						line = strings.Replace(line, match[j][0], (*secrets)[variable], 1)

					} else {
						mainLogger.Debug("unknown key", zap.String("variable", match[j][0]))
					}
				}
			}

			util.WriteLine(writer, &line)

		}
		writer.Flush()
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
			// TODO: secretsOut should be internal to the client - Andre
			secretsOut[key] = value
			// removed legacy format conversion. We are no longer hacking for
			// stack/stack_key path/key formats
			// additionally, keys are now case sensitive - Andre

			// order=override will override environment variables with vault values
			if _, ok := os.LookupEnv(key); order == "override" && ok {
				os.Setenv(key, secretsOut[key])
			}
		}
	}

	return &secretsOut, nil
}
