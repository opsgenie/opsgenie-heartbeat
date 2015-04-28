package main

import "testing"

func TestCreateUrl(t *testing.T) {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = "test"
	var url = createUrl("/v1/test", requestParams)
	if url != "https://api.opsgenie.com/v1/test?apiKey=test" {
		t.Errorf("Url not correct is [%s] but should be [%s]", url, "dd")
	}
}
