package testConfigs

import "os"

var (
	// AirtableTestAPIKey is the test Airtable api key.
	AirtableTestAPIKey = os.Getenv("AIRTABLE_TEST_API_KEY")

	// AirtableTestBaseID is the baseId used to run the integration tests.
	AirtableTestBaseID = os.Getenv("AIRTABLE_TEST_BASE_ID")
)
