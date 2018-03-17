package airtable_test

import (
	"fmt"
	"os"

	airtable "github.com/fabioberger/airtable-go"
)

func ExampleNew() {
	airtableAPIKey := os.Getenv("AIRTABLE_API_KEY")
	baseID := "apphllLCpWnySSF7q"

	client, err := airtable.New(airtableAPIKey, baseID)
	if err != nil {
		panic(err)
	}

	fmt.Println(client)
}

func ExampleClient_CreateRecord() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	type task struct {
		AirtableID string
		Fields     struct {
			Name  string
			Notes string
		}
	}

	aTask := task{}
	aTask.Fields.Name = "Contact suppliers"
	aTask.Fields.Notes = "Get pricing on both the blue and green variants"

	client.CreateRecord("TABLE_NAME", &aTask)

	// aTask.AirtableID is now set to the newly created Airtable recordID
}

func ExampleClient_DestroyRecord() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	if err := client.DestroyRecord("TABLE_NAME", "RECORD_ID"); err != nil {
		panic(err)
	}
}

func ExampleClient_ListRecords() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	type task struct {
		AirtableID string
		Fields     struct {
			Name  string
			Notes string
		}
	}

	tasks := []task{}
	if err := client.ListRecords("TABLE_NAME", &tasks); err != nil {
		panic(err)
	}

	fmt.Println(tasks)
}

func ExampleClient_RetrieveRecord() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	type task struct {
		AirtableID string
		Fields     struct {
			Name  string
			Notes string
		}
	}

	retrievedTask := task{}
	if err := client.RetrieveRecord("TABLE_NAME", "RECORD_ID", &retrievedTask); err != nil {
		panic(err)
	}

	fmt.Println(retrievedTask)
}

func ExampleClient_UpdateRecord() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	type task struct {
		AirtableID string
		Fields     struct {
			Name  string
			Notes string
		}
	}

	aTask := task{}
	aTask.Fields.Name = "Clean kitchen"
	aTask.Fields.Notes = "Make sure to clean all the counter tops"

	UpdatedFields := map[string]interface{}{
		"Name": "Clean entire kitchen",
	}
	if err := client.UpdateRecord("TABLE_NAME", "RECORD_ID", UpdatedFields, &aTask); err != nil {
		panic(err)
	}

	fmt.Println(aTask)
}

func ExampleListParameters() {
	client, _ := airtable.New("AIRTABLE_API_KEY", "BASE_ID")

	type task struct {
		AirtableID string
		Fields     struct {
			Name     string
			Notes    string
			Priority int
		}
	}

	listParams := airtable.ListParameters{
		Fields:          []string{"Name", "Notes", "Priority"},
		FilterByFormula: "{Priority} < 2",
		MaxRecords:      50,
		Sort: []airtable.SortParameter{
			airtable.SortParameter{
				Field:          "Priority",
				ShouldSortDesc: true,
			},
		},
		View: "Main View",
	}
	tasks := []task{}
	if err := client.ListRecords("TABLE_NAME", &tasks, listParams); err != nil {
		panic(err)
	}

	fmt.Println(tasks)
}
