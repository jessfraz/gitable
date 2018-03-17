/*
Copyright 2016 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	airtable "github.com/fabioberger/airtable-go"
	"github.com/google/go-github/github"
	"github.com/jessfraz/gitable/version"
	"github.com/sirupsen/logrus"
)

const (
	// BANNER is what is printed for help/info output.
	BANNER = `       _ _        _     _
  __ _(_) |_ __ _| |__ | | ___
 / _` + "`" + ` | | __/ _` + "`" + ` | '_ \| |/ _ \
| (_| | | || (_| | |_) | |  __/
 \__, |_|\__\__,_|_.__/|_|\___|
 |___/

 Bot to update an airtable sheet with GitHub pull request or issue data.
 Version: %s
 Build: %s

`
)

var (
	interval string
	once     bool

	githubToken string

	airtableAPIKey    string
	airtableBaseID    string
	airtableTableName string

	debug bool
	vrsn  bool
)

func init() {
	// parse flags
	flag.StringVar(&interval, "interval", "1m", "update interval (ex. 5ms, 10s, 1m, 3h)")
	flag.BoolVar(&once, "once", false, "run once and exit, do not run as a daemon")

	flag.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")

	flag.StringVar(&airtableAPIKey, "airtable-apikey", os.Getenv("AIRTABLE_APIKEY"), "Airtable API Key (or env var AIRTABLE_APIKEY)")
	flag.StringVar(&airtableBaseID, "airtable-baseid", os.Getenv("AIRTABLE_BASEID"), "Airtable Base ID (or env var AIRTABLE_BASEID)")
	flag.StringVar(&airtableTableName, "airtable-table", os.Getenv("AIRTABLE_TABLE"), "Airtable Table (or env var AIRTABLE_TABLE)")

	flag.BoolVar(&vrsn, "version", false, "print version and exit")
	flag.BoolVar(&vrsn, "v", false, "print version and exit (shorthand)")
	flag.BoolVar(&debug, "d", false, "run in debug mode")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(BANNER, version.VERSION, version.GITCOMMIT))
		flag.PrintDefaults()
	}

	flag.Parse()

	if vrsn {
		fmt.Printf("gitable version %s, build %s", version.VERSION, version.GITCOMMIT)
		os.Exit(0)
	}

	// set log level
	if debug {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if githubToken == "" {
		usageAndExit("GitHub token cannot be empty.", 1)
	}

	if airtableAPIKey == "" {
		usageAndExit("Airtable API Key cannot be empty.", 1)
	}

	if airtableBaseID == "" {
		usageAndExit("Airtable Base ID cannot be empty.", 1)
	}

	if airtableTableName == "" {
		usageAndExit("Airtable Table cannot be empty.", 1)
	}
}

func main() {
	var ticker *time.Ticker

	// On ^C, or SIGTERM handle exit.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		for sig := range c {
			ticker.Stop()
			logrus.Infof("Received %s, exiting.", sig.String())
			os.Exit(0)
		}
	}()

	ctx := context.Background()

	// Parse the duration.
	dur, err := time.ParseDuration(interval)
	if err != nil {
		logrus.Fatalf("parsing %s as duration failed: %v", interval, err)
	}
	ticker = time.NewTicker(dur)

	// Create the http client.
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	// Create the github client.
	ghClient := github.NewClient(tc)

	// Create the airtable client.
	airtableClient, err := airtable.New(airtableAPIKey, airtableBaseID)
	if err != nil {
		logrus.Fatal(err)
	}

	// If the user passed the once flag, just do the run once and exit.
	if once {
		if err := run(ctx, ghClient, airtableClient); err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Updated airtable table %s for base %s", airtableTableName, airtableBaseID)
		os.Exit(0)
	}

	logrus.Infof("Starting bot to update airtable table %s for base %s every %s", airtableTableName, airtableBaseID, interval)
	for range ticker.C {
		if err := run(ctx, ghClient, airtableClient); err != nil {
			logrus.Fatal(err)
		}
	}
}

// githubRecord holds the data for the airtable fields that define the github data.
type githubRecord struct {
	ID     string
	Fields struct {
		Reference string
		Title     string
		State     string
		Author    string
		Type      string
		Labels    []string
		Comments  int
		URL       string
		Updated   time.Time
		Created   time.Time
		Project   interface{}
	}
}

func run(ctx context.Context, ghClient *github.Client, airtableClient *airtable.Client) error {
	ghRecords := []githubRecord{}
	if err := airtableClient.ListRecords(airtableTableName, &ghRecords); err != nil {
		return fmt.Errorf("listing records for table %s failed: %v", airtableTableName, err)
	}

	// Iterate over the records.
	for _, record := range ghRecords {
		// Split the reference into repository and issue number.
		parts := strings.SplitN(record.Fields.Reference, "#", 2)
		if len(parts) < 2 {
			return fmt.Errorf("could not parse reference name into repository and issue number for %s, got: %#v", record.Fields.Reference, parts)
		}
		repolong := parts[0]
		i := parts[1]

		// Parse the string id into an int.
		id, err := strconv.Atoi(i)
		if err != nil {
			return err
		}

		// Split the repo name into owner and repo.
		parts = strings.SplitN(repolong, "/", 2)
		if len(parts) < 2 {
			return fmt.Errorf("could not parse reference name into owner and repo for %s, got: %#v", repolong, parts)
		}

		// Get the github issue.
		logrus.Debugf("getting issue %s/%s#%d", parts[0], parts[1], id)
		issue, _, err := ghClient.Issues.Get(ctx, parts[0], parts[1], id)
		if err != nil {
			return fmt.Errorf("getting issue %s/%s#%d failed: %v", parts[0], parts[1], id, err)
		}

		// Iterate over the labels.
		labels := []string{}
		for _, label := range issue.Labels {
			labels = append(labels, label.GetName())
		}

		issueType := "issue"
		if issue.IsPullRequest() {
			issueType = "pull request"
		}

		// Update the record fields.
		updatedFields := map[string]interface{}{
			"Title":    issue.GetTitle(),
			"State":    issue.GetState(),
			"Author":   issue.GetUser().GetLogin(),
			"Type":     issueType,
			"Comments": issue.GetComments(),
			"URL":      issue.GetHTMLURL(),
			"Updated":  issue.GetUpdatedAt(),
			"Created":  issue.GetCreatedAt(),
		}
		// Do without labels.
		logrus.Debugf("updating record %s for issue %s/%s#%d", record.ID, parts[0], parts[1], id)
		if err := airtableClient.UpdateRecord(airtableTableName, record.ID, updatedFields, &record); err != nil {
			return fmt.Errorf("updating record %s for issue %s/%s#%d failed: %v", record.ID, parts[0], parts[1], id, err)
		}
		// Try again with labels, since the user may not have pre-populated the label options.
		// TODO: add a create multiple select when the airtable API supports it.
		updatedFields["Labels"] = labels
		if err := airtableClient.UpdateRecord(airtableTableName, record.ID, updatedFields, &record); err != nil {
			logrus.Warnf("updating record with labels %s for issue %s/%s#%d failed: %v", record.ID, parts[0], parts[1], id, err)
		}
	}

	return nil
}

func usageAndExit(message string, exitCode int) {
	if message != "" {
		fmt.Fprintf(os.Stderr, message)
		fmt.Fprintf(os.Stderr, "\n\n")
	}
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(exitCode)
}
