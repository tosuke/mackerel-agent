package mackerel

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	mkr "github.com/mackerelio/mackerel-client-go"
)

func TestNewAPI(t *testing.T) {
	api, err := NewAPI(
		"http://example.com",
		"dummy-key",
		true,
	)

	if err != nil {
		t.Errorf("should not raise error: %v", err)
	}

	if api.BaseURL.String() != "http://example.com" {
		t.Error("should return URL")
	}

	if api.APIKey != "dummy-key" {
		t.Error("should return api key")
	}

	if api.Verbose != true {
		t.Error("should return verbose value")
	}
}

func TestUrlFor(t *testing.T) {
	api, _ := NewAPI(
		"http://example.com",
		"dummy-key",
		true,
	)

	if api.urlFor("/", "").String() != "http://example.com/" {
		t.Error("should return http://example.com/")
	}

	if api.urlFor("/path/to/api", "").String() != "http://example.com/path/to/api" {
		t.Error("should return http://example.com/path/to/api")
	}
}

func TestDo(t *testing.T) {
	handler := func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("X-Api-Key") != "dummy-key" {
			t.Error("X-Api-Key header should contains passed key")
		}
	}
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		handler(res, req)
	}))
	defer ts.Close()

	api, _ := NewAPI(
		ts.URL,
		"dummy-key",
		false,
	)

	req, _ := http.NewRequest("GET", api.urlFor("/", "").String(), nil)
	api.do(req)
}

func TestCreateHost(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		called = true
		if req.URL.Path != "/api/v0/hosts" {
			t.Error("request URL should be /api/v0/hosts but :", req.URL.Path)
		}

		if req.Method != "POST" {
			t.Error("request method should be POST but :", req.Method)
		}

		body, _ := ioutil.ReadAll(req.Body)
		content := string(body)

		var data struct {
			Name          string              `json:"name"`
			Type          string              `json:"type"`
			Status        string              `json:"status"`
			Meta          mkr.HostMeta        `json:"meta"`
			Interfaces    []map[string]string `json:"interfaces"`
			RoleFullnames []string            `json:"roleFullnames"`
		}

		err := json.Unmarshal(body, &data)
		if err != nil {
			t.Fatal("request content should be decoded as json", content)
		}

		if data.Meta.AgentName != "mackerel-agent" {
			t.Error("request sends json including agent-name but: ", data)
		}

		if len(data.Interfaces) == 0 {
			t.Error("request sends json including interfaces but: ", data)
		}
		iface := data.Interfaces[0]
		if iface["name"] != "eth0" || iface["ipAddress"] != "10.0.4.7" {
			t.Error("interface name and ipAddress should be eth0 and 10.0.4.7, respectively, but: ", data)
		}

		if len(data.RoleFullnames) != 1 {
			t.Errorf("roleFullnames must have size 1: %v", data.RoleFullnames)
		}

		if data.RoleFullnames[0] != "My-Service:app-default" {
			t.Errorf("Wrong data for roleFullnames: %v", data.RoleFullnames)
		}

		respJSON, _ := json.Marshal(map[string]interface{}{
			"id": "ABCD123",
		})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)

	var interfaces []mkr.Interface
	interfaces = append(interfaces, mkr.Interface{
		Name:       "eth0",
		IPAddress:  "10.0.4.7",
		MacAddress: "01:23:45:67:89:ab",
	})
	hostParam := mkr.CreateHostParam{
		Name: "dummy",
		Meta: mkr.HostMeta{
			AgentName: "mackerel-agent",
		},
		Interfaces:       interfaces,
		RoleFullnames:    []string{"My-Service:app-default"},
		DisplayName:      "my-display-name",
		CustomIdentifier: "",
	}
	hostID, err := api.CreateHost(hostParam)

	if err != nil {
		t.Error("should not raise error: ", err)
	}

	if !called {
		t.Error("should http-request")
	}

	if hostID != "ABCD123" {
		t.Error("should returns ABCD123 but:", hostID)
	}
}

