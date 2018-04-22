package main

import "testing"

func TestRunInstance(t *testing.T) {
	bootstrap()

	_, err := LaunchAMI("", "", "us-east-1", "ami-f517b88a", "", 1, 1)
	if err != nil {
		t.Fail()
		Log.Debugf("%s", err)
	}
}
