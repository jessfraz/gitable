package airtable

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/fabioberger/airtable-go/utils"
)

const majorAPIVersion = 0
const retryDelayIfRateLimited = 5 * time.Second

var apiBaseURL = fmt.Sprintf("https://api.airtable.com/v%d", majorAPIVersion)

// Client exposes the interface for sending requests to the Airtable API. After instantiation, one
// can further configure the client by setting the `ShouldRetryIfRateLimited` property which defaults
// to true, or by setting `HTTPClient` to a custom *http.Client instance.
type Client struct {
	apiKey                   string
	baseID                   string
	ShouldRetryIfRateLimited bool
	HTTPClient               *http.Client
}

// New creates a new instance of the Airtable client
func New(apiKey, baseID string) (*Client, error) {
	if !utils.IsValidAPIKey(apiKey) {
		return nil, fmt.Errorf("invalid API Key encountered: %s", apiKey)
	}
	if !utils.IsValidBaseID(baseID) {
		return nil, fmt.Errorf("invalid base ID encountered: %s", baseID)
	}

	c := Client{
		apiKey: apiKey,
		baseID: baseID,
		ShouldRetryIfRateLimited: true,
		HTTPClient:               http.DefaultClient,
	}
	return &c, nil
}

type recordList struct {
	Records []interface{} `json:"records"`
	Offset  string        `json:"offset"`
}

// ListRecords returns a list of records from a given Airtable table. The caller can optionally pass in
// a ListParameters struct as the last argument. If passed, it will be url encoded and sent with the request.
// ListRecords will return all the records matching the supplied ListParameters, making multiple requests
// to Airtable if the number of matching records exceeds the 100 record limit for any one API request.
func (c *Client) ListRecords(tableName string, recordsHolder interface{}, listParams ...ListParameters) error {
	endpoint := fmt.Sprintf("%s/%s/%s?", apiBaseURL, c.baseID, tableName)
	if len(listParams) != 0 {
		listParameters := listParams[len(listParams)-1]
		endpoint = fmt.Sprintf("%s%s", endpoint, listParameters.URLEncode())
	}
	tempRecordsHolder := reflect.New(reflect.TypeOf(recordsHolder).Elem()).Interface()
	offsetHash := ""
	// We pass tempRecordsHolder here as a perf optimization so that we do not need to re-derive
	// the tempRecord for each request using reflection, but can instead reuse a single one. Since
	// tempRecordsHolder is always a slice, it's contents will be entirely replaced with each
	// subsequent unmarshalling.
	return c.recursivelyListRecordsAtOffset(endpoint, offsetHash, tempRecordsHolder, recordsHolder)
}

func (c *Client) recursivelyListRecordsAtOffset(endpoint string, offsetHash string, tempRecordsHolder, finalRecordsHolder interface{}) error {
	finalEndpoint := fmt.Sprintf("%s&offset=%s", endpoint, offsetHash)
	rawBody, err := c.request("GET", finalEndpoint, nil)
	if err != nil {
		return err
	}

	// Unmarshal into generic recordList struct. We need to use json.NewDecoder instead of json.Unmarshal
	// in order to call "UseNumber()" which causes all numbers to unmarshal to json.Number, the original
	// representation of the number. Without this, json.Unmarshal would convert all numbers to floating
	// point values when unmarshalling into an interface{} type since it doesn't specify the desired number
	// format.
	// Source: http://stackoverflow.com/questions/22343083/json-marshaling-with-long-numbers-in-golang-gives-floating-point-number
	d := json.NewDecoder(strings.NewReader(string(rawBody)))
	d.UseNumber()
	rl := recordList{}
	if err = d.Decode(&rl); err != nil {
		return err
	}

	// Marshall inner "Records" array of records back to JSON
	jsonRecords, err := json.Marshal(rl.Records)
	if err != nil {
		return err
	}

	// Unmarshall once more into the supplied tempRecordsHolder, an array of records
	if err = json.Unmarshal(jsonRecords, tempRecordsHolder); err != nil {
		return err
	}

	// Append the records returned from this request to the final list of records using reflection
	finalRecordsHolderVal := reflect.ValueOf(finalRecordsHolder).Elem()
	tempRecordsHolderVal := reflect.ValueOf(tempRecordsHolder).Elem()
	finalRecordsHolderVal.Set(reflect.AppendSlice(finalRecordsHolderVal, tempRecordsHolderVal))

	if rl.Offset != "" {
		return c.recursivelyListRecordsAtOffset(endpoint, rl.Offset, tempRecordsHolder, finalRecordsHolder)
	}
	return nil
}

