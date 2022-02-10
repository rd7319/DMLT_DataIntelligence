package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"di.sap.com/ml/modules/message"
	"di.sap.com/ml/modules/mlclient"
	"di.sap.com/ml/vsystemclient"
)

func Setup() {
	gVsystemInfo.Endpoint = vsystemclient.GetvSystemEndpoint()
	Logf("%s: %s: using vSystemEndpoint: %q", getOperatorName(), "Setup", gVsystemInfo.Endpoint)
	if err := CheckMandatoryUIParameter(); nil != err {
		ProcessErrorSetup("MandatoryParameters", err)
		return
	}
	gLockInArtifact = &sync.Mutex{}
}

func InArtifact(msg interface{}) {
	Logf("%s: incoming message to %s", getOperatorName(), "InArtifact")
	lockingHandler, msg, err := AppendFileAttributesToMessage(msg)
	if nil != err {
		ProcessErrorInArtifact("dispatch", err)
		return
	}
	lockingHandler(msg)
}

func InFileReturn(msg interface{}) {
	Logf("%s: %s: incoming file message", getOperatorName(), "InFileReturn")
	unLockInArtifact := Dispatcher{}.MessageIsSingleFileUpload(msg)
	defer endingTransaction(unLockInArtifact)

	// Assumption:
	// WriteFile blocks until batch with lastBatch==true

	ID, name, kind, err := RegisterArtifactFromFileMessage(msg, gVsystemInfo.Endpoint)
	if nil != err {
		ProcessErrorInArtifact("InFileReturn", err)
		return
	}
	AppendArtifactAttributesToMessage(msg, ID, name, kind)

	CleanUpAttributes(msg)

	RemoveBodyFromMessage(msg)

	if OutArtifact != nil {
		OutArtifact(msg)
	}
}

func AppendFileAttributesToMessage(msg interface{}) (func(interface{}), interface{}, error) {
	scenarioHandler, lockHandler, err := Dispatcher{}.DispatchMessage(msg)
	if nil != err {
		return func(interface{}) {}, nil, err
	}
	Logf("%v: message dispatched to %s", getOperatorName(), nameOf(scenarioHandler))

	msg, err = scenarioHandler(msg)
	if nil != err {
		return func(interface{}) {}, nil, err
	}
	return lockHandler, msg, nil
}

type Scenarios struct{}

func (s Scenarios) AppendFileAttributeToMessage(msg interface{}, filePath string) (interface{}, error) {
	copiedMsgRef, fileMap, err := CopyFileMessage(msg)
	if nil != err {
		return nil, err
	}
	message.AppendFilePathAndConnection(fileMap, filePath, GetStorageConnectionID())

	// remove additional elements
	connection := fileMap["connection"]
	connectionMap := connection.(map[string]interface{})
	delete(connectionMap, "vrepRoot")

	Logf("%s: appending to Attributes:\n%q", getOperatorName(), fileMap)

	if OutFileSend == nil {
		return nil, errors.New("OutFileSend: make sure that a Write File operator is connected")
	}

	return copiedMsgRef, nil
}

func (s Scenarios) UploadingSingleFile(msg interface{}) (interface{}, error) {
	startingTransaction()
	gUniqueIdentifier = GetDefaultUniqueIdentifier()
	filePath := createArtifactURI(GetGraphHandle(), gUniqueIdentifier)
	return s.AppendFileAttributeToMessage(msg, filePath)
}

func (s Scenarios) UploadingSingleFileWithBatches(msg interface{}) (interface{}, error) {
	// Assumption: Same message as long as lastBatch was processed
	if !gLockUnfinishedBatching {
		gUniqueIdentifier = GetDefaultUniqueIdentifier()
	}
	gLockUnfinishedBatching = true
	defer resetLockAfterLastBatch(msg, &gLockUnfinishedBatching)
	// new or existing transaction
	filePath := createArtifactURI(GetGraphHandle(), gUniqueIdentifier)
	return s.AppendFileAttributeToMessage(msg, filePath)
}

