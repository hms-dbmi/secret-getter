package client

import (
	"flag"
	"os"
	"testing"

	"github.com/jcmturner/vaultmock"
)

func TestCreateVaultClient(t *testing.T) {

	// vault server
	//handler := func(w http.ResponseWriter, req *http.Request) {}
	//config, ln := testHTTPServer(t, http.HandlerFunc(handler))
	//defer ln.Close()
	_, addr, _, _, testAppId, testUserId := vaultmock.RunMockVault(t)

	//server.Start()
	//defer server.Close()
	//serverConfig := vaultmock.GetMockVaultConfig()
	var conf = flag.NewFlagSet("vault", flag.ExitOnError)
	print(testUserId)
	print(testAppId)
	conf.String("addr", "", "")
	conf.String("token", "../test_files/token.txt", "")

	os.Setenv("VAULT_ADDR", addr)
	defer os.Setenv("VAULT_ADDR", "")

	// vault client
	client, _ := CreateClient("vault", *conf)

	client.Name()
	client.List("secret")
	client.Read("secret/foo")
	// test cases
	// VAULT_ADDR set,
	// VAULT_ADDRR not set, and addr not made available
	// addr set
	// token is a file
	// token is a string
}
