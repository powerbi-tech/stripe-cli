package p400

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stripe/stripe-cli/pkg/version"
)

// RabbitServicePayload represents the JSON shape of all request bodies when calling Rabbit Service
type RabbitServicePayload struct {
	ID            int         `json:"id"`
	Service       string      `json:"service"`
	Method        string      `json:"method"`
	Content       string      `json:"content"`
	SessionToken  string      `json:"session_token"`
	RequestType   string      `json:"request_type"`
	VersionInfo   versionInfo `json:"version_info"`
	ParentTraceID string      `json:"parent_trace_id"`
	DeviceInfo    DeviceInfo  `json:"device_info"`
}

// RabbitServiceResponse is the response body from the Rabbit Service call
type RabbitServiceResponse struct {
	Content string `json:"content"`
}

type versionInfo struct {
	ClientType    string `json:"client_type"`
	ClientVersion string `json:"client_version"`
}

func encodeRPCContent(rpcContent interface{}) (string, error) {
	jsonContent, err := json.Marshal(rpcContent)

	if err != nil {
		return "", err
	}

	contentBytes := []byte(string(jsonContent))
	encodedContent := base64.StdEncoding.EncodeToString(contentBytes)

	return encodedContent, nil
}

// CallRabbitService takes a TerminalSessionContext and method information and calls that Rabbit Service RPC method
func CallRabbitService(tsCtx TerminalSessionContext, method string, methodContent interface{}, methodResponse interface{}, parentTraceID string) error {
	encodedMethodContent, err := encodeRPCContent(methodContent)

	var result RabbitServiceResponse

	if err != nil {
		return err
	}

	payload := CreateRabbitServicePayload(method, encodedMethodContent, parentTraceID, tsCtx)
	formattedIP := strings.Join(strings.Split(tsCtx.IPAddress, "."), "-")

	rabbitServiceURL := fmt.Sprintf(readerURL, formattedIP)
	res, err := http.Post(rabbitServiceURL, "application/json", &payload)

	if err != nil {
		return err
	}

	json.NewDecoder(res.Body).Decode(&result)

	res.Body.Close()

	if result.Content != "" {
		decoded, err := base64.StdEncoding.DecodeString(result.Content)

		if err != nil {
			return err
		}

		json.Unmarshal(decoded, methodResponse)
	} else {
		return errors.New("could not decode Rabbit Service response - no content")
	}

	return nil
}

// CreateRabbitServicePayload serializes the required information into a JSON payload used in calls to RabbitService
func CreateRabbitServicePayload(method string, methodContent string, parentTraceID string, tsCtx TerminalSessionContext) bytes.Buffer {
	sec := time.Now().Unix() * 1000

	payload := &RabbitServicePayload{
		ID:           int(sec),
		Service:      "JackRabbitService",
		Method:       method,
		Content:      methodContent,
		SessionToken: tsCtx.SessionToken,
		RequestType:  "STANDARD",
		VersionInfo: versionInfo{
			ClientType:    "STRIPE_CLI",
			ClientVersion: version.Version,
		},
		ParentTraceID: parentTraceID,
		DeviceInfo:    tsCtx.DeviceInfo,
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.Encode(payload)

	return buf
}
