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

 Bot to automatically sync and update an airtable sheet with
 GitHub pull request and issue data.
 Version: %s
 Build: %s

`
)

var (
	interval string
	autofill bool
	once     bool

	githubToken string
	orgs        stringSlice

	airtableAPIKey    string
	airtableBaseID    string
	airtableTableName string

	debug bool
	vrsn  bool
)

// stringSlice is a slice of strings
type stringSlice []string

// implement the flag interface for stringSlice
func (s *stringSlice) String() string {
	return fmt.Sprintf("%s", *s)
}
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func init() {
	// parse flags
	flag.StringVar(&interval, "interval", "1m", "update interval (ex. 5ms, 10s, 1m, 3h)")
	flag.BoolVar(&autofill, "autofill", false, "autofill all pull requests and issues for a user [or orgs] to a table (defaults to current user unless --orgs is set)")
	flag.BoolVar(&once, "once", false, "run once and exit, do not run as a daemon")

	flag.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	flag.Var(&orgs, "orgs", "organizations to include (this option only applies to --autofill)")

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

	// Affiliation must be set before we add the user to the "orgs".
	affiliation := "owner"
	if len(orgs) > 0 {
		affiliation += ",organization_member"
	}

	// Get the current user for the GitHub token.
	user, _, err := ghClient.Users.Get(ctx, "")
	if err != nil {
		logrus.Fatalf("getting current github user for token failed: %v", err)
	}
	// Add the current user to orgs.
	orgs = append(orgs, user.GetLogin())

	// Create our bot type.
	bot := &bot{
		ghClient:       ghClient,
		airtableClient: airtableClient,
		// Initialize our map.
		issues: map[string]*github.Issue{},
	}

	// If the user passed the once flag, just do the run once and exit.
	if once {
		if err := bot.run(ctx, affiliation); err != nil {
			logrus.Fatal(err)
		}
		logrus.Infof("Updated airtable table %s for base %s", airtableTableName, airtableBaseID)
		os.Exit(0)
	}

	logrus.Infof("Starting bot to update airtable table %s for base %s every %s", airtableTableName, airtableBaseID, interval)
	for range ticker.C {
		if err := bot.run(ctx, affiliation); err != nil {
			logrus.Fatal(err)
		}
	}
}

type bot struct {
	ghClient       *github.Client
	airtableClient *airtable.Client
	issues         map[string]*github.Issue
}

// githubRecord holds the data for the airtable fields that define the github data.
type githubRecord struct {
	ID     string `json:"id,omitempty"`
	Fields Fields `json:"fields,omitempty"`
}

// Fields defines the airtable fields for the data.
type Fields struct {
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
	Completed time.Time
	Project   interface{}
}

func (bot *bot) run(ctx context.Context, affiliation string) error {
	// if we are in autofill mode, get our repositories
	if autofill {
		page := 1
		perPage := 100
		logrus.Infof("getting repositories to be autofilled for org[s]: %s...", strings.Join(orgs, ", "))
		if err := bot.getRepositories(ctx, page, perPage, affiliation); err != nil {
			return err
		}
	}

	ghRecords := []githubRecord{}
	if err := bot.airtableClient.ListRecords(airtableTableName, &ghRecords); err != nil {
		return fmt.Errorf("listing records for table %s failed: %v", airtableTableName, err)
	}

	// Iterate over the records.
	for _, record := range ghRecords {
		// Parse the reference.
		user, repo, id, err := parseReference(record.Fields.Reference)
		if err != nil {
			return err
		}

		// Get the github issue.
		var issue *github.Issue

		// Check if we already have it from autofill.
		if autofill {
			if i, ok := bot.issues[record.Fields.Reference]; ok {
				logrus.Debugf("found github issue %s from autofill", record.Fields.Reference)
				issue = i
				// delete the key from the autofilled map
				delete(bot.issues, record.Fields.Reference)
			}
		}

		// If we don't already have the issue, then get it.
		if issue == nil {
			logrus.Debugf("getting issue %s", record.Fields.Reference)
			issue, _, err = bot.ghClient.Issues.Get(ctx, user, repo, id)
			if err != nil {
				return fmt.Errorf("getting issue %s failed: %v", record.Fields.Reference, err)
			}
		}

		if err := bot.applyRecordToTable(ctx, issue, record.Fields.Reference, record.ID); err != nil {
			return err
		}
	}

	// If we autofilled issues, loop over and create which ever ones remain.
	for key, issue := range bot.issues {
		if err := bot.applyRecordToTable(ctx, issue, key, ""); err != nil {
			return err
		}
	}

	return nil
}

func (bot *bot) applyRecordToTable(ctx context.Context, issue *github.Issue, key, id string) error {
	// Parse the reference.
	user, repo, number, err := parseReference(key)
	if err != nil {
		return err
	}

	// Iterate over the labels.
	labels := []string{}
	for _, label := range issue.Labels {
		labels = append(labels, label.GetName())
	}

	issueType := "issue"
	if issue.IsPullRequest() {
		issueType = "pull request"
		// If the status is closed, we should find out if the
		// _actual_ pull request status is "merged".
		merged, _, err := bot.ghClient.PullRequests.IsMerged(ctx, user, repo, number)
		if err != nil {
			return err
		}
		if merged {
			mstr := "merged"
			issue.State = &mstr
		}
	}

	// Create our empty record struct.
	record := githubRecord{
		Fields: Fields{
			Reference: key,
			Title:     issue.GetTitle(),
			State:     issue.GetState(),
			Author:    issue.GetUser().GetLogin(),
			Type:      issueType,
			Comments:  issue.GetComments(),
			URL:       issue.GetHTMLURL(),
			Updated:   issue.GetUpdatedAt(),
			Created:   issue.GetCreatedAt(),
			Completed: issue.GetClosedAt(),
		},
	}

	// Update the record fields.
	fields := map[string]interface{}{
		"Reference": record.Fields.Reference,
		"Title":     record.Fields.Title,
		"State":     record.Fields.State,
		"Author":    record.Fields.Author,
		"Type":      record.Fields.Type,
		"Comments":  record.Fields.Comments,
		"URL":       record.Fields.URL,
		"Updated":   record.Fields.Updated,
		"Created":   record.Fields.Created,
		"Completed": record.Fields.Completed,
	}

	if id != "" {
		// If we were passed a record ID, update the record instead of create.
		logrus.Debugf("updating record %s for issue %s", id, key)
		if err := bot.airtableClient.UpdateRecord(airtableTableName, id, fields, &record); err != nil {
			return fmt.Errorf("updating record %s for issue %s failed: %v", id, key, err)
		}
	} else {
		// Create the field.
		logrus.Debugf("creating new record for issue %s", key)
		if err := bot.airtableClient.CreateRecord(airtableTableName, &record); err != nil {
			return err
		}
	}

	// Try again with labels, since the user may not have pre-populated the label options.
	// TODO: add a create multiple select when the airtable API supports it.
	fields["Labels"] = labels
	if err := bot.airtableClient.UpdateRecord(airtableTableName, record.ID, fields, &record); err != nil {
		logrus.Warnf("updating record with labels %s for issue %s failed: %v", record.ID, key, err)
	}

	return nil
}

func (bot *bot) getRepositories(ctx context.Context, page, perPage int, affiliation string) error {
	opt := &github.RepositoryListOptions{
		Affiliation: affiliation,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}
	repos, resp, err := bot.ghClient.Repositories.List(ctx, "", opt)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		// logrus.Debugf("getting issues for repo %s...", repo.GetFullName())
		ipage := 0
		if err := bot.getIssues(ctx, ipage, perPage, repo.GetOwner().GetLogin(), repo.GetName()); err != nil {
			return err
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return bot.getRepositories(ctx, page, perPage, affiliation)
}

func (bot *bot) getIssues(ctx context.Context, page, perPage int, owner, repo string) error {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	issues, resp, err := bot.ghClient.Issues.ListByRepo(ctx, owner, repo, opt)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		key := fmt.Sprintf("%s/%s#%d", owner, repo, issue.GetNumber())

		// logrus.Debugf("handling issue %s...", key)
		bot.issues[key] = issue
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return bot.getIssues(ctx, page, perPage, owner, repo)
}

func parseReference(ref string) (string, string, int, error) {
	// Split the reference into repository and issue number.
	parts := strings.SplitN(ref, "#", 2)
	if len(parts) < 2 {
		return "", "", 0, fmt.Errorf("could not parse reference name into repository and issue number for %s, got: %#v", ref, parts)
	}
	repolong := parts[0]
	i := parts[1]

	// Parse the string id into an int.
	id, err := strconv.Atoi(i)
	if err != nil {
		return "", "", 0, err
	}

	// Split the repo name into owner and repo.
	parts = strings.SplitN(repolong, "/", 2)
	if len(parts) < 2 {
		return "", "", 0, fmt.Errorf("could not parse reference name into owner and repo for %s, got: %#v", repolong, parts)
	}

	return parts[0], parts[1], id, nil
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
