package main

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hms-dbmi/secret-getter/client/mocks"
	"github.com/stretchr/testify/mock"
)

func TestMain(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// test files
	files := []string{"../../test_files/secrets.txt", "../../test_files/sub_dir/"}

	// test cases
	testCases := []struct {
		SgCommand    string
		SgOptions    string
		args         []string
		expectedExit bool
	}{
		// no SG_COMMAND env variable
		// SG_OPTIONS env variable
		// secret-getter type found in args
		// add additional single file
		{
			"",
			"-path=" + files[0] + " -files=" + files[1],
			[]string{"file"},
			true,
		},
		// no SG_COMMAND
		// SG_OPTIONS
		// secret-getter type and additional options found in args
		{
			"",
			"-path=" + files[0],
			[]string{"file -prefix={", "-suffix=}", "-files=" + files[1]},
			true,
		},
		// SG_COMMAND
		// SG_OPTIONS
		// additional options found in args
		// -- NOTE: duplicate options in args
		{
			"file",
			"-prefix={ -suffix=} -files=" + files[1],
			[]string{"-path=" + files[0], "-prefix=\\${"},
			true,
		},
		// SG_COMMAND
		// no SG_OPTIONS
		// options found in args
		{
			"file",
			"",
			[]string{"-path=" + files[0], "-prefix={", "-suffix=}", "-files=" + files[1]},
			true,
		},
	}

	for ndx, tc := range testCases {

		// reset file if needed
		info, err := os.Stat(files[1])
		if err != nil {
			t.Fatal(err.Error())
		}

		contents := make(map[os.FileInfo][]byte)

		// if not a directory && file does not exist
		if info.IsDir() {
			testfiles, err := ioutil.ReadDir(files[1])
			if err != nil {
				t.Fatal(err.Error())
			}

			for ndx := range testfiles {
				name := files[1] + "/" + testfiles[ndx].Name()
				// save previous state
				contents[testfiles[ndx]], _ = ioutil.ReadFile(name)
				// write new state
				ioutil.WriteFile(name, []byte("{secret_key1}"), os.ModePerm)
			}

			defer func() {
				for file, content := range contents {
					// revert previous state
					ioutil.WriteFile(files[1]+"/"+file.Name(), content, os.ModePerm)
				}
			}()
		}

		os.Setenv("SG_COMMAND", tc.SgCommand)
		defer os.Unsetenv("SG_COMMAND")

		os.Setenv("SG_OPTIONS", tc.SgOptions)
		defer os.Unsetenv("SG_OPTIONS")

		os.Args = tc.args

		ret := t.Run("TestClientArgs:"+string(ndx), func(t *testing.T) {
			main()

		})
		if ret != tc.expectedExit {
			t.Fatalf("process ran with err %v, want exit status %v", ret, tc.expectedExit)
		}

		for file := range contents {
			// check results
			result, err := ioutil.ReadFile(files[1] + "/" + file.Name())
			if err != nil ||
				strings.Compare(strings.TrimSpace(string(result)), "secret_value_1") != 0 {
				t.Errorf("file:\n\n%s\n\nexpected file:\n\n %s", result, "secret_value_1")
				t.Fail()
			}
		}

	}

}

func TestEnvVariables(t *testing.T) {

	// test env lookup
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
	prefixes = "\\${"
	suffixes = "}"
	file := "../../test_files/sub_dir2/test.txt"

	// prep mock
	mockVault := &mocks.Client{}
	secrets := make([]interface{}, 2)
	secrets[0] = "ORACLEHOST"
	secrets[1] = "UNUSED_VARIABLE"

	mockVault.On("List", mock.Anything).Return(secrets)
	mockVault.On("Read", mock.Anything).Return("vault_localhost")

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
		order = tc.order

		// reset env variable
		// current format
		os.Setenv("ORACLEHOST", "env_localhost")

		// reset file
		contents, _ := ioutil.ReadFile(file)
		ioutil.WriteFile(file, []byte("${ORACLEHOST}"), os.ModePerm)

		defer ioutil.WriteFile(file, contents, os.ModePerm)

		// read secrets
		decryptedsecrets, _ := readSecrets(mockVault)
		// run method
		loadFiles([]string{file}, decryptedsecrets, false)

		// check results
		result, error := ioutil.ReadFile(file)

		if error != nil ||
			strings.Compare(strings.TrimSpace(string(result)), tc.expectedFileValue) != 0 {
			t.Errorf("In(%s) = file var: %s; expected file var: %s", tc.order, result, tc.expectedFileValue)
		}

		// normal format
		env := os.Getenv("ORACLEHOST")
		if strings.Compare(env, tc.expectedEnvValue) != 0 {
			t.Errorf("In(%s) = env var: %s; expected env var: %s", tc.order, env, tc.expectedEnvValue)
		}

	}

}