func TestCreateHostWithNilArgs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/hosts" {
			t.Error("request URL should be /api/v0/hosts but :", req.URL.Path)
		}

		if req.Method != "POST" {
			t.Error("request method should be POST but :", req.Method)
		}

		body, _ := ioutil.ReadAll(req.Body)
		content := string(body)

		var data struct {
			Name          string              `json:"name"`
			Type          string              `json:"type"`
			Status        string              `json:"status"`
			Meta          mkr.HostMeta        `json:"meta"`
			Interfaces    []map[string]string `json:"interfaces"`
			RoleFullnames []string            `json:"roleFullnames"`
		}

		err := json.Unmarshal(body, &data)
		if err != nil {
			t.Fatal("request content should be decoded as json", content)
		}

		respJSON, _ := json.Marshal(map[string]interface{}{
			"id": "ABCD123",
		})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)

	// with nil args
	hostParam := mkr.CreateHostParam{
		Name: "nilsome",
	}
	hostID, err := api.CreateHost(hostParam)
	if err != nil {
		t.Error("should not return error but got: ", err)
	}

	if hostID != "ABCD123" {
		t.Error("should returns ABCD123 but:", hostID)
	}
}

func TestUpdateHost(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		called = true
		if req.URL.Path != "/api/v0/hosts/ABCD123" {
			t.Error("request URL should be /api/v0/hosts/ABCD123 but :", req.URL.Path)
		}

		if req.Method != "PUT" {
			t.Error("request method should be PUT but :", req.Method)
		}

		body, _ := ioutil.ReadAll(req.Body)
		content := string(body)

		var data struct {
			Name          string              `json:"name"`
			Type          string              `json:"type"`
			Status        string              `json:"status"`
			Meta          mkr.HostMeta        `json:"meta"`
			Interfaces    []map[string]string `json:"interfaces"`
			RoleFullnames []string            `json:"roleFullnames"`
			Checks        []map[string]string `json:"checks"`
		}

		err := json.Unmarshal(body, &data)
		if err != nil {
			t.Fatal("request content should be decoded as json", content)
		}

		if data.Meta.AgentName != "mackerel-agent" {
			t.Error("request sends json including agent-name but: ", data)
		}

		if len(data.Interfaces) == 0 {
			t.Error("request sends json including interfaces but: ", data)
		}
		iface := data.Interfaces[0]
		if iface["name"] != "eth0" || iface["ipAddress"] != "10.0.4.7" {
			t.Error("interface name and ipAddress should be eth0 and 10.0.4.7, respectively, but: ", data)
		}

		if len(data.RoleFullnames) != 1 {
			t.Errorf("roleFullnames must have size 1: %v", data.RoleFullnames)
		}

		if data.RoleFullnames[0] != "My-Service:app-default" {
			t.Errorf("Wrong data for roleFullnames: %v", data.RoleFullnames)
		}

		if data.Checks == nil {
			t.Errorf("Wrong data for checks: %v", data.Checks)

		}
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)

	var interfaces []mkr.Interface
	interfaces = append(interfaces, mkr.Interface{
		Name:       "eth0",
		IPAddress:  "10.0.4.7",
		MacAddress: "01:23:45:67:89:ab",
	})

	hostParam := mkr.UpdateHostParam{
		Name: "dummy",
		Meta: mkr.HostMeta{
			AgentName: "mackerel-agent",
		},
		Interfaces:    interfaces,
		RoleFullnames: []string{"My-Service:app-default"},
		Checks: []mkr.CheckConfig{
			mkr.CheckConfig{Name: "check", Memo: "memo"},
		},
	}

	err := api.UpdateHost("ABCD123", hostParam)

	if err != nil {
		t.Error("should not raise error: ", err)
	}

	if !called {
		t.Error("should http-request")
	}
}

func TestUpdateHostStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/hosts/9rxGOHfVF8F/status" {
			t.Error("request URL should be /api/v0/hosts/9rxGOHfVF8F/status but :", req.URL.Path)
		}
		if req.Method != "POST" {
			t.Error("request method should be POST but: ", req.Method)
		}

		body, _ := ioutil.ReadAll(req.Body)

		var data struct {
			Status string `json:"status"`
		}
		err := json.Unmarshal(body, &data)
		if err != nil {
			t.Fatal("request body should be decoded as json", string(body))
		}

		if data.Status != "maintenance" {
			t.Error("request sends json including status but: ", data.Status)
		}

		respJSON, _ := json.Marshal(map[string]bool{
			"success": true,
		})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)
	err := api.UpdateHostStatus("9rxGOHfVF8F", "maintenance")

	if err != nil {
		t.Error("err shoud be nil but: ", err)
	}
}

func TestFindHost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/hosts/9rxGOHfVF8F" {
			t.Error("request URL should be /api/v0/hosts/9rxGOHfVF8F but :", req.URL.Path)
		}

		if req.Method != "GET" {
			t.Error("request method should be GET but :", req.Method)
		}

		respJSON, _ := json.Marshal(map[string]map[string]interface{}{
			"host": {
				"id":     "9rxGOHfVF8F",
				"name":   "mydb001",
				"status": "working",
				"memo":   "memo",
				"roles":  map[string][]string{"My-Service": {"db-master", "db-slave"}},
			},
		})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)
	host, err := api.FindHost("9rxGOHfVF8F")

	if err != nil {
		t.Error("err shoud be nil but: ", err)
	}

	if reflect.DeepEqual(host, &mkr.Host{
		ID:     "9rxGOHfVF8F",
		Name:   "mydb001",
		Type:   "",
		Status: "working",
		Memo:   "memo",
		Roles: mkr.Roles{
			"My-Service": []string{"db-master", "db-slave"},
		},
		CustomIdentifier: "",
	}) != true {
		t.Error("request sends json including memo but: ", host)
	}
}

func TestFindHostByCustomIdentifier(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/hosts" {
			t.Error("request URL should be /api/v0/hosts but :", req.URL.Path)
		}

		if req.Method != "GET" {
			t.Error("request method should be GET but :", req.Method)
		}

		var hosts []map[string]interface{}

		customIdentifier := req.URL.Query().Get("customIdentifier")

		if customIdentifier == "foo-bar" {
			hosts = []map[string]interface{}{
				{
					"id":               "9rxGOHfVF8F",
					"CustomIdentifier": "foo-bar",
					"name":             "mydb001",
					"status":           "working",
					"memo":             "memo",
					"roles":            map[string][]string{"My-Service": {"db-master", "db-slave"}},
				},
			}
		}

		respJSON, _ := json.Marshal(map[string]interface{}{"hosts": hosts})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)

	var tests = []struct {
		customIdentifier string
		host             *mkr.Host
		returnInfoError  bool
	}{
		{
			customIdentifier: "foo-bar",
			host: &mkr.Host{
				ID:               "9rxGOHfVF8F",
				Name:             "mydb001",
				Type:             "",
				Status:           "working",
				CustomIdentifier: "foo-bar",
				Memo:             "memo",
				Roles: mkr.Roles{
					"My-Service": []string{"db-master", "db-slave"},
				},
			},
			returnInfoError: false,
		},
		{
			customIdentifier: "unregistered-custom_identifier",
			host:             nil,
			returnInfoError:  true,
		},
		{
			customIdentifier: "",
			host:             nil,
			returnInfoError:  true,
		},
	}

	for _, tc := range tests {
		host, err := api.FindHostByCustomIdentifier(tc.customIdentifier)
		if tc.returnInfoError {
			if _, ok := err.(*InfoError); !ok {
				t.Error("err shoud be type of *InfoError but: ", reflect.TypeOf(err))
			}
		} else {
			if err != nil {
				t.Error("err shoud be nil but: ", err)
			}
		}
		if reflect.DeepEqual(host, tc.host) != true {
			t.Error("request sends json including memo but: ", host)
		}
	}
}

