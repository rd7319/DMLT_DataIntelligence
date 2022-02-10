// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

//nolint:misspell
//example:
/*const exampleJSON = `{
	"INTERNALID" : "ER9_TEST",
	"HOST" : "ldcier9",
	"IDENTIFICATIONKEY":"ER9.001.0020270862.J1308266800",
	"OBJECTS":[
	   {
		  "OBJECTNAME":"ABAPTREE",
		  "OBJECTDESCR":"Obsolete Table",
		  "OBJECTTYPE":"TABLE",
		  "OBJECTPATH":"/TABLES/BC/ABAPTREE",
		  "OBJECTPARENT":"/TABLES/BC",
		  "CONTFLAG":"S",
		  "TABART":"APPL0",
		  "CONTENTTYPE":"SYSTEM",
		  "DATACLASS":"MASTER",
		  "PACKAGE":"S_SENSELESS_AND_OBSOLETE_OBJS",
		  "PACKAGEDESCR":"Senseless and Obsolete Objects",
		  "CLIDEP":"",
		  "COMPONENT":"BC",
		  "COMPDESCR":"Basis Components",
		  "CDS_PUBLISHED":"",
		  "CDS_TYPE":"",
		  "BASETABLE":"",
		  "LASTCHANGE":"20110909200218",
		  "COLUMNS":[
			 {
				"COLUMNNAME":"ID",
				"COLUMNDESC":"Demo hierarchy: Node ID",
				"IS_KEY":"X",
				"DATAELEMENT":"SEU_DEMID",
				"DOMAIN":"SEU_DEMID",
				"ABAPTYPE":"NUMC",
				"ABAPLEN":"000006",
				"ABAPDEC":"000000",
				"OUTPUTLEN":"000006",
				"REFTABLE":"",
				"REFFIELD":"",
				"VALUETABLE":"*"
			 },
			 {
				"COLUMNNAME":"REPNAME",
				"COLUMNDESC":"ABAP Program Name",
				"IS_KEY":"",
				"DATAELEMENT":"REPID",
				"DOMAIN":"PROGNAME",
				"ABAPTYPE":"CHAR",
				"ABAPLEN":"000040",
				"ABAPDEC":"000000",
				"OUTPUTLEN":"000040",
				"REFTABLE":"",
				"REFFIELD":"",
				"VALUETABLE":""
			 }
		  ],
		  "INDICES":[

		  ]
	   }
	]
 }`*/

//nolint:gochecknoglobals
var (
	GetString      func(string) string
	GetInt         func(string) int
	Log            func(string)
	Logf           func(string, ...interface{})
	Errorf         func(string, ...interface{})
	Out            func(interface{})
	UpdateProgress func(interface{})
)

// data structure from ABAP system.
type Datasets struct {
	Host              string         `json:"HOST"`
	IdentificationKey string         `json:"IDENTIFICATIONKEY"`
	Version           string         `json:"VERSION"`
	Data              []Dataset      `json:"OBJECTS"`
	Errors            []DatasetError `json:"ERRORS"`
}

type DatasetError struct {
	QualifiedName string   `json:"OBJECTPATH"`
	Messages      []string `json:"MESSAGES"`
}

type Dataset struct {
	Qualifiedname          string                  `json:"OBJECTPATH"`
	Name                   string                  `json:"OBJECTNAME"`
	Version                string                  `json:"LASTCHANGE"`
	Parent                 string                  `json:"OBJECTPARENT"`
	Type                   string                  `json:"OBJECTTYPE"`
	Package                string                  `json:"PACKAGE"`
	PackageDescription     string                  `json:"PACKAGEDESCR"`
	AppComponent           string                  `json:"COMPONENT"`
	AppComponentDesc       string                  `json:"COMPDESCR"`
	DataClass              string                  `json:"DATACLASS"`
	Description            string                  `json:"OBJECTDESCR"`
	Properties             []Property              `json:"PROPERTIES"`
	DatasetTableattributes []DatasetTableAttribute `json:"COLUMNS"`
}