func (s Scenarios) RegisteringPathToConnectionID(msg interface{}) (interface{}, error) {
	// No interaction with WriteFile is needed
	InFileReturn(msg)
	return msg, nil
}

type Dispatcher struct{}

func (d Dispatcher) DispatchMessage(msg interface{}) (func(interface{}) (interface{}, error), func(interface{}), error) {
	// Dispatcher depends directly on the scenarios
	if d.MessageIsSingleFileUpload(msg) {
		return Scenarios{}.UploadingSingleFile, finishTransaction, nil
	}
	if d.MessageIsSingleFileUploadInBatches(msg) {
		return Scenarios{}.UploadingSingleFileWithBatches, OutFileSend, nil
	}
	if d.MessageIsRegisterPathToConnectionID(msg) {
		return Scenarios{}.RegisteringPathToConnectionID, func(interface{}) {}, nil
	}
	return nil, func(interface{}) {}, DispatcherError()
}

func DispatcherError() error {
	return errors.New("unsupported message format")
}

func (d Dispatcher) MessageIsSingleFileUpload(m interface{}) bool {
	bodyReader, err := message.GetBodyAsBlob(m)
	if nil != err {
		Logf("%v: found error in GetBodyAsBlob: %v", getOperatorName(), err)
		return false
	}
	return bodyReader.Len() != 0 && !message.FoundMandatoryBatchingAttributes(m)
}

func (d Dispatcher) MessageIsSingleFileUploadInBatches(m interface{}) bool {
	bodyReader, err := message.GetBodyAsBlob(m)
	if nil != err {
		return false
	}
	return bodyReader.Len() != 0 && message.FoundMandatoryBatchingAttributes(m)
}

func (d Dispatcher) MessageIsRegisterPathToConnectionID(m interface{}) bool {
	bodyReader, err := message.GetBodyAsBlob(m)
	if nil != err {
		return false
	}
	return bodyReader.Len() == 0 && message.FoundMandatoryFileAttributes(m)
}

func CopyFileMessage(msg interface{}) (interface{}, map[string]interface{}, error) {
	copiedMsgMap, err := CopyMessage(msg.(map[string]interface{}))
	if err != nil {
		return nil, nil, err
	}
	attributeRef, err := message.GetAttributes(copiedMsgMap)
	if nil != err {
		return nil, nil, err
	}
	fileRef, err := message.GetFileAttributes(copiedMsgMap)
	if nil == err {
		newFileRef := message.ShallowCopyMap(fileRef)
		connectionRef := newFileRef["connection"].(map[string]interface{})
		newConnectionRef := message.ShallowCopyMap(connectionRef)
		newFileRef["connection"] = newConnectionRef
		attributeRef["file"] = newFileRef
	} else {
		copiedMsgMap = message.AppendFileProtocolToMessage(copiedMsgMap).(map[string]interface{})
	}
	fileRef, err = message.GetFileAttributes(copiedMsgMap)
	if nil != err {
		return nil, nil, err
	}
	return copiedMsgMap, fileRef, nil
}

func RegisterArtifactFromFileMessage(msg interface{}, vsystemEndpoint string) (string, string, string, error) {
	connectionID, path := message.ReceiveFileInformationFromMessage(msg)
	Logf("%s: found connectionID: %q and path: %q", getOperatorName(), connectionID, path)
	if connectionID != GetStorageConnectionID() {
		return "", "", "", fmt.Errorf("only %s is supported", GetStorageConnectionID())
	}
	mlAPIEndpoint := mlclient.CreateArtifactEndpoint(vsystemEndpoint, "v1", "artifacts")
	requestData := NewRequestMetaData(connectionID, path)
	responseMetadata, err := RegisterArtifact(&requestData, mlAPIEndpoint)
	if nil != err {
		return "", "", "", err
	}
	Logf("%s: ArtifactResponseData:\n%v", getOperatorName(), *responseMetadata)
	return responseMetadata.ID, requestData.Name, requestData.Kind, nil
}

func AppendArtifactAttributesToMessage(msg interface{}, id string, name string, kind string) {
	artifactAttributes := CreateArtifactAttributes(id, name, kind)
	// structure tested before
	msg.(map[string]interface{})["Attributes"].(map[string]interface{})["artifact"] = artifactAttributes
}

