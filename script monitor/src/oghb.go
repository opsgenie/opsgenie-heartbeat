package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

var TIMEOUT = 30
var apiKey *string
var name *string
var apiUrl *string
var action *string
var description *string
var interval *int
var intervalUnit *string
var delete *bool

func main() {
	parseFlags()
	if *action == "start" {
		startHeartbeat()
	} else if *action == "stop" {
		stopHeartbeat()
	} else if *action == "send" {
		sendHeartbeat()
	} else {
		panic("Unknown action flag; use start or stop")
	}
}

func parseFlags() {
	action = flag.String("action", "", "start, stop or send")
	apiKey = flag.String("apiKey", "", "API key")
	name = flag.String("name", "", "heartbeat name")
	apiUrl = flag.String("apiUrl", "https://api.opsgenie.com", "OpsGenie API url")
	description = flag.String("description", "", "heartbeat description")
	interval = flag.Int("timetoexpire", 10, "amount of time OpsGenie waits for a send request before creating alert")
	intervalUnit = flag.String("intervalUnit", "minutes", "minutes, hours or days")
	delete = flag.Bool("delete", false, "delete the heartbeat on stop")
	flag.Parse()

	if *action == "" {
		panic("-action flag must be provided")
	}
	if *apiKey == "" {
		panic("-apiKey flag must be provided")
	}
	if *name == "" {
		panic("-name flag must be provided")
	}
}

func startHeartbeat() {
	heartbeat := getHeartbeat()
	if heartbeat == nil {
		addHeartbeat()
	} else {
		updateHeartbeatWithEnabledTrue(*heartbeat)
	}
	sendHeartbeat()
}

func getHeartbeat() *heartbeat {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = *apiKey
	requestParams["name"] = *name
	statusCode, responseBody := doHttpRequest("GET", "/v1/json/heartbeat/", requestParams, nil)
	if statusCode == 200 {
		heartbeat := &heartbeat{}
		err := json.Unmarshal(responseBody, &heartbeat)
		if err != nil {
			panic(err)
		}
		fmt.Println("Successfully retrieved heartbeat [" + *name + "]")
		return heartbeat
	} else {
		errorResponse := createErrorResponse(responseBody)
		if statusCode == 400 && errorResponse.Code == 17 {
			fmt.Println("Heartbeat [" + *name + "] doesn't exist")
			return nil
		}
		panic("Failed to get heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
}

func addHeartbeat() {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	if *description != "" {
		contentParams["description"] = description
	}
	if *interval != 0 {
		contentParams["interval"] = interval
	}
	if *intervalUnit != "" {
		contentParams["intervalUnit"] = intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to add heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully added heartbeat [" + *name + "]")
}

func updateHeartbeatWithEnabledTrue(heartbeat heartbeat) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = heartbeat.Name
	contentParams["enabled"] = true
	if *description != "" {
		contentParams["description"] = description
	}
	if *interval != 0 {
		contentParams["interval"] = interval
	}
	if *intervalUnit != "" {
		contentParams["intervalUnit"] = intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to update heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully enabled and updated heartbeat [" + *name + "]")
}

func sendHeartbeat() {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/send", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to send heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully sent heartbeat [" + *name + "]")
}

func stopHeartbeat() {
	if *delete == true {
		deleteHeartbeat()
	} else {
		disableHeartbeat()
	}
}

func deleteHeartbeat() {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = *apiKey
	requestParams["name"] = *name
	statusCode, responseBody := doHttpRequest("DELETE", "/v1/json/heartbeat", requestParams, nil)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to delete heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully deleted heartbeat [" + *name + "]")
}

func disableHeartbeat() {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/disable", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to disable heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully disabled heartbeat [" + *name + "]")
}

func createErrorResponse(responseBody []byte) ErrorResponse {
	errResponse := &ErrorResponse{}
	err := json.Unmarshal(responseBody, &errResponse)
	if err != nil {
		panic(err)
	}
	return *errResponse
}

func doHttpRequest(method string, urlSuffix string, requestParameters map[string]string, contentParameters map[string]interface{}) (int, []byte) {
	var buf, _ = json.Marshal(contentParameters)
	body := bytes.NewBuffer(buf)

	var Url *url.URL
	Url, err := url.Parse(*apiUrl + urlSuffix)
	if err != nil {
		panic(err)
	}
	parameters := url.Values{}
	for k, v := range requestParameters {
		parameters.Add(k, v)
	}
	Url.RawQuery = parameters.Encode()

	var request *http.Request
	var _ error
	if contentParameters == nil {
		request, _ = http.NewRequest(method, Url.String(), nil)
	} else {
		request, _ = http.NewRequest(method, Url.String(), body)
	}
	client := getHttpClient(TIMEOUT)

	resp, error := client.Do(request)
    if resp != nil {
		defer resp.Body.Close()
	}
	if error == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			return resp.StatusCode, body
		}
		fmt.Println("Couldn't read the response from opsgenie")
		panic(err)
	} else {
		fmt.Println("Couldn't send the request to opsgenie")
		panic(error)
	}
	return 0, nil
}

func getHttpClient(seconds int) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*time.Duration(seconds))
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * time.Duration(seconds)))
				return conn, nil
			},
		},
	}
	return client
}

type heartbeat struct {
	Name string `json:"name"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}
