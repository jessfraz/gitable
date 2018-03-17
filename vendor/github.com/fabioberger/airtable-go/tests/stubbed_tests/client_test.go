package stubbedTests

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	airtable "github.com/fabioberger/airtable-go"
	"github.com/fabioberger/airtable-go/tests/test_base"
	"github.com/fabioberger/airtable-go/tests/test_configs"
	. "gopkg.in/check.v1"
)

func newFakeHTTPClient(statusCode int, filePath string) *http.Client {
	c := &http.Client{}
	c.Transport = fileRoundTripper{
		statusCode: statusCode,
		filePath:   filePath,
	}
	return c
}

type fileRoundTripper struct {
	statusCode int
	filePath   string
}

func (f fileRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	rawResponse, err := ioutil.ReadFile(f.filePath)
	if err != nil {
		return nil, err
	}
	httpResponse := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader(rawResponse)),
		StatusCode: f.statusCode,
	}
	return httpResponse, nil
}

var client *airtable.Client

func (s *ClientSuite) SetUpSuite(c *C) {
	var err error
	client, err = airtable.New(testConfigs.AirtableTestAPIKey, testConfigs.AirtableTestBaseID)
	if err != nil {
		c.Error(err)
	}
}

var fakeRecordID = "recSG8Ytl8KWpFAKE"

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) TearDownTest(c *C) {
	client.HTTPClient = http.DefaultClient
}

func (s *ClientSuite) TestListRecords(c *C) {
	tasks := []testBase.Task{}
	client.HTTPClient = newFakeHTTPClient(200, "../test_responses/list_tasks.json")
	err := client.ListRecords(testBase.TasksTableName, &tasks)
	c.Assert(err, Equals, nil)
	c.Assert(len(tasks), Equals, 3)
}

func (s *ClientSuite) TestAirtableError(c *C) {
	tasks := []testBase.Task{}
	client.HTTPClient = newFakeHTTPClient(404, "../test_responses/404_error.json")
	err := client.ListRecords(testBase.TasksTableName, &tasks)
	c.Assert(err.Error(), Equals, "NOT_FOUND: Could not find table x in application appmUJMUx1SyZYQYX [HTTP code 404]")
}

func (s *ClientSuite) TestRetrieveRecord(c *C) {
	aTask := testBase.Task{}
	client.HTTPClient = newFakeHTTPClient(200, "../test_responses/retrieve_task.json")
	client.RetrieveRecord(testBase.TasksTableName, fakeRecordID, &aTask)
	c.Assert("Research other tea packaging", Equals, aTask.Fields.Name)
}

func (s *ClientSuite) TestCreateRecord(c *C) {
	tm := testBase.TeamMate{}
	tm.Fields.Name = "Bob"
	client.HTTPClient = newFakeHTTPClient(200, "../test_responses/create_teammate.json")
	err := client.CreateRecord(testBase.TeamMatesTableName, &tm)
	c.Assert(err, Equals, nil)
}

func (s *ClientSuite) TestDestroyRecord(c *C) {
	client.HTTPClient = newFakeHTTPClient(200, "../test_responses/delete_teammate.json")
	err := client.DestroyRecord(testBase.TeamMatesTableName, fakeRecordID)
	c.Assert(err, Equals, nil)
}

func (s *ClientSuite) TestUpdateRecord(c *C) {
	updatedFields := map[string]interface{}{
		"Name": "John Coltrain",
	}
	client.HTTPClient = newFakeHTTPClient(200, "../test_responses/update_teammate.json")
	t := testBase.TeamMate{}
	err := client.UpdateRecord(testBase.TeamMatesTableName, fakeRecordID, updatedFields, &t)
	c.Assert(err, Equals, nil)
	c.Assert(t.Fields.Name, Equals, "John Coltrain")
}

func (s *ClientSuite) TestRetryLogicIfRateLimited(c *C) {
	channel := make(chan bool)
	go func() {
		updatedFields := map[string]interface{}{
			"Name": "Bill Bob",
		}
		client.HTTPClient = newFakeHTTPClient(http.StatusTooManyRequests, "../test_responses/update_teammate.json")
		err := client.UpdateRecord(testBase.TeamMatesTableName, fakeRecordID, updatedFields, nil)
		c.Assert(err, Equals, nil)
		channel <- true
	}()

	for {
		select {
		case _ = <-channel:
			c.Error("Request terminated before rateLimit sleep completed.")
		case _ = <-time.After(2 * time.Second):
			return // Request correctly still waiting to retry after rateLimited
		}
	}
}
