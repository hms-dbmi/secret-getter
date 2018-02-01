package client

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/vault/api"
	"go.uber.org/zap"
)

// Factory ... class of avaiable secret reading clients
type Factory func(conf flag.FlagSet) (Client, error)

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

var clientFactories = make(map[string]Factory)

// Register ... register available clients
func Register(name string, factory Factory) {

	var logger, _ = zap.NewProduction()
	defer logger.Sync()

	if factory == nil {
		logger.Panic("Client factory %s does not exist.", zap.String("name", name))
	}
	_, registered := clientFactories[name]
	if registered {
		logger.Error("Client factory %s already registered. Ignoring.", zap.String("name", name))
	}
	clientFactories[name] = factory
}

func init() {
	Register("vault", NewVaultClient)
	//Register("file", NewFileClient)
}

// CreateClient ... create client by available types ("vault", "file")
func CreateClient(clientName string, conf flag.FlagSet) (Client, error) {

	clientFactory, ok := clientFactories[clientName]
	if !ok {
		// Factory has not been registered.
		// Make a list of all available datastore factories for logging.
		availableClients := make([]string, len(clientFactories))
		for k := range clientFactories {
			availableClients = append(availableClients, k)
		}
		return nil, fmt.Errorf(fmt.Sprintf("Invalid client name. Must be one of: %s", strings.Join(availableClients, ", ")))
	}

	// Run the factory with the configuration.
	return clientFactory(conf)
}
