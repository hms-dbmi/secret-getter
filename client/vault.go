package client

import (
	"flag"
	"os"

	"github.com/hashicorp/vault/api"
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
	var logger, _ = zap.NewProduction()
	defer logger.Sync()

	var err error
	clientLogger, err := zap.NewProduction()
	defer clientLogger.Sync()

	if err != nil {
		logger.Fatal("failed to initialize logger", zap.Error(err))
	}

	vaultClient, err := api.NewClient(nil)

	if err != nil {
		conf.Usage()
		logger.Fatal("failed to initialize Vault client", zap.Error(err))
	}

	// specific environment variable. VAULT_ADDR, VAULT_TOKEN, VAULT_PATH
	// Vault address
	// Vault path
	// Vault access token are encrypted variables

	for _, key := range []string{"VAULT_ADDR", "VAULT_TOKEN", "VAULT_PATH"} {
		if value := os.Getenv(key); value != "" {
			conf.Set(key, value)
		}

	}

	if conf.Lookup("path").Value.String() == "" {
		conf.Usage()
		logger.Fatal("Vault path must be defined")
	}

	vaultClient.SetAddress(conf.Lookup("addr").Value.String())
	vaultClient.SetToken(conf.Lookup("token").Value.String())

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
	if err != nil || secret.Data == nil || secret.Data["keys"] == nil {
		v.logger.Error("Failed to list keys", zap.Errors("error", []error{err}))
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
