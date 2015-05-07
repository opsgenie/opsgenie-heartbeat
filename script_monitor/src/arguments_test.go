package main

import (
	"flag"
	"testing"

	"github.com/codegangsta/cli"
)

func TestMandatoryFlagsNotProvided(t *testing.T) {
	flagsTestHelper(t, mandatoryFlags, createCli("", "", ""))
}

func TestMandatoryFlagsNameNotProvided(t *testing.T) {
	flagsTestHelper(t, mandatoryFlags, createCli("key", "", ""))
}

func TestMandatoryFlagsApiKeyNotProvided(t *testing.T) {
	flagsTestHelper(t, mandatoryFlags, createCli("", "name", ""))
}

func TestFlagWrongValue(t *testing.T) {
	flagsTestHelper(t, intervalWrong, createCli("key", "name", "fake"))
}

func TestAllKeysProvided(t *testing.T) {
	ops := extractArgs(createCliAll("apiKey", "name", "intervalUnit", "description", 11, true))
	if ops.apiKey != "apiKey" && ops.name != "name" && ops.description != "description" && ops.interval != 11 && ops.intervalUnit != "intervalUnit" && ops.delete != true {
		t.Errorf("OpsArgs struct not correct [%+v]", ops)
	}
}

func flagsTestHelper(t *testing.T, msg string, c *cli.Context) {
	var incomingMsg string

	logAndExit = func(msg string) {
		incomingMsg = msg
	}

	extractArgs(c)

	if incomingMsg != msg {
		t.Errorf("Wrong error message [%s]", incomingMsg)
	}
}

func createCli(apiKey string, name string, intervalUnit string) *cli.Context {
	return createCliAll(apiKey, name, intervalUnit, "", 0, true)
}

func createCliAll(apiKey string, name string, intervalUnit string, description string, interval int, delete bool) *cli.Context {
	globalSet := flag.NewFlagSet("testGlobal", 0)
	globalSet.String("apiKey", apiKey, "")
	globalSet.String("name", name, "")
	set := flag.NewFlagSet("test", 0)
	set.String("description", description, "")
	set.Int("interval", interval, "")
	set.String("intervalUnit", intervalUnit, "")
	set.Bool("delete", delete, "")
	return cli.NewContext(nil, set, globalSet)
}
