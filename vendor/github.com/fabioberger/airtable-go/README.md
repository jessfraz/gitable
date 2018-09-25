<a href="https://godoc.org/github.com/fabioberger/airtable-go" ><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square" /></a>

Airtable Go Client Library
-------------------------------

### Installation

Make sure you have Golang v1.6 or higher installed. If not, <a href="https://golang.org/dl/">install it now</a>.

Fetch airtable-go:

```
go get github.com/fabioberger/airtable-go
```
Import airtable-go into your project:

```go
import "github.com/fabioberger/airtable-go"
```

### Usage

Create an instance of the airtable-go client:

```go
import (
	"os"
	"github.com/fabioberger/airtable-go"
)

airtableAPIKey := os.Getenv("AIRTABLE_API_KEY")
baseID := "apphllLCpWnySSF7q" // replace this with your airtable base's id

client, err := airtable.New(airtableAPIKey, baseID)
if err != nil {
	panic(err)
}
```
You can now call methods on the client instance. All client methods are documented in the project's <a href="https://godoc.org/github.com/fabioberger/airtable-go">GoDoc page</a>. You can also check out the <a href="https://github.com/fabioberger/airtable-go/blob/master/tests/stubbed_tests/client_test.go">stubbed</a> and <a href="https://github.com/fabioberger/airtable-go/blob/master/tests/integration_tests/client_test.go">integration</a> tests included in this project for working examples of all the client methods and options.

For Airtable specific documentation, see the interactive documentation at [https://airtable.com/api](https://airtable.com/api).

### Tests

Execute the following to run the stubbed out unit tests:

```
go test -v ./tests/stubbed_tests
```

In order to run the integration tests, you will need to copy <a href="https://airtable.com/shrnNgxIHdqd2Hu15">this test Airtable base</a> into your own Airtable account using the "copy base" button in the top right.

Next, set two environment variables corresponding to your Airtable API key and the baseID of your copy of the test Airtable base.

```
export AIRTABLE_TEST_API_KEY=YOUR_API_KEY
export AIRTABLE_TEST_BASE_ID=YOUR_TEST_BASE_ID
```

Now you can run the integration tests with:

```
go test -v ./tests/integration_tests
```

### Versioning

This library uses [Semantic Versioning](http://semver.org/).
