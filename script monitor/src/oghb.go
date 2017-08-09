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
var enabled *bool

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
	enabled = flag.Bool("enabled", true, "enable hearthbeat")
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
	heartbeatName := getHeartbeat()
	if len(heartbeatName) > 0 {
		updateHeartbeatWithEnabledTrue(heartbeatName)
	} else {
		addHeartbeat()
	}
	sendHeartbeat()
}

func getHeartbeat() string {
	var requestParams = make(map[string]string)
	statusCode, responseBody := doHttpRequest("GET", "/v2/heartbeats/" + *name, requestParams, nil)
	if statusCode < 399 {
		heartbeat := &heartbeat{}
		err := json.Unmarshal(responseBody, &heartbeat)
		heartbeatName := heartbeat.Data["name"].(string)

		if err != nil {
			panic(err)
		}
		fmt.Println("Successfully retrieved heartbeat [" + *name + "]")
		return heartbeatName
	} else {
		errorResponse := createErrorResponse(responseBody)
		if statusCode > 399 && statusCode < 500 {
			fmt.Println("Heartbeat [" + *name + "] doesn't exist")
			return ""
		}
		fmt.Println(errorResponse)
		panic("Failed to get heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
}

func addHeartbeat() {
	var contentParams = make(map[string]interface{})
	contentParams["name"] = name
	contentParams["enabled"] = enabled

	if *description != "" {
		contentParams["description"] = description
	}
	if *interval != 0 {
		contentParams["interval"] = interval
	}
	if *intervalUnit != "" {
		contentParams["intervalUnit"] = intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v2/heartbeats", nil, contentParams)
	if statusCode > 399 && statusCode < 500 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to add heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully added heartbeat [" + *name + "]")
}

func updateHeartbeatWithEnabledTrue(heartbeatName string) {
	var contentParams = make(map[string]interface{})
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
	statusCode, responseBody := doHttpRequest("PATCH", "/v2/heartbeats/" + heartbeatName , nil, contentParams)

	if statusCode > 399 && statusCode < 500  {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to update heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully enabled and updated heartbeat [" + *name + "]")
}

func sendHeartbeat() {
	statusCode, responseBody := doHttpRequest("POST", "/v2/heartbeats/" + *name + "/ping", nil, nil)

	if statusCode > 399 && statusCode < 500 {
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
	statusCode, responseBody := doHttpRequest("DELETE", "/v2/heartbeats/" + *name, nil, nil)
	if statusCode > 399 && statusCode < 500 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to delete heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully deleted heartbeat [" + *name + "]")
}

func disableHeartbeat() {
	statusCode, responseBody := doHttpRequest("POST", "/v2/heartbeats/" + *name + "/disable", nil, nil)
	if statusCode > 399 && statusCode < 500 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to disable heartbeat [" + *name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully disabled heartbeat [" + *name + "]")
}

func createErrorResponse(responseBody []byte) ErrorResponse {
	fmt.Println(responseBody)
	errResponse := &ErrorResponse{}
	err := json.Unmarshal(responseBody, &errResponse)
	fmt.Println(err)
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

	request.Header.Set("Authorization", "GenieKey " + *apiKey)
	request.Header.Set("Content-Type", "application/json;charset=UTF-8")
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
	Data map[string]interface{
	}
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}