type DatasetTableAttribute struct {
	Columnname  string     `json:"COLUMNNAME"`
	Columndesc  string     `json:"COLUMNDESC"`
	IsKey       string     `json:"IS_KEY"`
	DataElement string     `json:"DATAELEMENT"`
	Domain      string     `json:"DOMAIN"`
	Abaptype    string     `json:"ABAPTYPE"`
	Abaplen     string     `json:"ABAPLEN"`
	Abapdec     string     `json:"ABAPDEC"`
	Outputlen   string     `json:"OUTPUTLEN"`
	Properties  []Property `json:"PROPERTIES"`
}

// data structure for extraction.
type Annotation struct {
	Value  string `json:"value"`
	Locale string `json:"locale"`
	Type   string `json:"type"`
}

type Container struct {
	Qualifiedname string `json:"qualifiedName"`
	DisplayName   string `json:"displayName"`
}

type ExtractedSystem struct {
	InternalID       string           `json:"internalId"`
	Type             string           `json:"type"`
	SystemIdentifier SystemIdentifier `json:"systemIdentifier"`
}

/*"DataTypeEnum": {
	"description": "the attribute type in the BDH type system.",
	"type": "string",
	"enum": [
		"BINARY",
		"BOOLEAN",
		"DATE",
		"DATETIME",
		"TIME",
		"INTEGER",
		"FLOATING",
		"DECIMAL",
		"STRING",
		"LARGE_BINARY_OBJECT",
		"LARGE_CHARACTER_OBJECT",
		"UNKNOWN"
	]
},*/

type Attribute struct {
	Name           string       `json:"name"`
	Datatype       string       `json:"datatype"`
	Length         int          `json:"length"`
	NativeDatatype string       `json:"nativeDatatype"`
	Scale          int          `json:"scale"`
	Properties     []Property   `json:"properties"`
	Annotations    []Annotation `json:"annotations"`
}

type UniqueKey struct {
	AttributeReferences []string `json:"attributeReferences"`
}

type Table struct {
	UniqueKeys  []UniqueKey  `json:"uniqueKeys"`
	Attributes  []Attribute  `json:"attributes"`
	Properties  []Property   `json:"properties"`
	Annotations []Annotation `json:"annotations"`
}

type DatasetSchema struct {
	GenericType string `json:"genericType"`
	Tables      Table  `json:"tables"`
}

type SystemIdentifier struct {
	Host              string `json:"host"`
	IdentificationKey string `json:"identificationKey"`
}

type Property struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Value     string `json:"value"`
}

type ReferencedObject struct {
	Container Container `json:"container"`
}

type RuntimeObject struct {
	QualifiedName       string `json:"qualifiedName"`
	Name                string `json:"name"`
	NativeQualifiedName string `json:"nativeQualifiedName"`
	RemoteObjectType    string `json:"remoteObjectType"`
	// Owner                string        `json:"owner"`
	NativeVersion        string             `json:"nativeVersion"`
	VersionNumber        int                `json:"versionNumber"`
	MetadataLastModified int64              `json:"metadataLastModified"`
	DataLastModified     int                `json:"dataLastModified"`
	ReferencedObjects    []ReferencedObject `json:"referencedObjects"`
	Properties           []Property         `json:"properties"`
	Annotations          []Annotation       `json:"annotations"`
	DatasetSchema        DatasetSchema      `json:"datasetSchema"`
}

type Extraction struct {
	Schema string `json:"$schema"`
	// ExtractionDate  string          `json:"extractionDate"`
	ExtractedSystem ExtractedSystem         `json:"extractedSystem"`
	RuntimeObjects  []RuntimeObject         `json:"runtimeObjects"`
	Containers      []Container             `json:"containers"`
	Connection      string                  `json:"connection"`
	Publication     string                  `json:"publication"`
	Errors          []ExtractionErrorReport `json:"errors"`
}

