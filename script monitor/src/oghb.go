package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/codegangsta/cli"
)

var TIMEOUT = 30
var apiUrl = "https://api.opsgenie.com"

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Version = "1.0"
	app.Usage = "Send hartbeats to OpsGenie"
	app.Flags = SharedFlags
	app.Commands = Commands
	app.Run(os.Args)
}

func startHeartbeat(args OpsArgs) {
	heartbeat := getHeartbeat(args)
	if heartbeat == nil {
		addHeartbeat(args)
	} else {
		updateHeartbeatWithEnabledTrue(args, *heartbeat)
	}
	sendHeartbeat(args)
}

func getHeartbeat(args OpsArgs) *Heartbeat {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = args.apiKey
	requestParams["name"] = args.name
	statusCode, responseBody := doHttpRequest("GET", "/v1/json/heartbeat/", requestParams, nil)
	if statusCode == 200 {
		heartbeat := &Heartbeat{}
		err := json.Unmarshal(responseBody, &heartbeat)
		if err != nil {
			panic(err)
		}
		fmt.Println("Successfully retrieved heartbeat [" + args.name + "]")
		return heartbeat
	} else {
		errorResponse := createErrorResponse(responseBody)
		if statusCode == 400 && errorResponse.Code == 17 {
			fmt.Println("Heartbeat [" + args.name + "] doesn't exist")
			return nil
		}
		panic("Failed to get heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
}

func addHeartbeat(args OpsArgs) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = args.apiKey
	contentParams["name"] = args.name
	if args.description != "" {
		contentParams["description"] = args.description
	}
	if args.interval != 0 {
		contentParams["interval"] = args.interval
	}
	if args.intervalUnit != "" {
		contentParams["intervalUnit"] = args.intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to add heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully added heartbeat [" + args.name + "]")
}

func updateHeartbeatWithEnabledTrue(args OpsArgs, heartbeat Heartbeat) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = args.apiKey
	contentParams["id"] = heartbeat.Id
	contentParams["enabled"] = true
	if args.description != "" {
		contentParams["description"] = args.description
	}
	if args.interval != 0 {
		contentParams["interval"] = args.interval
	}
	if args.intervalUnit != "" {
		contentParams["intervalUnit"] = args.intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to update heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully enabled and updated heartbeat [" + args.name + "]")
}

func sendHeartbeat(args OpsArgs) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = args.apiKey
	contentParams["name"] = args.name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/send", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to send heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully sent heartbeat [" + args.name + "]")
}

func stopHeartbeat(args OpsArgs) {
	if args.delete == true {
		deleteHeartbeat(args)
	} else {
		disableHeartbeat(args)
	}
}

func deleteHeartbeat(args OpsArgs) {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = args.apiKey
	requestParams["name"] = args.name
	statusCode, responseBody := doHttpRequest("DELETE", "/v1/json/heartbeat", requestParams, nil)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to delete heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully deleted heartbeat [" + args.name + "]")
}

func disableHeartbeat(args OpsArgs) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = args.apiKey
	contentParams["name"] = args.name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/disable", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to disable heartbeat [" + args.name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully disabled heartbeat [" + args.name + "]")
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
	Url, err := url.Parse(apiUrl + urlSuffix)
	if err != nil {
		panic(err)
	}
	parameters := url.Values{}
	for k, v := range requestParameters {
		parameters.Add(k, v)
	}
	Url.RawQuery = parameters.Encode()

	request, _ := http.NewRequest(method, Url.String(), body)
	client := getHttpClient(TIMEOUT)

	resp, error := client.Do(request)
	if error == nil {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err == nil {
			return resp.StatusCode, body
		} else {
			fmt.Println("Couldn't read the response from opsgenie")
			panic(err)
		}
	} else {
		fmt.Println("Couldn't send the request to opsgenie")
		panic(error)
	}
	if resp != nil {
		defer resp.Body.Close()
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

type Heartbeat struct {
	Id string `json:"id"`
}

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"error"`
}
