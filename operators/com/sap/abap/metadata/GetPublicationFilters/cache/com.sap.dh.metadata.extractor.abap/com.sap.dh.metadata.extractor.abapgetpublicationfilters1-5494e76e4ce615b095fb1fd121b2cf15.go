// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type ExtractFiler struct {
	Filters []string `json:"extractionFilters"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

//nolint:gochecknoglobals
var (
	Logf      func(string, ...interface{})
	Errorf    func(string, ...interface{})
	GetString func(string) string
	Out       func(interface{})
	//nolint:exhaustivestruct
	httpClient HTTPClient = &http.Client{}
)

// In handle the port `in` value.
//nolint:unparam
func In(in interface{}) {
	url := GetString("url")
	ctx := context.TODO()

	Logf("get publication filter url: %s\n", url)

	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		Errorf("[get publication filter] failed to create request: %v", err)

		return
	}

	res, err := httpClient.Do(request)
	if err != nil && res.StatusCode != http.StatusOK {
		Errorf("[get publication filter] %v", err)

		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		Errorf("[get publication filter] %v", err)

		return
	}
	defer res.Body.Close()

	extractFilterList := parsePublicationFilter(body)

	if len(extractFilterList) > 0 {
		AdjustFilterItem(extractFilterList)
		Logf("[get publication filter] send extract filter list %+v to next operator", extractFilterList)

		outputJSON, err := json.Marshal(extractFilterList)
		if err != nil {
			Errorf("[get publication filter] Error on make extract filter JSON: %v", err)

			return
		}

		Outf("%s", outputJSON)
	}
}

func AdjustFilterItem(filters []string) {
	for index, filter := range filters {
		if (strings.HasPrefix(filter, "/ODP_BW/") ||
			strings.HasPrefix(filter, "/ODP_SAPI/")) && strings.Contains(filter, "\\/") {
			filter = strings.ReplaceAll(filter, "\\/", "%2F")
			paths := strings.Split(filter, "/")

			for pathIndex, path := range paths {
				if path == "*" || path == "**" {
					continue
				}

				path, err := url.QueryUnescape(path)
				if err != nil {
					break
				}

				paths[pathIndex] = url.QueryEscape(path)
			}

			filters[index] = strings.Join(paths, "/")
		}
	}
}

func parsePublicationFilter(jsonData []byte) []string {
	Logf("[get publication filter] parse publication filter JSON %s", jsonData)

	//nolint:exhaustivestruct
	filter := ExtractFiler{}

	err := json.Unmarshal(jsonData, &filter)
	if err != nil {
		Errorf("%v", err)

		return nil
	}

	return filter.Filters
}

func Outf(format string, args ...interface{}) {
	Out(fmt.Sprintf(format, args...))
}

func main() {
}

// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