type ExtractionContainer struct {
	Extraction Extraction `json:"extraction"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ExtractionError struct {
	Message    string `json:"message"`
	Module     string `json:"module"`
	StatusCode int    `json:"statusCode"`
}

type ExtractionErrorReport struct {
	QualifiedNames []string        `json:"qualifiedNames"`
	Type           string          `json:"type"`
	Error          ExtractionError `json:"error"`
}

type GraphFatalErrorCause struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type GraphFatalError struct {
	Message    string                 `json:"message"`
	Module     string                 `json:"module"`
	StatusCode int                    `json:"statusCode"`
	Causes     []GraphFatalErrorCause `json:"causes"`
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// #nosec G101
const (
	nameSpace                    = "com.sap.abap"
	defaultChunkSize             = 20
	defaultResultSendingInterval = 50
	inputDataTimeOut             = 15 * time.Minute
)

//nolint:gochecknoglobals
var (
	chunkSize             int
	resultSendingInterval int
	//nolint:deadcode
	Timer                  = inputDataTimeOut
	inportDataReceivedTime time.Time
	//nolint:exhaustivestruct
	httpClient HTTPClient = &http.Client{}
)

//nolint:deadcode
func Setup() {
	inportDataReceivedTime = time.Now()
}

func Tick() {
	ctx := context.TODO()
	currentTime := time.Now()
	elapsedTime := currentTime.Sub(inportDataReceivedTime)

	Log("[Update Publication] Tick start to check whether no input data during 15 minutes")

	if elapsedTime >= inputDataTimeOut {
		Log("[Update Publication] no input data for 15 minutes, stop the graph")

		fatalURL := GetString("fatalErrorUrl")

		err := sendFatalError(ctx, fatalURL,
			"No data from ABAP system for 15 minutes, "+
				"please refer SAP note 2890171 and make sure all the necessary notes have been applied",
		)
		if err != nil {
			Errorf("[update publication] failed to send fatal error message %v", err)
		}

		// send signal to graph terminator
		Out("{}")
	}
}

// In handle the port 'in' value.
//nolint:funlen
func In(in interface{}) {
	inportDataReceivedTime = time.Now()
	ctx := context.TODO()

	//nolint:forcetypeassert
	ins := in.(string)

	var datasets Datasets

	err := json.Unmarshal([]byte(ins), &datasets)
	if err != nil {
		Errorf("[update publication] error on unmarshal metadata detail JSON %w", err)

		return
	}

	chunkSize = GetInt("updateChunkSize")
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}

	resultSendingInterval = GetInt("updateInterval")
	if resultSendingInterval <= 0 {
		resultSendingInterval = defaultResultSendingInterval
	}

	successedCount := len(datasets.Data)
	failedCount := len(datasets.Errors)
	Logf("[Update Publication] successed count: %d, failed count: %d", successedCount, failedCount)

	pageCount := successedCount / chunkSize
	leftOverCount := successedCount % chunkSize

	for i, start := 0, 0; i < pageCount; i, start = i+1, start+chunkSize {
		Logf("[Update Publication] sendResultByChunk: %d", chunkSize)

		err = sendResultByChunk(ctx,
			datasets.Data[start:start+chunkSize], datasets.Host, datasets.IdentificationKey, datasets.Version)
		if err != nil {
			Errorf("[update publication] failed to send chunk: %w", err)

			return
		}

		time.Sleep(time.Duration(resultSendingInterval) * time.Millisecond)
	}

	if leftOverCount > 0 {
		Logf("[Update Publication] sendResultByChunk: %d", leftOverCount)

		err = sendResultByChunk(ctx,
			datasets.Data[successedCount-leftOverCount:], datasets.Host, datasets.IdentificationKey, datasets.Version)
		if err != nil {
			Errorf("[update publication] failed to send chunk: %w", err)

			return
		}
	}

	err = sendExtractionErrors(ctx, datasets.Host, datasets.IdentificationKey, datasets.Errors)
	if err != nil {
		Errorf("[update publication] failed to send extraction error: %w", err)

		return
	}

	Logf("[Update Publication] UpdateProgress: %d", successedCount+failedCount)
	UpdateProgress(int64(successedCount + failedCount))
	Log("[Update Publication] UpdateProgress: finish")
}

// End handle the port 'in' value.
//nolint:unparam
func End(in interface{}) {
	ctx := context.TODO()
	url := GetString("url")

	Log("[update publication] send signal to metadata service")

	if err := sendEndSignal(ctx, url); err != nil {
		Errorf("[update publication] failed to send end of Extraction JSON %w", err)
	}

	Log("[update publication] send signal to graph terminator")
	// send signal to graph terminator
	Out("{}")
}

func postRequest(ctx context.Context, url string, content []byte) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(content))
	if err != nil {
		return nil, fmt.Errorf("falied to create new request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		Errorf("postRequest-> failed to do request %w", err)

		return nil, err
	}

	return response, nil
}

func sendExtraction(ctx context.Context, url string, extractionContainer *ExtractionContainer) error {
	jsonString, err := json.Marshal(*extractionContainer)
	if err != nil {
		Errorf("[update publication] failed to create ExtractionContainer JSON %w", err)

		return err
	}

	response, err := postRequest(ctx, url, jsonString)
	if err != nil {
		Errorf("[update publication] failed to send Extraction JSON %w", err)

		return err
	}

	if !isSuccessful(response) {
		return extractionError(response)
	}

	return nil
}

func isSuccessful(response *http.Response) bool {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return true
	}

	return false
}

func extractionError(response *http.Response) error {
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to send Extraction %w", err)
	}

	var errorResponse ErrorResponse

	err = json.Unmarshal(body, &errorResponse)
	if err != nil {
		return fmt.Errorf("failed to Unmarshal errorResponse %w", err)
	}

	//nolint:goerr113
	return fmt.Errorf("failed to send Extraction: %s", errorResponse.Message)
}

func sendEndSignal(ctx context.Context, url string) error {
	endExtractionSignal := []byte("{}")

	response, err := postRequest(ctx, url, endExtractionSignal)
	if err != nil {
		Errorf("failed to send Extraction JSON %w", err)

		return err
	}

	if !isSuccessful(response) {
		return extractionError(response)
	}

	return nil
}

//nolint:goconst
func getBDHDataType(nativeType string) string {
	var BDHType string

	switch nativeType {
	case "INT1", "INT2", "INT4", "INT8", "PREC":
		BDHType = "INTEGER"
	case "DEC", "CURR", "QUAN", "FLTP", "DECFLOAT16", "D16N", "DECFLOAT34", "D34N", "DF16_DEC", "D16D":
		BDHType = "DECIMAL"
	case "DF16_RAW", "D16R", "DF16_SCL", "D16S", "DF34_DEC", "D34D", "DF34_RAW", "D34R", "DF34_SCL", "D34S":
		BDHType = "DECIMAL"
	case "RAWSTRING", "LRAW", "RAW", "GEOM_EWKB", "GGM1", "SRAWSTRING", "SRST", "RSTR":
		BDHType = "BINARY"
	case "BOOLEAN":
		BDHType = "BOOLEAN"
	case "CHAR", "LCHR", "SSTRING", "SSTR", "ACCP", "NUMC", "CLNT", "LANG", "CUKY", "UNIT", "VARC", "STRING", "STRG":
		BDHType = "STRING"
	case "DATS", "DATN":
		BDHType = "DATE"
	case "TIMS", "TIMN":
		BDHType = "TIME"
	case "TIMESTAMP", "UTCLONG", "UTCL":
		BDHType = "DATETIME"
	default:
		BDHType = "UNKNOWN"
	}

	return BDHType
}

func getRemoteObjectType(objectType string) string {
	var remoteObjectType string

	switch objectType {
	case "CDS":
		remoteObjectType = "VIEW.CDS"
	case "EXTRACTOR":
		remoteObjectType = "EXTRACTOR"
	case "VIEW":
		remoteObjectType = "VIEW"
	case "TABLE":
		fallthrough
	default:
		remoteObjectType = "TABLE"
	}

	return remoteObjectType
}

func convertObjectVersionToMilliseconds(verson string) (int64, error) {
	year, err := strconv.Atoi(verson[0:4])
	if err != nil {
		return 0, err
	}

	month, err := strconv.Atoi(verson[4:6])
	if err != nil {
		return 0, err
	}

	date, err := strconv.Atoi(verson[6:8])
	if err != nil {
		return 0, err
	}

	hour, err := strconv.Atoi(verson[8:10])
	if err != nil {
		return 0, err
	}

	minute, err := strconv.Atoi(verson[10:12])
	if err != nil {
		return 0, err
	}

	seconds, err := strconv.Atoi(verson[12:14])
	if err != nil {
		return 0, err
	}

	lastModifiedDate := time.Date(year, time.Month(month), date, hour, minute, seconds, 0, time.UTC)

	const nanoToMilli = 1_000_000

	return lastModifiedDate.UnixNano() / nanoToMilli, nil
}

func getParentPath(path string) string {
	index := strings.LastIndex(path, "/")
	if index == -1 {
		return "/"
	}

	return path[0:index]
}

func getDisplayName(name string) string {
	index := strings.LastIndex(name, "-")
	if index == -1 {
		return ""
	}

	return name[0:index]
}

func getFolderDisplayName(runtimeObjects []RuntimeObject) []Container {
	var containers []Container

	var appComponentName string

	pathMap := map[string]bool{}

	for _, runtimeObject := range runtimeObjects {
		for _, property := range runtimeObject.Properties {
			if property.Name == "applicationComponent" {
				appComponentName = property.Value

				break
			}
		}

		if appComponentName == "" {
			continue
		}

		parentPath := path.Dir(runtimeObject.QualifiedName)

		if _, exist := pathMap[parentPath]; !exist {
			newContainer := Container{
				Qualifiedname: parentPath,
				DisplayName:   appComponentName,
			}
			containers = append(containers, newContainer)
			pathMap[parentPath] = true

			for newParentPath, newDisplayName := parentPath, appComponentName; newParentPath != "/"; {
				newParentPath = getParentPath(newParentPath)
				newDisplayName = getDisplayName(newDisplayName)

				if newDisplayName == "" {
					break
				}

				if _, exist = pathMap[newParentPath]; !exist {
					newContainer = Container{
						Qualifiedname: newParentPath,
						DisplayName:   newDisplayName,
					}

					containers = append(containers, newContainer)
					pathMap[newParentPath] = true
				}
			}
		}
	}

	return containers
}

func sendResultByChunk(ctx context.Context, datasets []Dataset, host, identificationKey, version string) error {
	url := GetString("url")

	runtimeObjects := make([]RuntimeObject, 0, len(datasets))

	for _, dataset := range datasets {
		runtimeObject, err := generateRuntimeObject(dataset, version)
		if err != nil {
			Errorf("[update publication] failed to generate run time object %w", err)

			continue
		}

		runtimeObjects = append(runtimeObjects, *runtimeObject)
	}

	systemIdentifier := SystemIdentifier{
		Host:              host,
		IdentificationKey: identificationKey,
	}

	connectionID := GetString("connectionID")

	extractedSystem := ExtractedSystem{
		InternalID:       connectionID,
		SystemIdentifier: systemIdentifier,
		Type:             "ABAP",
	}

	publicationID := GetString("publicationID")

	//nolint:exhaustivestruct
	extraction := Extraction{
		Schema:          "http://sap.com/dh/metadata/2.3.0/extractionModel",
		ExtractedSystem: extractedSystem,
		RuntimeObjects:  runtimeObjects,
		Containers:      getFolderDisplayName(runtimeObjects),
		Connection:      connectionID,
		Publication:     publicationID,
	}

	extractionContainer := ExtractionContainer{
		Extraction: extraction,
	}

	err := sendExtraction(ctx, url, &extractionContainer)
	if err != nil {
		Errorf("[update publication] failed to send extraction %v", err)

		return err
	}

	return nil
}

func sendExtractionErrors(ctx context.Context, host, identificationKey string, datasetErrors []DatasetError) error {
	if len(datasetErrors) > 0 {
		url := GetString("url")

		systemIdentifier := SystemIdentifier{
			Host:              host,
			IdentificationKey: identificationKey,
		}

		connectionID := GetString("connectionID")

		extractedSystem := ExtractedSystem{
			InternalID:       connectionID,
			SystemIdentifier: systemIdentifier,
			Type:             "ABAP",
		}

		publicationID := GetString("publicationID")

		var errorReports []ExtractionErrorReport

		for _, datasetError := range datasetErrors {
			//nolint:exhaustivestruct
			errorReport := ExtractionErrorReport{
				Type:           "METADATA_EXTRACTION",
				QualifiedNames: []string{datasetError.QualifiedName},
				Error: ExtractionError{
					Module:     "ABAP metadata extraction",
					StatusCode: http.StatusInternalServerError,
				},
			}

			if len(datasetError.Messages) > 0 {
				errorReport.Error.Message = datasetError.Messages[0]
			}

			errorReports = append(errorReports, errorReport)
		}

		//nolint:exhaustivestruct
		extraction := Extraction{
			Schema:          "http://sap.com/dh/metadata/2.3.0/extractionModel",
			ExtractedSystem: extractedSystem,
			Connection:      connectionID,
			Publication:     publicationID,
			Errors:          errorReports,
		}

		extractionContainer := ExtractionContainer{
			Extraction: extraction,
		}

		err := sendExtraction(ctx, url, &extractionContainer)
		if err != nil {
			return err
		}
	}

	return nil
}

func sendFatalError(ctx context.Context, url, fatalMessage string) error {
	//nolint:exhaustivestruct
	fatalError := GraphFatalError{
		Message:    "Error occurs during ABAP metadata extraction",
		Module:     "ABAP metadata extraction",
		StatusCode: http.StatusInternalServerError,
	}

	var detailErrors []GraphFatalErrorCause

	detailError := GraphFatalErrorCause{
		Code:    "500",
		Message: fatalMessage,
	}

	detailErrors = append(detailErrors, detailError)
	fatalError.Causes = detailErrors

	fatalErrorResult, err := json.Marshal(fatalError)
	if err != nil {
		Errorf("[update publication] failed to create fatal errors result %v", err)

		return err
	}

	response, err := postRequest(ctx, url, fatalErrorResult)
	if err != nil {
		Errorf("[update publication] failed to send fatal errors %w", err)

		return err
	}

	if !isSuccessful(response) {
		return extractionError(response)
	}

	return nil
}

func generateAttributes(dataset Dataset, version string) []Attribute {
	attributes := make([]Attribute, 0, len(dataset.DatasetTableattributes))

	for _, datasetAtrribute := range dataset.DatasetTableattributes {
		if len(datasetAtrribute.Columnname) == 0 {
			Errorf("[update publication] skip empty column name of %s", dataset.Qualifiedname)

			continue
		}

		//nolint:exhaustivestruct
		attribute := Attribute{
			Name:           datasetAtrribute.Columnname,
			Datatype:       getBDHDataType(datasetAtrribute.Abaptype),
			NativeDatatype: datasetAtrribute.DataElement,

			Annotations: []Annotation{},
			Properties: []Property{
				{
					Name:      "domain",
					Namespace: nameSpace,
					Value:     datasetAtrribute.Domain,
				},
				{
					Name:      "abapType",
					Namespace: nameSpace,
					Value:     datasetAtrribute.Abaptype,
				},
			},
		}

		if version == "v2" {
			attribute.Properties = append(attribute.Properties, datasetAtrribute.Properties...)
		}

		redefineAttribute(&attribute)

		attributes = append(attributes, attribute)
	}

	return attributes
}

func redefineAttribute(attribute *Attribute) {
	for _, property := range attribute.Properties {
		if property.Name == "semanticType" {
			semanticType := strings.ToUpper(property.Value)
			switch semanticType {
			case "TIMESTAMP":
				attribute.Datatype = "DATETIME"
			case "BOOLEAN":
				attribute.Datatype = "STRING"
				attribute.Length = 1
			case "UUID":
				attribute.Datatype = "STRING"
				attribute.Length = 36
			}

			break
		}
	}
}

func generateRuntimeObjectProperties(dataset Dataset, version string) []Property {
	var runtimeObjectProperties []Property

	objectTypeProperty := Property{
		Name:      "objectType",
		Namespace: nameSpace,
		Value:     dataset.Type,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, objectTypeProperty)

	packageProperty := Property{
		Name:      "package",
		Namespace: nameSpace,
		Value:     dataset.Package,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, packageProperty)

	packageDescriptionProperty := Property{
		Name:      "packageDescription",
		Namespace: nameSpace,
		Value:     dataset.PackageDescription,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, packageDescriptionProperty)

	appComponentProperty := Property{
		Name:      "applicationComponent",
		Namespace: nameSpace,
		Value:     dataset.AppComponent,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, appComponentProperty)

	appComponentDescProperty := Property{
		Name:      "applicationComponentDescription",
		Namespace: nameSpace,
		Value:     dataset.AppComponentDesc,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, appComponentDescProperty)

	dataClassProperty := Property{
		Name:      "dataClass",
		Namespace: nameSpace,
		Value:     dataset.DataClass,
	}

	runtimeObjectProperties = append(runtimeObjectProperties, dataClassProperty)

	if version == "v2" {
		runtimeObjectProperties = append(runtimeObjectProperties, dataset.Properties...)
	}

	return runtimeObjectProperties
}

func generateRuntimeObject(dataset Dataset, version string) (*RuntimeObject, error) {
	versionNumber, err := strconv.Atoi(dataset.Version)
	if err != nil {
		return nil, err
	}

	lastModified, err := convertObjectVersionToMilliseconds(dataset.Version)
	if err != nil {
		return nil, err
	}

	tables := Table{
		Attributes: generateAttributes(dataset, version),
		// consider to add uniquekey
		UniqueKeys:  []UniqueKey{},
		Annotations: []Annotation{},
		Properties:  []Property{},
	}

	//nolint:lll
	// https://github.wdf.sap.corp/bdh/datahub-app-base/blob/master/src/apps/dh-app-metadata/spec/schemas/ExtractionModel_V1_1_0.json
	// ObjectGenericTypeEnum: "TABLE", "NESTED", "GRAPH", "BUSINESS_OBJECT"
	datasetSchema := DatasetSchema{
		GenericType: "TABLE",
		Tables:      tables,
	}

	var runtimeObjectAnnotations []Annotation

	objectDescriptionAnnotation := Annotation{
		Locale: "en",
		Type:   "SHORT",
		Value:  dataset.Description,
	}

	runtimeObjectAnnotations = append(runtimeObjectAnnotations, objectDescriptionAnnotation)

	//nolint:exhaustivestruct
	runtimeObject := RuntimeObject{
		QualifiedName:       dataset.Qualifiedname,
		Name:                dataset.Name,
		NativeQualifiedName: fmt.Sprintf("%s.%s", dataset.Type, dataset.Name),
		RemoteObjectType:    getRemoteObjectType(dataset.Type),
		// Owner:                "SYSTEM",
		NativeVersion:        dataset.Version,
		VersionNumber:        versionNumber,
		MetadataLastModified: lastModified,
		DatasetSchema:        datasetSchema,
		Properties:           generateRuntimeObjectProperties(dataset, version),
		Annotations:          runtimeObjectAnnotations,
	}

	return &runtimeObject, nil
}

func main() {
}

// © 2019-2021 SAP SE or an SAP affiliate company. All rights reserved.
