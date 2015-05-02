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
	code, body := doHttpRequest("GET", "/v1/json/heartbeat/", mandatoryRequestParams(args), nil)
	if code != 200 {
		errorResponse := createErrorResponse(body)
		if code == 400 && errorResponse.Code == 17 {
			log.Infof("Heartbeat [%s] doesn't exist", args.name)
			return nil
		}
		logAndExit(fmt.Sprintf("%#v", errorResponse))
	}
	heartbeat := &Heartbeat{}
	err := json.Unmarshal(body, &heartbeat)
	handleError(err)
	log.Info("Successfully retrieved heartbeat [" + args.name + "]")
	return heartbeat
}

func addHeartbeat(args OpsArgs) {
	doOpsGenieHttpRequest("POST", "/v1/json/heartbeat/", nil, allContentParams(args))
	log.Info("Successfully added heartbeat [" + args.name + "]")
}

func updateHeartbeatWithEnabledTrue(args OpsArgs, heartbeat Heartbeat) {
	var contentParams = allContentParams(args)
	contentParams["id"] = heartbeat.Id
	contentParams["enabled"] = true
	doOpsGenieHttpRequest("POST", "/v1/json/heartbeat", nil, contentParams)
	log.Info("Successfully enabled and updated heartbeat [" + args.name + "]")
}

func sendHeartbeat(args OpsArgs) {
	doOpsGenieHttpRequest("POST", "/v1/json/heartbeat/send", nil, mandatoryContentParams(args))
	log.Info("Successfully sent heartbeat [" + args.name + "]")
}

func sendHeartbeatLoop(args OpsArgs) {
	for _ = range time.Tick(args.loopInterval) {
		sendHeartbeat(args)
	}
}

func stopHeartbeat(args OpsArgs) {
	if args.delete {
		deleteHeartbeat(args)
	} else {
		disableHeartbeat(args)
	}
}

func deleteHeartbeat(args OpsArgs) {
	doOpsGenieHttpRequest("DELETE", "/v1/json/heartbeat", mandatoryRequestParams(args), nil)
	log.Info("Successfully deleted heartbeat [" + args.name + "]")
}

func disableHeartbeat(args OpsArgs) {
	doOpsGenieHttpRequest("POST", "/v1/json/heartbeat/disable", nil, mandatoryContentParams(args))
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

func doOpsGenieHttpRequest(method string, urlSuffix string, requestParameters map[string]string, contentParameters map[string]interface{}) []byte {
	code, body := doHttpRequest(method, urlSuffix, requestParameters, contentParameters)
	if code != 200 {
		logAndExit(fmt.Sprintf("%#v", createErrorResponse(body)))
	}
	return body
}

func doHttpRequest(method string, urlSuffix string, requestParameters map[string]string, contentParameters map[string]interface{}) (int, []byte) {
	resp, err := getHttpClient().Do(createRequest(method, urlSuffix, requestParameters, contentParameters))
	handleError(err)
	body, err := ioutil.ReadAll(resp.Body)
	handleError(err)
	defer resp.Body.Close()
	return resp.StatusCode, body
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
