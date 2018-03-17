// Package testBase contains the information about the Airtable test base used by both stubbed and
// integration tests. Information includes table names and structs representing records in the different
// tables
package testBase

import airtable "github.com/fabioberger/airtable-go"

var (
	// TasksTableName is the name of the Airtable table containing Task records
	TasksTableName = "Tasks"
	// TeamMatesTableName is the name of the Airtable table containing TeamMate records
	TeamMatesTableName = "Teammates"
	// LogTableName is the name of the Airtable table containing Log records
	LogTableName = "Log"
)

// Task represents a single record in the `Task` Airtable table
type Task struct {
	AirtableID string `json:"id,omitempty"`
	Fields     struct {
		Name      string  `json:"name"`
		Notes     string  `json:"notes"`
		Completed bool    `json:"Completed"`
		TimeEst   float64 `json:"Time Estimate (days)"`
	} `json:"fields"`
}

// TeamMate represents a single record in the `TeamMate` Airtable table
type TeamMate struct {
	AirtableID string `json:"id,omitempty"`
	Fields     struct {
		Name  string
		Photo []airtable.Attachment
	} `json:"fields"`
}

// Log represents a single record in the `Log` Airtable table
type Log struct {
	AirtableID string `json:"id,omitempty"`
	Fields     struct {
		AutoNumber int      `json:"Auto Number"`
		Projects   []string `json:"Projects"`
	}
}
