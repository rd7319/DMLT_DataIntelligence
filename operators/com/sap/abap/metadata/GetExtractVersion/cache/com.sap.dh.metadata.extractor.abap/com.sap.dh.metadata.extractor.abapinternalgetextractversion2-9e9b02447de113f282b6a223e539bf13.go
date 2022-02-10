// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

//nolint:gochecknoglobals
var (
	GetString func(string) string
	GetInt    func(string) int
	Log       func(string)
	Logf      func(string, ...interface{})
	Errorf    func(string, ...interface{})
	Out       func(interface{})
	End       func(interface{})
)

// #nosec G101
const (
	defaultPostObjectVersionChunkSize = 1000
	defaultChunkSize                  = 50
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

//nolint:gochecknoglobals
var (
	postObjectVersionChunkSize int
	chunkSize                  int
	totalProcessingCount       int

	errEmptyInputData = errors.New("empty input data")

	//nolint:exhaustivestruct
	httpClient HTTPClient = &http.Client{}
)

type ObjectToExtract struct {
	QualifiedName string `json:"qualifiedName"`
}

type ObjectsToExtract struct {
	Count   int               `json:"count"`
	Objects []ObjectToExtract `json:"objectsToExtract"`
}

type ObjectVersion struct {
	QualifiedName string            `json:"qualifiedName"`
	Version       ObjectVersionInfo `json:"versionInfo"`
}

type ObjectVersionInfo struct {
	VersionNumber int64 `json:"metadataVersionNumber"`
}

func postVersion(ctx context.Context, url string, objectsVersions []ObjectVersion) error {
	objectVersionList, err := json.Marshal(objectsVersions)
	if err != nil {
		return fmt.Errorf("failed to marshal objectsVersions: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(objectVersionList))
	if err != nil {
		return err
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 && response.StatusCode != 204 {
		//nolint:goerr113
		return fmt.Errorf("postVersion with error response code: %d", response.StatusCode)
	}

	return nil
}

func getExtractionList(ctx context.Context, extractObjectURL string, top, skip int) (int, *ObjectsToExtract, error) {
	extractObjectURI, err := url.Parse(extractObjectURL)
	if err != nil {
		return 0, nil, err
	}

	query := extractObjectURI.Query()
	query.Add("$top", strconv.Itoa(top))
	query.Add("$skip", strconv.Itoa(skip))
	query.Add("$count", "true")
	extractObjectURI.RawQuery = query.Encode()

	Logf("[Get extract version]getExtractionList from %s", extractObjectURI.String())

	request, err := http.NewRequestWithContext(ctx, "GET", extractObjectURI.String(), nil)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create new request: %w", err)
	}

	request.Header.Set("Origin", "http://vsystem-internal:8796")
	request.Header.Set("X-Requested-With", "Fetch")

	response, err := httpClient.Do(request)
	if err != nil {
		return 0, nil, err
	}

	if response.StatusCode != 200 && response.StatusCode != 204 {
		defer response.Body.Close()

		data, err := ioutil.ReadAll(response.Body)
		if err != nil {
			// more or less we need to show stats code
			data = []byte("")
		}

		//nolint:goerr113
		return 0, nil, fmt.Errorf("getExtractionList error response code: %d with %s",
			response.StatusCode, string(data))
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()

	var objectsToExtract ObjectsToExtract

	err = json.Unmarshal(body, &objectsToExtract)
	if err != nil {
		return 0, nil, err
	}

	return objectsToExtract.Count, &objectsToExtract, nil
}

func handleVersions(ctx context.Context, versions string) error {
	var objectstVersions []ObjectVersion

	err := json.Unmarshal([]byte(versions), &objectstVersions)
	if err != nil {
		return fmt.Errorf("failed to unmarshal input data: : %w", err)
	}

	totalPostObjectVersions := len(objectstVersions)

	Logf("[Get extract version] total post object versions %d", totalPostObjectVersions)

	if totalPostObjectVersions == 0 {
		return errEmptyInputData
	}

	pageCount := totalPostObjectVersions / postObjectVersionChunkSize
	leftOverCount := totalPostObjectVersions % postObjectVersionChunkSize

	objectVersionURL := GetString("urlPost")

	for i, start := 0, 0; i < pageCount; i, start = i+1, start+postObjectVersionChunkSize {
		Logf("[Get extract version]postVersion within %d objects", postObjectVersionChunkSize)

		err := postVersion(ctx, objectVersionURL, objectstVersions[start:start+postObjectVersionChunkSize])
		if err != nil {
			return fmt.Errorf("fail to post version: %w", err)
		}
	}

	if leftOverCount > 0 {
		Logf("[Get extract version]postVersion within %d left over objects", leftOverCount)

		err := postVersion(ctx, objectVersionURL, objectstVersions[totalPostObjectVersions-leftOverCount:])
		if err != nil {
			return fmt.Errorf("fail to post version: %w", err)
		}
	}

	return nil
}

func sendOutResults(extractObjects []ObjectToExtract) error {
	totalCount := len(extractObjects)

	pageCount := totalCount / chunkSize
	leftOverCount := totalCount % chunkSize

	for i, start := 0, 0; i < pageCount; i, start = i+1, start+chunkSize {
		Logf("[Get extract version] sendOutResult within %d objects", chunkSize)

		err := sendOutResult(extractObjects[start : start+chunkSize])
		if err != nil {
			return err
		}
	}

	if leftOverCount > 0 {
		Logf("[Get extract version] sendOutResult within %d left over objects", leftOverCount)

		err := sendOutResult(extractObjects[totalCount-leftOverCount:])
		if err != nil {
			return err
		}
	}

	return nil
}

func handleExtractObjects(ctx context.Context) error {
	objectsToExtractURL := GetString("urlGet")

	totalCount, _, err := getExtractionList(ctx, objectsToExtractURL, 1, 0)
	if err != nil {
		return err
	}

	Logf("[Get extract version] handleExtractObjects totalCount %d", totalCount)

	if totalCount == 0 {
		Log("[Get extract version] handleExtractObjects return empty list because of no update available")
		Out("[]")

		return nil
	}

	totalProcessingCount = totalCount

	pageCount := totalCount / postObjectVersionChunkSize
	leftOverCount := totalCount % postObjectVersionChunkSize

	Logf("[Get extract version] handleExtractObjects pageCount -> %d leftOverCount -> %d", pageCount, leftOverCount)

	for i := 0; i < pageCount; i++ {
		_, objectsToExtract, err :=
			getExtractionList(ctx, objectsToExtractURL, postObjectVersionChunkSize, postObjectVersionChunkSize*i)
		if err != nil {
			return err
		}

		Logf("[Get extract version] sendOutResults %d", len(objectsToExtract.Objects))

		err = sendOutResults(objectsToExtract.Objects)
		if err != nil {
			return err
		}
	}

	if leftOverCount > 0 {
		_, objectsToExtract, err := getExtractionList(ctx, objectsToExtractURL, leftOverCount, totalCount-leftOverCount)
		if err != nil {
			return err
		}

		Logf("[Get extract version] sendOutResults with left over %d", len(objectsToExtract.Objects))

		err = sendOutResults(objectsToExtract.Objects)
		if err != nil {
			return err
		}
	}

	return nil
}

// In handle the `in` port of the operator.
func In(in interface{}) {
	ctx := context.TODO()

	// ex:
	// `[{ "qualifiedName": "test",  "versionInfo": { "metadataVersionNumber": 1}}]`

	//nolint:forcetypeassert
	ins := in.(string)

	Logf("[Get extract version] input data %s", ins)

	postObjectVersionChunkSize = GetInt("urlPostChunkSize")
	if postObjectVersionChunkSize <= 0 {
		postObjectVersionChunkSize = defaultPostObjectVersionChunkSize
	}

	chunkSize = GetInt("outPortChunkSize")
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	err := handleVersions(ctx, ins)
	if err != nil {
		Errorf("[get extract version] fail to post version: %+v", err)
		Out("[]")

		return
	}

	err = handleExtractObjects(ctx)
	if err != nil {
		Errorf("[get extract version] fail to handle extract object: %+v", err)
		Out("[]")

		return
	}
}

// UpdateProgress is used update the input value .
func UpdateProgress(in interface{}) {
	//nolint:forcetypeassert
	newProcessedCount := in.(int64)

	Logf("[Get extract version]UpdateProgress: new processed: (%d) total processing count: (%d)",
		newProcessedCount, totalProcessingCount)

	totalProcessingCount -= int(newProcessedCount)

	if totalProcessingCount <= 0 {
		Log("[Get extract version]UpdateProgress: send end signal")
		End("{}")
	}
}

func sendOutResult(objects []ObjectToExtract) error {
	objectNameList := make([]string, 0, len(objects))
	for _, object := range objects {
		objectNameList = append(objectNameList, object.QualifiedName)
	}

	out, err := json.Marshal(objectNameList)
	if err != nil {
		return fmt.Errorf("failed to marshal extract object list %w", err)
	}

	Out(fmt.Sprintf("%s", out))
	Logf("[Get extract version]sendOutResult with %d object: %+v ", len(objectNameList), objectNameList)

	return nil
}

func main() {
}

// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
