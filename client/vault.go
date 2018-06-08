package client

import (
	"flag"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
	"github.com/hms-dbmi/secret-getter/util"
	"go.uber.org/zap"
)

// Vault client implemented
// wrapper for Hashicorp's Vault client
type Vault struct {
	// Hashicorp Vault's client
	Client *api.Client
	logger *zap.Logger
}

// NewVaultClient ... create new Vault client
func NewVaultClient(conf flag.FlagSet) (Client, error) {
	// NewVaultClient logger
	var logger, _ = zap.NewProduction()
	defer logger.Sync()

	// Vault struct logger
	var err error
	clientLogger, err := zap.NewProduction()

	defer clientLogger.Sync()

	if err != nil {
		logger.Fatal("failed to initialize logger", zap.Error(err))
	}

	// initialize Vault Client
	vaultClient, err := api.NewClient(nil)

	if err != nil {
		conf.Usage()
		logger.Fatal("failed to initialize Vault client", zap.Error(err))
	}

	// Set Address
	address := conf.Lookup("addr").Value.String()
	if address != "" {
		vaultClient.SetAddress(address)
	} // else VAULT_ADDR will be used

	// Set Token
	// is the token within a file?
	token := conf.Lookup("token").Value.String()

	if info, err := os.Stat(token); err == nil && !util.IsDirectory(info) {
		// grab it as a single batch
		data, err := ioutil.ReadFile(token)
		if err != nil {
			logger.Error("failed to read token file", zap.Error(err))
		}

		// only set non-empty token
		if len(data) > 0 {
			vaultClient.SetToken(strings.TrimSpace(string(data)))
		}
	} else if token != "" {
		vaultClient.SetToken(token)
	} // else VAULT_TOKEN will be used

	return &Vault{
		Client: vaultClient,
		logger: clientLogger,
	}, nil
}

// Name ... Type of client
func (v *Vault) Name() string {
	return "vault"
}

// List returns list of keys
func (v *Vault) List(path string) interface{} {
	secret, err := v.Client.Logical().List(path)
	if err != nil || secret == nil || secret.Data == nil || secret.Data["keys"] == nil {
		v.logger.Warn("Failed to list keys", zap.Error(err))
		// return empty interface
		return interface{}(nil)
	}
	return secret.Data["keys"]
}

// Read returns value for path/key
func (v *Vault) Read(path string) string {
	secret, err := v.Client.Logical().Read(path)
	if err != nil || secret == nil || secret.Data["value"] == nil {
		v.logger.Warn("Failed to read secret", zap.Error(err))
		return ""
	}
	return secret.Data["value"].(string)
}
