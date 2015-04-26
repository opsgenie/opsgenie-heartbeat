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

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "apiKey, k",
			Value: "",
			Usage: "API key",
		},
		cli.StringFlag{
			Name:  "name, n",
			Value: "",
			Usage: "heartbeat name",
		},
		cli.StringFlag{
			Name:  "description, d",
			Value: "",
			Usage: "heartbeat description",
		},
		cli.IntFlag{
			Name:  "interval, i",
			Value: 10,
			Usage: "amount of time OpsGenie waits for a send request before creating alert",
		},
		cli.StringFlag{
			Value: "minutes",
			Name:  "intervalUnit, u",
			Usage: "minutes, hours or days",
		},
		cli.BoolFlag{
			Name:  "delete",
			Usage: "elete the heartbeat on stop",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:        "start",
			Usage:       "Adds a new heartbeat and then sends a hartbeat",
			Description: "Adds a new heartbeat to OpsGenie with the configuration from the given flags. If the heartbeat with the name specified in -name exists, updates the heartbeat accordingly and enables it. It also sends a heartbeat message to activate the heartbeat.",
			Action: func(c *cli.Context) {
				startHeartbeat(c.GlobalString("apiKey"), c.GlobalString("name"), c.GlobalString("description"), c.GlobalInt("interval"), c.GlobalString("intervalUnit"))
			},
		},
		{
			Name:        "stop",
			Usage:       "Disables the heartbeat",
			Description: "Disables the heartbeat specified with -name, or deletes it if -delete is true. This can be used to end the heartbeat monitoring that was previously started.",
			Action: func(c *cli.Context) {
				stopHeartbeat(c.GlobalString("apiKey"), c.GlobalString("name"), c.GlobalBool("delete"))
			},
		},
		{
			Name:        "send",
			Usage:       "Sends a heartbeat",
			Description: "Sends a heartbeat message to reactivate the heartbeat specified with -name.",
			Action: func(c *cli.Context) {
				sendHeartbeat(c.GlobalString("apiKey"), c.GlobalString("name"))
			},
		},
	}

	app.Run(os.Args)
}

func startHeartbeat(apiKey string, name string, description string, interval int, intervalUnit string) {
	heartbeat := getHeartbeat(apiKey, name)
	if heartbeat == nil {
		addHeartbeat(apiKey, name, description, interval, intervalUnit)
	} else {
		updateHeartbeatWithEnabledTrue(apiKey, name, description, interval, intervalUnit, *heartbeat)
	}
	sendHeartbeat(apiKey, name)
}

func getHeartbeat(apiKey string, name string) *Heartbeat {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = apiKey
	requestParams["name"] = name
	statusCode, responseBody := doHttpRequest("GET", "/v1/json/heartbeat/", requestParams, nil)
	if statusCode == 200 {
		heartbeat := &Heartbeat{}
		err := json.Unmarshal(responseBody, &heartbeat)
		if err != nil {
			panic(err)
		}
		fmt.Println("Successfully retrieved heartbeat [" + name + "]")
		return heartbeat
	} else {
		errorResponse := createErrorResponse(responseBody)
		if statusCode == 400 && errorResponse.Code == 17 {
			fmt.Println("Heartbeat [" + name + "] doesn't exist")
			return nil
		}
		panic("Failed to get heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
}

func addHeartbeat(apiKey string, name string, description string, interval int, intervalUnit string) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	if description != "" {
		contentParams["description"] = description
	}
	if interval != 0 {
		contentParams["interval"] = interval
	}
	if intervalUnit != "" {
		contentParams["intervalUnit"] = intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to add heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully added heartbeat [" + name + "]")
}

func updateHeartbeatWithEnabledTrue(apiKey string, name string, description string, interval int, intervalUnit string, heartbeat Heartbeat) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["id"] = heartbeat.Id
	contentParams["enabled"] = true
	if description != "" {
		contentParams["description"] = description
	}
	if interval != 0 {
		contentParams["interval"] = interval
	}
	if intervalUnit != "" {
		contentParams["intervalUnit"] = intervalUnit
	}
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to update heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully enabled and updated heartbeat [" + name + "]")
}

func sendHeartbeat(apiKey string, name string) {
	fmt.Print(apiKey)
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/send", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to send heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully sent heartbeat [" + name + "]")
}

func stopHeartbeat(apiKey string, name string, delete bool) {
	if delete == true {
		deleteHeartbeat(apiKey, name)
	} else {
		disableHeartbeat(apiKey, name)
	}
}

func deleteHeartbeat(apiKey string, name string) {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = apiKey
	requestParams["name"] = name
	statusCode, responseBody := doHttpRequest("DELETE", "/v1/json/heartbeat", requestParams, nil)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to delete heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully deleted heartbeat [" + name + "]")
}

func disableHeartbeat(apiKey string, name string) {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = apiKey
	contentParams["name"] = name
	statusCode, responseBody := doHttpRequest("POST", "/v1/json/heartbeat/disable", nil, contentParams)
	if statusCode != 200 {
		errorResponse := createErrorResponse(responseBody)
		panic("Failed to disable heartbeat [" + name + "]; response from OpsGenie:" + errorResponse.Message)
	}
	fmt.Println("Successfully disabled heartbeat [" + name + "]")
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
