package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
)

func startHeartbeatAndSend(args OpsArgs) {
	startHeartbeat(args)
	sendHeartbeat(args)
}

func startHeartbeat(args OpsArgs) {
	heartbeat := getHeartbeat(args)
	if heartbeat == nil {
		addHeartbeat(args)
	} else {
		updateHeartbeatWithEnabledTrue(args, *heartbeat)
	}
}

func startHeartbeatLoop(args OpsArgs) {
	startHeartbeat(args)
	sendHeartbeatLoop(args)
}

func getHeartbeat(args OpsArgs) *Heartbeat {
	responseBody := doHttpRequest("GET", "/v1/json/heartbeat/", mandatoryRequestParams(args), nil)
	heartbeat := &Heartbeat{}
	err := json.Unmarshal(responseBody, &heartbeat)
	handleError(err)
	log.Info("Successfully retrieved heartbeat [" + args.name + "]")
	return heartbeat
}

func addHeartbeat(args OpsArgs) {
	doHttpRequest("POST", "/v1/json/heartbeat/", nil, allContentParams(args))
	log.Info("Successfully added heartbeat [" + args.name + "]")
}

func updateHeartbeatWithEnabledTrue(args OpsArgs, heartbeat Heartbeat) {
	var contentParams = allContentParams(args)
	contentParams["id"] = heartbeat.Id
	contentParams["enabled"] = true
	doHttpRequest("POST", "/v1/json/heartbeat", nil, contentParams)
	log.Info("Successfully enabled and updated heartbeat [" + args.name + "]")
}

func sendHeartbeat(args OpsArgs) {
	doHttpRequest("POST", "/v1/json/heartbeat/send", nil, mandatoryContentParams(args))
	log.Info("Successfully sent heartbeat [" + args.name + "]")
}

func sendHeartbeatLoop(args OpsArgs) {
	ticker := time.NewTicker(time.Second * args.loopInterval)
	go func() {
		for range ticker.C {
			sendHeartbeat(args)
		}
	}()
}

func stopHeartbeat(args OpsArgs) {
	if args.delete {
		deleteHeartbeat(args)
	} else {
		disableHeartbeat(args)
	}
}

func deleteHeartbeat(args OpsArgs) {
	doHttpRequest("DELETE", "/v1/json/heartbeat", mandatoryRequestParams(args), nil)
	log.Info("Successfully deleted heartbeat [" + args.name + "]")
}

func disableHeartbeat(args OpsArgs) {
	doHttpRequest("POST", "/v1/json/heartbeat/disable", nil, mandatoryContentParams(args))
	log.Info("Successfully disabled heartbeat [" + args.name + "]")
}

func mandatoryContentParams(args OpsArgs) map[string]interface{} {
	var contentParams = make(map[string]interface{})
	contentParams["apiKey"] = args.apiKey
	contentParams["name"] = args.name
	return contentParams
}

func allContentParams(args OpsArgs) map[string]interface{} {
	var contentParams = mandatoryContentParams(args)
	if args.description != "" {
		contentParams["description"] = args.description
	}
	if args.interval != 0 {
		contentParams["interval"] = args.interval
	}
	if args.intervalUnit != "" {
		contentParams["intervalUnit"] = args.intervalUnit
	}
	return contentParams
}

func mandatoryRequestParams(args OpsArgs) map[string]string {
	var requestParams = make(map[string]string)
	requestParams["apiKey"] = args.apiKey
	requestParams["name"] = args.name
	return requestParams
}

func createErrorResponse(responseBody []byte) ErrorResponse {
	errResponse := &ErrorResponse{}
	err := json.Unmarshal(responseBody, &errResponse)
	handleError(err)
	return *errResponse
}

func doHttpRequest(method string, urlSuffix string, requestParameters map[string]string, contentParameters map[string]interface{}) []byte {
	resp, err := getHttpClient().Do(createRequest(method, urlSuffix, requestParameters, contentParameters))
	handleError(err)
	body, err := ioutil.ReadAll(resp.Body)
	handleError(err)
	if resp.StatusCode != 200 {
		logAndExit(fmt.Sprintf("%#v", createErrorResponse(body)))
	}
	defer resp.Body.Close()
	return body
}

func createRequest(method string, urlSuffix string, requestParameters map[string]string, contentParameters map[string]interface{}) *http.Request {
	var body, err = json.Marshal(contentParameters)
	handleError(err)
	request, err := http.NewRequest(method, createUrl(urlSuffix, requestParameters), bytes.NewReader(body))
	handleError(err)
	return request
}

func createUrl(urlSuffix string, requestParameters map[string]string) string {
	var Url *url.URL
	Url, err := url.Parse(apiUrl + urlSuffix)
	handleError(err)
	parameters := url.Values{}
	for k, v := range requestParameters {
		parameters.Add(k, v)
	}
	Url.RawQuery = parameters.Encode()
	return Url.String()
}

func handleError(err error) {
	if err != nil {
		logAndExit(err.Error())
	}
}

func getHttpClient() *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, TIMEOUT)
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(TIMEOUT))
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
