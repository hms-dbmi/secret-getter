package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hms-dbmi/vault-getter/client/mocks"
	"github.com/stretchr/testify/mock"
)

func TestEnvVariables(t *testing.T) {

	os.Setenv("testkey", "testvalue")
	if _, ok := os.LookupEnv("testkey"); !ok {
		t.Fail()
	}

	if _, ok := os.LookupEnv("testkey2"); ok {
		t.Fail()
	}
}

func TestReplaceVars(t *testing.T) {

	// config values
	*prefixes = "\\${"
	*suffixes = "}"
	*files = "../../test_files/test.txt"

	// prep mock
	mockVault := &mocks.Client{}
	secrets := make([]interface{}, 2)
	secrets[0] = "path_ORACLEHOST"
	secrets[1] = "path_UNUSED_VARIABLE"

	mockVault.On("List", mock.AnythingOfType("string")).Return(secrets)
	mockVault.On("Read", mock.AnythingOfType("string")).Return("vault_localhost")

	// test cases
	testCases := []struct {
		order             string
		expectedEnvValue  string
		expectedFileValue string
	}{
		{"vault", "env_localhost", "vault_localhost"},
		{"env", "env_localhost", "env_localhost"},
		{"override", "vault_localhost", "vault_localhost"},
	}

	for _, tc := range testCases {

		// option
		*order = tc.order

		// reset env variable
		os.Setenv("ORACLEHOST", "env_localhost")

		// reset file
		ioutil.WriteFile(*files, []byte("${ORACLEHOST}"), os.ModePerm)

		// read secrets
		decryptedsecrets, _ := readSecrets(mockVault)
		// run method
		loadFiles(strings.Split(*files, ","), decryptedsecrets, false)

		// check results
		result, error := ioutil.ReadFile(*files)

		defer os.Remove(*files)

		if error != nil ||
			strings.Compare(strings.TrimSpace(string(result)), tc.expectedFileValue) != 0 {
			t.Errorf("In(%s) = file var: %s; expected file var: %s", tc.order, result, tc.expectedFileValue)
		}
		env := os.Getenv("ORACLEHOST")
		if strings.Compare(env, tc.expectedEnvValue) != 0 {
			t.Errorf("In(%s) = env var: %s; expected env var: %s", tc.order, env, tc.expectedEnvValue)
		}

	}

}
