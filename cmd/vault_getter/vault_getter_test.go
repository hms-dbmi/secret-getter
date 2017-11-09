package main

import "testing"
import "os"

func TestEnvVariables(t *testing.T) {

	os.Setenv("testkey", "testvalue")
	if _, ok := os.LookupEnv("testkey"); !ok {
		t.Fail()
	}

	if _, ok := os.LookupEnv("testkey2"); ok {
		t.Fail()
	}
}