// RetrieveRecord returns a single record from a given Airtable table.
func (c *Client) RetrieveRecord(tableName string, recordID string, recordHolder interface{}) error {
	if err := utils.CheckForValidRecordID(recordID); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/%s", apiBaseURL, c.baseID, tableName, recordID)
	rawBody, err := c.request("GET", endpoint, nil)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawBody, &recordHolder); err != nil {
		return err
	}
	return nil
}

// CreateRecord creates a new record in an Airtable table and updates the `record` struct with the created
// records field values i.e fields with default values would be populated as well as AirtableID with the
// record's id.
func (c *Client) CreateRecord(tableName string, record interface{}) error {
	endpoint := fmt.Sprintf("%s/%s/%s", apiBaseURL, c.baseID, tableName)
	rawBody, err := c.request("POST", endpoint, record)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawBody, &record); err != nil {
		return err
	}
	return nil
}

type updateBody struct {
	Fields map[string]interface{} `json:"fields"`
}

// UpdateRecord updates an existing record in an Airtable table and updates the new field values in
// the `record` struct passed in.
func (c *Client) UpdateRecord(tableName, recordID string, updatedFields map[string]interface{}, record interface{}) error {
	if err := utils.CheckForValidRecordID(recordID); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/%s", apiBaseURL, c.baseID, tableName, recordID)
	body := updateBody{}
	body.Fields = updatedFields
	rawBody, err := c.request("PATCH", endpoint, body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawBody, &record); err != nil {
		return err
	}
	return nil
}

// DestroyRecord deletes a record from an Airtable table by recordID
func (c *Client) DestroyRecord(tableName, recordID string) error {
	if err := utils.CheckForValidRecordID(recordID); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/%s/%s/%s", apiBaseURL, c.baseID, tableName, recordID)
	if _, err := c.request("DELETE", endpoint, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) request(method string, endpoint string, body interface{}) (rawBody []byte, err error) {
	var req *http.Request
	switch {
	case method == "GET" || method == "DELETE":
		req, err = http.NewRequest(method, endpoint, nil)
	case method == "POST" || method == "PATCH":
		req, err = c.requestWithBody(method, endpoint, body)
	default:
		return []byte{}, utils.SwitchCaseError("method", method)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	fullAPIVersion := fmt.Sprintf("%d.1.0", majorAPIVersion)
	req.Header.Add("x-api-version", fullAPIVersion)
	req.Header.Add("x-airtable-application-id", c.baseID)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()
	statusCode := resp.StatusCode
	rawBody, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	if statusCode == http.StatusTooManyRequests && c.ShouldRetryIfRateLimited {
		time.Sleep(retryDelayIfRateLimited)
		return c.request(method, endpoint, body)
	}
	if err := checkStatusCodeForError(statusCode, rawBody); err != nil {
		return []byte{}, err
	}
	return rawBody, nil
}

func (c *Client) requestWithBody(method string, endpoint string, body interface{}) (*http.Request, error) {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return &http.Request{}, err
	}
	bodyAsBuffer := bytes.NewBuffer(rawBody)
	req, err := http.NewRequest(method, endpoint, bodyAsBuffer)
	if err != nil {
		return &http.Request{}, err
	}
	req.Header.Add("content-type", "application/json")

	return req, nil
}

// ListParameters let's the caller describe the parameters he want's sent with a ListRecords request
// See the documentation at https://airtable.com/api for more information on how to use these parameters
type ListParameters struct {
	Fields          []string
	FilterByFormula string
	MaxRecords      int
	Sort            []SortParameter
	View            string
}

// URLEncode url encodes the ListParameters.
func (l *ListParameters) URLEncode() string {
	v := url.Values{}
	if len(l.Fields) != 0 {
		for _, field := range l.Fields {
			v.Add("fields[]", field)
		}
	}
	if l.FilterByFormula != "" {
		v.Add("filterByFormula", l.FilterByFormula)
	}
	if l.MaxRecords != 0 {
		v.Add("maxRecords", strconv.Itoa(l.MaxRecords))
	}
	if len(l.Sort) != 0 {
		for i, sort := range l.Sort {
			v.Add(fmt.Sprintf("sort[%d][field]", i), sort.Field)
			direction := "asc"
			if sort.ShouldSortDesc {
				direction = "desc"
			}
			v.Add(fmt.Sprintf("sort[%d][direction]", i), direction)
		}
	}
	if l.View != "" {
		v.Add("view", l.View)
	}
	return v.Encode()
}

// SortParameter is a sort object sent as part of the ListParameters that describes how the records
// should be sorted.
type SortParameter struct {
	Field          string
	ShouldSortDesc bool
}

// Error represents an error returned by the Airtable API.
type Error struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	StatusCode int
}

