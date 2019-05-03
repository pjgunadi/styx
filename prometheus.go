package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
)

type promResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Values [][]interface{}   `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

type Result struct {
	Metric string
	Values map[string]string
}

func Query(host string, start time.Time, end time.Time, step int, query string) ([]Result, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	u.Path = "api/v1/query_range"
	q := u.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%d", start.Unix()))
	q.Set("end", fmt.Sprintf("%d", end.Unix()))
	q.Set("step", fmt.Sprintf("%d", steps(step, end.Sub(start))))
	u.RawQuery = q.Encode()

	caCert, err := ioutil.ReadFile("cert.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}
	//	response, err := http.Get(u.String())
	response, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("didn't return 200 OK but %s: %s", response.Status, u.String())
	}

	var resp promResponse
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, err
	}

	if resp.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("result type isn't of type matrix: %s", resp.Data.ResultType)
	}

	if len(resp.Data.Result) == 0 {
		return nil, fmt.Errorf(color.YellowString("no timeseries found"))
	}

	var results []Result
	for _, res := range resp.Data.Result {
		r := Result{}
		r.Metric = metricName(res.Metric)

		values := make(map[string]string)
		for _, vals := range res.Values {
			timestamp := vals[0].(float64)
			value := vals[1].(string)
			values[fmt.Sprintf("%.f", timestamp)] = value
		}
		r.Values = values

		results = append(results, r)
	}

	return results, nil
}

func IcpQuery(host string, token string, start time.Time, end time.Time, step int, query string) ([]Result, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	u.Path = "prometheus/api/v1/query_range"
	q := u.Query()
	q.Set("query", query)
	q.Set("start", fmt.Sprintf("%d", start.Unix()))
	q.Set("end", fmt.Sprintf("%d", end.Unix()))
	q.Set("step", fmt.Sprintf("%d", steps(step, end.Sub(start))))
	u.RawQuery = q.Encode()

	caCert, err := ioutil.ReadFile("cert.crt")
	if err != nil {
		log.Fatal(err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	//Request
	req, err := http.NewRequest("GET", u.String(), strings.NewReader(""))
	if err != nil {
		log.Fatal("Error reading request. ", err)
	}

	//Set headers
	req.Header.Set("Authorization", token)

	//Certificate
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		},
	}

	//	response, err := http.Get(u.String())
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("didn't return 200 OK but %s: %s", response.Status, u.String())
	}

	var resp promResponse
	if err := json.NewDecoder(response.Body).Decode(&resp); err != nil {
		return nil, err
	}

	if resp.Data.ResultType != "matrix" {
		return nil, fmt.Errorf("result type isn't of type matrix: %s", resp.Data.ResultType)
	}

	if len(resp.Data.Result) == 0 {
		return nil, fmt.Errorf(color.YellowString("no timeseries found"))
	}

	var results []Result
	for _, res := range resp.Data.Result {
		r := Result{}
		r.Metric = metricName(res.Metric)

		values := make(map[string]string)
		for _, vals := range res.Values {
			timestamp := vals[0].(float64)
			value := vals[1].(string)
			values[fmt.Sprintf("%.f", timestamp)] = value
		}
		r.Values = values

		results = append(results, r)
	}

	return results, nil
}

func steps(step int, dur time.Duration) int {
	if step > 0 {
		return step
	}
	if dur < 15*time.Minute {
		return 1
	}
	if dur < 30*time.Minute {
		return 3
	}
	return int(dur.Minutes() / 4.2)
}

func metricName(metric map[string]string) string {
	if len(metric) == 0 {
		return "{}"
	}

	out := ""
	var inner []string
	for key, value := range metric {
		if key == "__name__" {
			out = value
			continue
		}
		inner = append(inner, fmt.Sprintf(`%s="%s"`, key, value))
	}

	if len(inner) == 0 {
		return out
	}

	sort.Slice(inner, func(i, j int) bool {
		return inner[i] < inner[j]
	})

	return out + "{" + strings.Join(inner, ",") + "}"
}
