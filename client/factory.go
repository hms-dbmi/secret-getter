package client

import (
	"flag"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// Factory ... class of avaiable secret reading clients
type Factory func(conf flag.FlagSet) (Client, error)

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
	Register("file", NewFileClient)
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