func (e Error) Error() string {
	return fmt.Sprintf("%s: %s [HTTP code %d]", e.Type, e.Message, e.StatusCode)
}

func checkStatusCodeForError(statusCode int, rawBody []byte) error {
	if statusCode == 200 {
		return nil
	}

	response := map[string]interface{}{}
	if err := json.Unmarshal(rawBody, &response); err != nil {
		return err
	}

	errorObj, ok := response["error"]
	if !ok {
		return Error{
			Type:       "MALFORMED_AIRTABLE_RESPONSE",
			Message:    "Airtable returned a non-200 response without an error json body",
			StatusCode: statusCode,
		}
	}
	// Marshall inner error back to JSON
	jsonError, err := json.Marshal(errorObj)
	if err != nil {
		return err
	}

	// Unmarshall once more into the error object
	errorResponse := Error{}
	if err = json.Unmarshal(jsonError, &errorResponse); err != nil {
		return err
	}

	switch statusCode {
	case 401:
		return Error{
			Type:       "AUTHENTICATION_REQUIRED",
			Message:    "You must provide a valid api key to perform this operation",
			StatusCode: statusCode,
		}
	case 403:
		return Error{
			Type:       "NOT_AUTHORIZED",
			Message:    "You are not authorized to perform this operation",
			StatusCode: statusCode,
		}
	case 404:
		ae := Error{
			Type:       "NOT_FOUND",
			StatusCode: statusCode,
		}
		if errorResponse.Message != "" {
			ae.Message = errorResponse.Message
		} else {
			ae.Message = "Could not find what you are looking for"
		}
		return ae
	case 413:
		return Error{
			Type:       "REQUEST_TOO_LARGE",
			Message:    "Request body is too large",
			StatusCode: statusCode,
		}
	case 422:
		ae := Error{
			StatusCode: statusCode,
		}
		if errorResponse.Message != "" {
			ae.Message = errorResponse.Message
		}
		if errorResponse.Type != "" {
			ae.Type = errorResponse.Type
		}
		return ae
	case 500:
		return Error{
			Type:       "SERVER_ERROR",
			Message:    "Try again. If the problem persists, contact support.",
			StatusCode: statusCode,
		}
	case 503:
		return Error{
			Type:       "SERVICE_UNAVAILABLE",
			Message:    "The service is temporarily unavailable. Please retry shortly.",
			StatusCode: statusCode,
		}
	}
	return nil
}
