package main

import (
	"testing"
	"time"
)

var testArgs = OpsArgs{"testKey", "testName", "testDescription", 99, "month", time.Second * 10, true}

func TestCreateUrl(t *testing.T) {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = "test"
	var url = createUrl("/v1/test", requestParams)
	var testUrl = "https://api.opsgenie.com/v1/test?apiKey=test"
	if url != testUrl {
		t.Errorf("Url not correct is [%s] but should be [%s]", url, testUrl)
	}
}

func TestAllContentParams(t *testing.T) {
	var all = allContentParams(testArgs)
	if all["apiKey"] != testArgs.apiKey || all["name"] != testArgs.name || all["description"] != testArgs.description || all["interval"] != testArgs.interval || all["intervalUnit"] != testArgs.intervalUnit {
		t.Errorf("OpsArgs [%+v] are not the same as all content params [%s]", testArgs, all)
	}
}

func TestMandatoryRequestParams(t *testing.T) {
	var params = mandatoryRequestParams(testArgs)
	if params["apiKey"] != testArgs.apiKey || params["name"] != testArgs.name {
		t.Errorf("Requested params [%s] are not the same as from OpsArgs [%+v]", params, testArgs)
	}
}

func TestCreateErrorResponse(t *testing.T) {
	json := `{"code":10, "error": "test error"}`
	error := createErrorResponse([]byte(json))
	if error.Code != 10 || error.Message != "test error" {
		t.Errorf("Error [%+v] does not correspond to json [%s]", error, json)
	}
}