func CleanUpAttributes(msg interface{}) {
	// structure tested before
	attributesRef := msg.(map[string]interface{})["Attributes"].(map[string]interface{})
	delete(attributesRef, "message.batchCount")
	delete(attributesRef, "message.batchIndex")
	delete(attributesRef, "message.batchSize")
	delete(attributesRef, "message.batchSizeUnit")
	delete(attributesRef, "message.lastBatch")
}

func RemoveBodyFromMessage(msg interface{}) {
	body, err := message.GetBody(msg)
	if nil == err {
		if len(body.([]byte)) != 0 {
			// release message["Body"]
			msg.(map[string]interface{})["Body"] = []byte{}
		}
	}
}

func CreateArtifactAttributes(artifactID string, artifactName string, artifactKind string) map[string]interface{} {
	headers := make(map[string]interface{}, 3)
	headers["version"] = 1
	headers["id"] = artifactID
	headers["name"] = artifactName
	headers["kind"] = artifactKind
	return headers
}

func resetLockAfterLastBatch(msgI interface{}, lockVar *bool) bool {
	message, ok := msgI.(map[string]interface{})
	if !ok {
		return false
	}
	attributesI, ok := message["Attributes"]
	if !ok {
		return false
	}
	attributes, ok := attributesI.(map[string]interface{})
	if !ok {
		return false
	}
	lastBatchI, ok := attributes["message.lastBatch"]
	if !ok {
		return false
	}
	lastBatchAttribute, ok := lastBatchI.(bool)
	if !ok {
		return false
	}
	if lastBatchAttribute {
		*lockVar = false
	}
	return true
}

func NewRequestMetaData(connectionID string, path string) mlclient.ArtifactPostRequestMetaData {
	return mlclient.ArtifactPostRequestMetaData{
		Name:        GetArtifactName(),
		Kind:        GetArtifactKind(),
		URI:         createArtifactFullURI(connectionID, path),
		Description: GetArtifactDescription(),
		Type:        "EXECUTION",
		ExecutionID: GetGraphHandle(),
	}
}

func RegisterArtifact(metaData *mlclient.ArtifactPostRequestMetaData, artifactsEndpoint string) (*mlclient.MLAPIMessage, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(&metaData); err != nil {
		return nil, err
	}
	Logf("%s: registering artifact to %q", getOperatorName(), artifactsEndpoint)

	m, err := mlclient.PostArtifactInformation(artifactsEndpoint, &buf, getMLAPITimeout())
	if nil != err {
		return nil, fmt.Errorf("PostApiArtifact: %v", err)
	}
	Logf("%s: artifact registered with ID %q", getOperatorName(), m.ID)
	return m, nil
}

func createArtifactFullURI(connectionID string, path string) string {
	if strings.HasPrefix(path, "/") {
		path = strings.TrimLeft(path, "/")
	}
	return fmt.Sprintf("dh-dl://%v/%v", connectionID, path)
}

func createArtifactURI(executionID string, uniqueIdentifier string) string {
	const fixedPath = "shared/ml/artifacts/executions"
	return fmt.Sprintf("%v/%v/%v", fixedPath, executionID, uniqueIdentifier)
}

func GetUniqueIdentifier(artifactPrefix string, operatorName string, counter string, artifactSuffix string) string {
	return fmt.Sprintf("%v%v_%v%v", artifactPrefix, operatorName, counter, artifactSuffix)
}

func GetDefaultUniqueIdentifier() string {
	// To make it usable with multiplicity of groups,
	// an additional parameter is required
	prefix := GetString("prefix")
	suffix := GetString("suffix")
	return GetUniqueIdentifier(prefix, GetString("processName"), fmt.Sprint(GetSingletonCounter()), suffix)
}

// can be extended later with string from UI
func GetStorageConnectionID() string {
	return "DI_DATA_LAKE"
}

func GetArtifactName() string {
	return GetString("artifactName")
}

func GetArtifactKind() string {
	return GetString("artifactKind")
}

