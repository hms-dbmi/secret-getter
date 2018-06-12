package client

import (
	"flag"
	"strings"
	"testing"
)

func TestFileSecrets(t *testing.T) {
	var conf = flag.NewFlagSet("file", flag.ExitOnError)
	conf.String("path", "../test_files/secrets.txt", "File path")
	cli, _ := CreateClient("file", *conf)

	// test cases
	testCases := []struct {
		key           string
		expectedValue string
	}{
		{"secret_key1", "secret_value_1"},
		{"secret_key2", "secret_value_2"},
		// has spaces surrounding  =
		{"secret_key3", "secret_value_3"},
		// value has spaces within quotes
		{"secret_key4", "\"secret_value 4\""},
		// secret 5 is malformed, key has inner spaces
		// key has empty value
		{"secret6", ""},
		// value has quotes
		{"secret7", "\"\""},
	}

	t.Logf("Name of client: %s", cli.Name())

	if len(cli.List("").([]interface{})) != 6 {
		t.Errorf("result len of secrets: (%d), expected len: (%d)", len(cli.List("").([]interface{})), 6)
	}
	for _, tc := range testCases {

		result := cli.Read(tc.key)
		if strings.Compare(result, tc.expectedValue) != 0 {
			t.Errorf("In(%s), result value: (%s); expected value: (%s)", tc.key, result, tc.expectedValue)
		}

	}
	// check for non-existing keys
	result := cli.Read("secret5")
	if strings.Compare(result, "") != 0 {
		t.Errorf("In(%s), result value: (%s); expected value: (%s)", "secret5", result, "")
	}
}

func TestFileErrors(t *testing.T) {
	var conf = flag.NewFlagSet("file", flag.ExitOnError)
	conf.String("path", "../../secrets.txt", "File path")

	_, err := CreateClient("file", *conf)

	if err == nil {
		t.Fail()
	} else {
		t.Logf("Error: (%s)", err.Error())
	}

	conf.Lookup("path").Value.Set("../")

	_, err = CreateClient("file", *conf)
	if err == nil {
		t.Fail()
	} else {
		t.Logf("Error: (%s)", err.Error())
	}

}
