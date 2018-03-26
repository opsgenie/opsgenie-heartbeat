package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/apex/log"
)

const TIMEOUT = 30

type ErrorResponse struct {
	RequestID string `json:"requestId"`
	Message   string `json:"message"`
}

func (e *ErrorResponse) Error() string {
	return fmt.Sprintf("Message: %s RequestID: %s", e.Message, e.RequestID)
}

var (
	apiKey       *string
	name         *string
	apiURL       *string
	action       *string
	description  *string
	interval     *int
	intervalUnit *string
	delete       *bool
	enabled      *bool
)

func init() {
	action = flag.String("action", "", "start, stop or send")
	apiKey = flag.String("apiKey", "", "API key")
	name = flag.String("name", "", "heartbeat name")
	apiURL = flag.String("apiUrl", "https://api.opsgenie.com", "OpsGenie API url")
	description = flag.String("description", "", "heartbeat description")
	interval = flag.Int("timetoexpire", 10, "amount of time OpsGenie waits for a send request before creating alert")
	intervalUnit = flag.String("intervalUnit", "minutes", "minutes, hours or days")
	enabled = flag.Bool("enabled", true, "enable hearthbeat")
	delete = flag.Bool("delete", false, "delete the heartbeat on stop")
	flag.Parse()

	if *action == "" {
		print("-action flag must be provided")
		os.Exit(-1)
	}

	if *apiKey == "" {
		print("-apiKey flag must be provided")
		os.Exit(-1)
	}

	if *name == "" {
		print("-name flag must be provided")
		os.Exit(-1)
	}
}

func main() {
	heart := NewHeartbeat()

	switch *action {
	case "start":
		heart.Start()
	case "stop":
		heart.Stop()
	case "send":
		heart.Send()
	default:
		heart.log.Error("Unknown action flag; use start, stop or send")
	}
}

type Heartbeat struct {
	client  *http.Client
	log     *logrus.Logger
	BaseURL *url.URL

	apiKey string
}

func NewHeartbeat() *Heartbeat {
	init := &Heartbeat{}
	init.log = logrus.New()

	apiurl := *apiURL

	init.BaseURL, _ = url.Parse(apiurl)
	init.apiKey = "GenieKey " + *apiKey

	init.client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, time.Second*time.Duration(TIMEOUT))
				if err != nil {
					return nil, err
				}
				conn.SetDeadline(time.Now().Add(time.Second * time.Duration(TIMEOUT)))
				return conn, nil
			},
		},
	}

	return init
}

func (h *Heartbeat) NewRequest(method, relURL string, body io.Reader) (*http.Request, error) {
	rel, err := url.Parse(relURL)
	if err != nil {
		return nil, err
	}

	var u *url.URL = h.BaseURL.ResolveReference(rel)

	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}

	req = req.WithContext(context.Background())

	req.Header.Set("Authorization", h.apiKey)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	return req, nil
}

func (h *Heartbeat) Do(r *http.Request, v interface{}) (*http.Response, error) {
	resp, err := h.client.Do(r)
	if err != nil {
		return nil, err
	}

	err = h.checkResponse(resp)
	if err != nil {
		resp.Body.Close()
		return resp, err
	}

	if v == nil {
		return resp, nil
	}

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(v)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *Heartbeat) Get() string {
	req, err := h.NewRequest("GET", "/v2/heartbeats/"+*name, nil)
	if err != nil {
		h.Error("add", err)
		return ""
	}

	var r struct {
		Data struct {
			Name string `json:"name"`
		} `json:"data"`
	}

	_, err = h.Do(req, &r)

	if err != nil {
		h.Error("add", err)
		return ""
	}

	h.log.Info("Successfully retrieved heartbeat [" + *name + "]")

	return r.Data.Name
}

func (h *Heartbeat) Add() {
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

	var buf, _ = json.Marshal(contentParams)
	body := bytes.NewBuffer(buf)

	req, err := h.NewRequest("POST", "/v2/heartbeats", body)
	if err != nil {
		h.Error("add", err)
	}

	_, err = h.Do(req, nil)

	if err != nil {
		h.Error("add", err)
	} else {
		log.Info("Successfully added heartbeat [" + *name + "]")
	}

}

func (h *Heartbeat) UpdateWithEnabledTrue(heartbeatName string) {
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

	var buf, _ = json.Marshal(contentParams)
	body := bytes.NewBuffer(buf)

	req, err := h.NewRequest("PATCH", "/v2/heartbeats/"+heartbeatName, body)
	if err != nil {
		h.Error("update", err)
	}

	_, err = h.Do(req, nil)

	if err != nil {
		h.Error("update", err)
	} else {
		h.log.Info("Successfully enabled and updated heartbeat heartbeat [" + *name + "]")
	}
}

func (h *Heartbeat) Send() {
	req, err := h.NewRequest("POST", "/v2/heartbeats/"+*name+"/ping", nil)
	if err != nil {
		h.Error("send", err)
	}

	_, err = h.Do(req, nil)

	if err != nil {
		h.Error("send", err)
	} else {
		h.log.Info("Successfully sent heartbeat [" + *name + "]")
	}
}

func (h *Heartbeat) Delete() {
	req, err := h.NewRequest("DELETE", "/v2/heartbeats/"+*name, nil)
	if err != nil {
		h.Error("delete", err)
	}

	_, err = h.Do(req, nil)

	if err != nil {
		h.Error("delete", err)
	} else {
		h.log.Info("Successfully deleted heartbeat [" + *name + "]")
	}
}

func (h *Heartbeat) Disable() {
	req, err := h.NewRequest("POST", "/v2/heartbeats/"+*name+"/disable", nil)
	if err != nil {
		h.Error("disable", err)
	}

	_, err = h.Do(req, nil)

	if err != nil {
		h.Error("disable", err)
	} else {
		h.log.Info("Successfully disabled heartbeat [" + *name + "]")
	}
}

func (h *Heartbeat) Stop() {
	if *delete {
		h.Delete()
	} else {
		h.Disable()
	}
}

func (h *Heartbeat) Start() {
	heartbeatName := h.Get()
	if len(heartbeatName) > 0 {
		h.UpdateWithEnabledTrue(heartbeatName)
	} else {
		h.Add()
	}

	h.Send()
}

func (h *Heartbeat) Error(action string, err error) {
	h.log.Error("Failed to " + action + " heartbeat [" + *name + "]\nError: " + err.Error())
}

func (h *Heartbeat) checkResponse(r *http.Response) error {
	statusCode := r.StatusCode

	if statusCode > 399 && statusCode < 500 {
		errorResponse := &ErrorResponse{}

		data, err := ioutil.ReadAll(r.Body)
		if err == nil && len(data) > 0 {
			err = json.Unmarshal(data, errorResponse)
			if err != nil {
				return fmt.Errorf("unexpected HTTP status: %v", statusCode)
			}
		}

		return errorResponse
	}

	return nil

}