func GetArtifactDescription() string {
	return GetString("description")
}

func CheckMandatoryUIParameter() error {
	const currentAPIVersion = "v1"

	var value string
	if err := CheckMandatoryParameter(&value, "artifactName", GetString); nil != err {
		return err
	}
	if err := CheckMandatoryParameter(&value, "artifactKind", GetString); nil != err {
		return err
	}
	if err := CheckArtifactKind(value); nil != err {
		return err
	}
	if err := CheckMandatoryParameter(&value, "apiVersion", GetString); nil != err {
		return err
	}
	if value != currentAPIVersion {
		return fmt.Errorf("apiVersion should be %q, got %q", currentAPIVersion, value)
	}
	return nil
}

func CheckArtifactKind(artifactKind string) (err error) {
	switch artifactKind {
	case
		"model",
		"dataset",
		"other":
		return nil
	}
	return fmt.Errorf("artifactKind should be %q, got %q", "{model,dataset,other}", artifactKind)
}

func CheckMandatoryParameter(mandatoryValue *string, key string, getFunc func(string) string) error {
	value := getFunc(key)
	if len(value) == 0 {
		return fmt.Errorf("mandatory parameter %q is not set", key)
	}
	*mandatoryValue = value
	return nil
}

func startingTransaction() {
	// locking only without batches
	gLockInArtifact.Lock()
	Logf("%s: %s is locked", getOperatorName(), "InArtifact")
}

func endingTransaction(shouldBeUnlocked bool) {
	if shouldBeUnlocked {
		gLockInArtifact.Unlock()
		Logf("%s: %s: Function is released", getOperatorName(), "InFileReturn")
	}
}

func finishTransaction(msg interface{}) {
	OutFileSend(msg)
	Logf("%s: %s: waiting for processing InFileReturn", getOperatorName(), "InArtifact")
	gLockInArtifact.Lock()
	gLockInArtifact.Unlock() //nolint:staticcheck
	Logf("%s: %s is unlocked", getOperatorName(), "InArtifact")
}

func GetSingletonCounter() uint64 {
	gOnce.Do(func() {
		gCounter = new(uint64)
	})

	gCounterLock.Lock()
	defer gCounterLock.Unlock()

	activeCounter := *gCounter
	*gCounter++
	return activeCounter
}

// this functions returns the function name and
// is used to log the dispatched message
func nameOf(f interface{}) string {
	v := reflect.ValueOf(f)
	if v.Kind() == reflect.Func {
		if rf := runtime.FuncForPC(v.Pointer()); rf != nil {
			s := strings.Split(rf.Name(), ".")
			return s[len(s)-1]
		}
	}
	return v.String()
}

func ProcessErrorSetup(operation string, err error) {
	gOperatorConfig.Operation = operation
	vsystemclient.ProcessError(err, Errorf, OutError, gOperatorConfig, GetString("processName"), "Setup")
}

func ProcessErrorInArtifact(operation string, err error) {
	gOperatorConfig.Operation = operation
	vsystemclient.ProcessError(err, Errorf, OutError, gOperatorConfig, GetString("processName"), "InArtifact")
}

var gOperatorConfig = vsystemclient.OperatorConfig{
	OperatorName: getOperatorName(),
	Operation:    "artifact production",
	OperatorPath: "com.sap.ml.artifact.producer.v2",
}

func getOperatorName() string {
	return "ArtifactProducer"
}

func getMLAPITimeout() time.Duration {
	return 15 * time.Second
}

var (
	gVsystemInfo vsystemclient.VSystemInfo

	gOnce        sync.Once
	gCounterLock = &sync.Mutex{}
	gCounter     *uint64

	gLockInArtifact         *sync.Mutex
	gLockUnfinishedBatching bool = false
	gUniqueIdentifier       string
)

var (
	CopyMessage    func(map[string]interface{}) (map[string]interface{}, error)
	Errorf         func(string, ...interface{})
	GetGraphHandle func() string
	GetString      func(string) string
	Logf           func(string, ...interface{})

	OutArtifact func(interface{})
	OutFileSend func(interface{})
	OutError    func(interface{})
)
