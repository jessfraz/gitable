package integrationTests

import (
	"testing"

	airtable "github.com/fabioberger/airtable-go"
	"github.com/fabioberger/airtable-go/tests/test_base"
	"github.com/fabioberger/airtable-go/tests/test_configs"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

var client *airtable.Client

func (s *ClientSuite) SetUpSuite(c *C) {
	var err error
	client, err = airtable.New(testConfigs.AirtableTestAPIKey, testConfigs.AirtableTestBaseID)
	if err != nil {
		c.Error(err)
	}
}

func (s *ClientSuite) TestListTeammateRecords(c *C) {
	teamMates := []testBase.TeamMate{}
	err := client.ListRecords(testBase.TeamMatesTableName, &teamMates)
	c.Assert(err, Equals, nil)
	c.Assert(len(teamMates), Equals, 9)
}

func (s *ClientSuite) TestRetrieveRecord(c *C) {
	tasks := []testBase.Task{}
	client.ListRecords(testBase.TasksTableName, &tasks)
	t := tasks[0]

	aTask := testBase.Task{}
	client.RetrieveRecord(testBase.TasksTableName, t.AirtableID, &aTask)
	c.Assert(aTask.Fields.Name, Equals, t.Fields.Name)
}

func (s *ClientSuite) TestCreateAndDestroyRecord(c *C) {
	tm := testBase.TeamMate{}
	tm.Fields.Name = "Bob"
	err := client.CreateRecord(testBase.TeamMatesTableName, &tm)
	c.Assert(err, Equals, nil)

	err = client.DestroyRecord(testBase.TeamMatesTableName, tm.AirtableID)
	c.Assert(err, Equals, nil)
}

func (s *ClientSuite) TestUpdateRecord(c *C) {
	tasks := []testBase.Task{}
	listParams := airtable.ListParameters{
		FilterByFormula: "{Name} = \"Design Tea Packaging\"",
	}
	client.ListRecords(testBase.TasksTableName, &tasks, listParams)
	t := tasks[0]
	oldName := t.Fields.Name

	newName := "Redesign Tea Packaging"
	updatedFields := map[string]interface{}{
		"Name": newName,
	}
	err := client.UpdateRecord(testBase.TeamMatesTableName, t.AirtableID, updatedFields, &t)
	c.Assert(err, Equals, nil)
	c.Assert(t.Fields.Name, Equals, newName)
	c.Assert(t.Fields.Name, Not(Equals), oldName)

	// revert to old name
	updatedFields = map[string]interface{}{
		"Name": oldName,
	}
	err = client.UpdateRecord(testBase.TeamMatesTableName, t.AirtableID, updatedFields, &t)
	c.Assert(err, Equals, nil)
}

func (s *ClientSuite) TestListRecordsRequiringMultipleRequests(c *C) {
	logs := []testBase.Log{}
	if err := client.ListRecords(testBase.LogTableName, &logs); err != nil {
		c.Error(err)
	}
	c.Assert(len(logs), Equals, 117)
}

func (s *ClientSuite) TestListRecordsInSpecificView(c *C) {
	tasks := []testBase.Task{}
	listParameters := airtable.ListParameters{
		View: "Tea Packaging Tasks",
	}
	err := client.ListRecords(testBase.TasksTableName, &tasks, listParameters)
	c.Assert(err, Equals, nil)
	c.Assert(len(tasks), Equals, 1)
}

func (s *ClientSuite) TestListRecordsUsingFilterByFormula(c *C) {
	tasks := []testBase.Task{}
	listParameters := airtable.ListParameters{
		FilterByFormula: "{Time Estimate (days)} > 2",
	}
	err := client.ListRecords(testBase.TasksTableName, &tasks, listParameters)
	c.Assert(err, Equals, nil)
	c.Assert(len(tasks), Equals, 2)
}

func (s *ClientSuite) TestListRecordsWithASortedOrder(c *C) {
	tasks := []testBase.Task{}
	listParameters := airtable.ListParameters{
		Sort: []airtable.SortParameter{
			airtable.SortParameter{
				Field:          "Time Estimate (days)",
				ShouldSortDesc: false,
			},
		},
	}
	err := client.ListRecords(testBase.TasksTableName, &tasks, listParameters)
	c.Assert(err, Equals, nil)
	c.Assert(tasks[0].Fields.TimeEst, Equals, 1.5)
	c.Assert(tasks[1].Fields.TimeEst, Equals, 2.5)
	c.Assert(tasks[2].Fields.TimeEst, Equals, 15.0)
}

func (s *ClientSuite) TestListRecordsWithSpecifiedMaxRecords(c *C) {
	tasks := []testBase.Task{}
	listParameters := airtable.ListParameters{
		MaxRecords: 1,
	}
	err := client.ListRecords(testBase.TasksTableName, &tasks, listParameters)
	c.Assert(err, Equals, nil)
	c.Assert(len(tasks), Equals, 1)
}

func (s *ClientSuite) TestListRecordsWithSpecifiedFields(c *C) {
	tasks := []testBase.Task{}
	listParameters := airtable.ListParameters{
		Fields: []string{"Time Estimate (days)"},
	}
	err := client.ListRecords(testBase.TasksTableName, &tasks, listParameters)
	c.Assert(err, Equals, nil)
	for _, task := range tasks {
		c.Assert(task.Fields.Name, Equals, "")
		c.Assert(task.Fields.Notes, Equals, "")
		c.Assert(task.Fields.Completed, Equals, false)
	}
}
