package main

import (
	"bufio"
	"flag"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

var (
	addr      = flag.String("addr", "", "Vault address")
	token     = flag.String("token", "", "Vault token")
	path      = flag.String("path", "", "Vault path")
	prefixes  = flag.String("prefix", "{", "Front prefix")
	suffixes  = flag.String("suffix", "}", "End prefix")
	files     = flag.String("files", "", "List of files to replace with Vault secrets")
	order     = flag.String("order", "vault", "Order of precedence: vault or env")
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

	client := initClient()
	// get secrets
	decryptedSecrets, _ := readSecrets(client)
	// variable replacement
	loadFiles(strings.Split(*files, ","), decryptedSecrets)
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

func initClient() *api.Client {

	// specific environment variable. VAULT_ADDR, VAULT_TOKEN, VAULT_PATH
	// Vault address
	// Vault path
	// Vault access token are encrypted variables
	client, err := api.NewClient(nil)
	if err != nil {
		logger.Fatal("faled to initialize Vault client", zap.Error(err))
	}

	if vaultAddr := os.Getenv("VAULT_ADDR"); *addr == "" {
		*addr = vaultAddr
	}
	client.SetAddress(*addr)

	if vaultToken := os.Getenv("VAULT_TOKEN"); *token == "" {
		*token = vaultToken
	}

	//logger.Info("vault token", zap.Object("token", *token))
	client.SetToken(*token)

	if vaultPath := os.Getenv("VAULT_PATH"); *path == "" {
		*path = vaultPath
	}

	if *path == "" {
		logger.Fatal("Vault path must be defined")
	}

	return client

}

func loadFiles(files []string, secrets *map[string]string) {

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

		//var lines []string
		for scanner.Scan() {
			line := scanner.Text()
			//logger.Info("", zap.String("line", line))
			//captures := make(map[string][])
			match := exp.FindAllStringSubmatch(line, -1)
			if match == nil || len(match) == 0 {
				_writeline(writer, &line)
				continue
			}

			// search through all found variable matches
			//logger.Info("match", zap.Object("match", match), zap.Object("len", len(match)))
			for j := range match {
				for i, name := range exp.SubexpNames() {
					if name != "var" {
						continue
					}
					//logger.Info("", zap.Object("name", name), zap.Object("match", match[j][i]))
					// replace
					variable := match[j][i]
					//logger.Info("", zap.Object("variable", variable), zap.Object("value", (*secrets)[variable]))

					if (*secrets)[variable] != "" {
						// use env variable instead of vault (if env has precedence)
						if *order == "env" && os.Getenv(variable) != "" {
							(*secrets)[variable] = os.Getenv(variable)
						}

						line = strings.Replace(line, match[j][0], (*secrets)[variable], 1)
						//line := exp.ReplaceAllString(line, (*secrets)[variable])
						//logger.Info("new line", zap.Object("", line))
					} else {
						logger.Warn("unknown variable", zap.String("variable", match[j][0]))
					}
				}
			}

			_writeline(writer, &line)
			//lines := append(lines, line)
			//logger.Info("lines", zap.Object("", lines))

		}
		writer.Flush()
		fi.Close()
		fo.Close()
		os.Rename(file+".tmp", file)
	}
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

func readSecrets(cli *api.Client) (*map[string]string, error) {

	// temporary
	/*clientToken, err := cli.Logical().Write("auth/github/login", map[string]interface{}{
		"token": *token,
	})

	if err != nil {
		logger.Error("error", zap.Object("error", err), zap.Object("token", *token))
	}
	// end temporary
	logger.Info("clientToken", zap.Object("client", clientToken), zap.Object("auth", clientToken.Auth), zap.Object("path", *path))
	cli.SetToken(clientToken.Auth.ClientToken)*/

	// return list of secret keys
	secret, err := cli.Logical().List(*path)
	if err != nil || secret.Data == nil || secret.Data["keys"] == nil {
		logger.Error("Failed to list keys", zap.String("token", cli.Token()), zap.Errors("error", []error{err}))
		return nil, err
	}

	// get secret values
	secretsOut := make(map[string]string)
	if keys, ok := secret.Data["keys"].([]interface{}); ok {
		for _, key := range keys {

			key := key.(string)

			//logger.Debug("", zap.Object("key", key))

			secret, err = cli.Logical().Read(*path + "/" + key)
			if err != nil || secret == nil || secret.Data["value"] == nil {
				logger.Warn("Failed to read secret", zap.Error(err))
				continue
			}

			// standard format
			// hack
			if std := (strings.SplitN(key, "_", 2))[1]; std != "" {
				secretsOut[strings.ToUpper(std)] = secret.Data["value"].(string)
			}
			// legacy format
			secretsOut[key] = secret.Data["value"].(string)
		}
	}

	//for key := range secretsOut {
	//logger.Info("output", zap.Object(key, secretsOut[key]))
	//}
	return &secretsOut, nil
}