func TestPostHostMetricValues(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/tsdb" {
			t.Error("request URL should be /api/v0/tsdb but :", req.URL.Path)
		}

		if req.Method != "POST" {
			t.Error("request method should be POST but: ", req.Method)
		}

		body, _ := ioutil.ReadAll(req.Body)

		var values []struct {
			HostID string      `json:"hostID"`
			Name   string      `json:"name"`
			Time   float64     `json:"time"`
			Value  interface{} `json:"value"`
		}

		err := json.Unmarshal(body, &values)
		if err != nil {
			t.Fatal("request body should be decoded as json", string(body))
		}

		if values[0].HostID != "9rxGOHfVF8F" {
			t.Error("request sends json including hostID but: ", values[0].HostID)
		}
		if values[0].Name != "custom.metric.mysql.connections" {
			t.Error("request sends json including name but: ", values[0].Name)
		}
		if values[0].Time != 123456789 {
			t.Error("request sends json including time but: ", values[0].Time)
		}
		if values[0].Value.(float64) != 100 {
			t.Error("request sends json including value but: ", values[0].Value)
		}

		respJSON, _ := json.Marshal(map[string]bool{
			"success": true,
		})

		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)
	err := api.PostMetricValues([]*mkr.HostMetricValue{
		{
			HostID: "9rxGOHfVF8F",
			MetricValue: &mkr.MetricValue{
				Name:  "custom.metric.mysql.connections",
				Time:  123456789,
				Value: 100,
			},
		},
	})

	if err != nil {
		t.Error("err shoud be nil but: ", err)
	}
}

func TestCreateGraphDefs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/graph-defs/create" {
			t.Error("request URL should be /api/v0/graph-defs/create but :", req.URL.Path)
		}

		if req.Method != "POST" {
			t.Error("request method should be GET but :", req.Method)
		}
		body, _ := ioutil.ReadAll(req.Body)

		var datas []struct {
			Name        string                 `json:"name"`
			DisplayName string                 `json:"displayName"`
			Unit        string                 `json:"unit"`
			Metrics     []*mkr.GraphDefsMetric `json:"metrics"`
		}

		err := json.Unmarshal(body, &datas)
		if err != nil {
			t.Fatal("request body should be decoded as json", string(body))
		}
		data := datas[0]

		if data.Name != "mackerel" {
			t.Errorf("request sends json including name but: %s", data.Name)
		}
		if data.DisplayName != "HorseMackerel" {
			t.Errorf("request sends json including DisplayName but: %s", data.Name)
		}
		if !reflect.DeepEqual(
			data.Metrics[0],
			&mkr.GraphDefsMetric{
				Name:        "saba1",
				DisplayName: "aji1",
				IsStacked:   false,
			},
		) {
			t.Error("request sends json including GraphDefsMetric but: ", data.Metrics[0])
		}
		respJSON, _ := json.Marshal(map[string]string{
			"result": "OK",
		})
		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)
	err := api.CreateGraphDefs([]*mkr.GraphDefsParam{
		{
			Name:        "mackerel",
			DisplayName: "HorseMackerel",
			Unit:        "percentage",
			Metrics: []*mkr.GraphDefsMetric{
				{
					Name:        "saba1",
					DisplayName: "aji1",
					IsStacked:   false,
				},
				{
					Name:        "saba2",
					DisplayName: "aji2",
					IsStacked:   false,
				},
			},
		},
	})

	if err != nil {
		t.Error("err shoud be nil but: ", err)
	}
}

func TestRetireHost(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/v0/hosts/9rxGOHfVF8F/retire" {
			t.Error("request URL should be /api/v0/hosts/9rxGOHfVF8F/retire but :", req.URL.Path)
		}
		if req.Method != "POST" {
			t.Error("request method should be POST but: ", req.Method)
		}
		respJSON, _ := json.Marshal(map[string]bool{
			"success": true,
		})
		res.Header()["Content-Type"] = []string{"application/json"}
		fmt.Fprint(res, string(respJSON))
	}))
	defer ts.Close()

	api, _ := NewAPI(ts.URL, "dummy-key", false)
	err := api.RetireHost("9rxGOHfVF8F")

	if err != nil {
		t.Error("err shoud be nil but: ", err)
	}
}

func TestApiError(t *testing.T) {
	aperr := apiError(400, "bad request")

	if !aperr.IsClientError() {
		t.Error("something went wrong")
	}

	if aperr.IsServerError() {
		t.Error("something went wrong")
	}
}
