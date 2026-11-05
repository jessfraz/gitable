package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/oauth2"

	airtable "github.com/iwoj/airtable-go"
	"github.com/genuinetools/pkg/cli"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/jessfraz/gitable/version"
	"github.com/sirupsen/logrus"
)

var (
	interval    time.Duration
	autofill    bool
	verboseKeys bool
	once        bool

	githubToken string
	enturl      string
	orgs        stringSlice
	watched     bool
	watchSince  string

	airtableAPIKey    string
	airtableBaseID    string
	airtableTableName string

	debug bool
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

func main() {
	// Create a new cli program.
	p := cli.NewProgram()
	p.Name = "gitable"
	p.Description = "Bot to automatically sync and update an airtable sheet with GitHub pull request and issue data"

	// Set the GitCommit and Version.
	p.GitCommit = version.GITCOMMIT
	p.Version = version.VERSION

	// Setup the global flags.
	p.FlagSet = flag.NewFlagSet("global", flag.ExitOnError)
	p.FlagSet.DurationVar(&interval, "interval", time.Minute, "update interval (ex. 5ms, 10s, 1m, 3h)")
	p.FlagSet.BoolVar(&autofill, "autofill", false, "autofill all pull requests and issues for a user [or orgs] to a table (defaults to current user unless --orgs is set)")
	p.FlagSet.BoolVar(&verboseKeys, "verbose-keys", false, "include title data in keys")
	p.FlagSet.BoolVar(&once, "once", false, "run once and exit, do not run as a daemon")

	p.FlagSet.StringVar(&githubToken, "github-token", os.Getenv("GITHUB_TOKEN"), "GitHub API token (or env var GITHUB_TOKEN)")
	p.FlagSet.Var(&orgs, "orgs", "organizations to include (this option only applies to --autofill)")
	p.FlagSet.StringVar(&enturl, "github-url", "", "Connect to a specific GitHub server, provide full API URL (ex. https://github.example.com/api/v3/)")

	p.FlagSet.StringVar(&airtableAPIKey, "airtable-apikey", os.Getenv("AIRTABLE_APIKEY"), "Airtable API Key (or env var AIRTABLE_APIKEY)")
	p.FlagSet.StringVar(&airtableBaseID, "airtable-baseid", os.Getenv("AIRTABLE_BASEID"), "Airtable Base ID (or env var AIRTABLE_BASEID)")
	p.FlagSet.StringVar(&airtableTableName, "airtable-table", os.Getenv("AIRTABLE_TABLE"), "Airtable Table (or env var AIRTABLE_TABLE)")

	p.FlagSet.BoolVar(&watched, "watched", false, "include the watched repositories")
	p.FlagSet.StringVar(&watchSince, "watch-since", "2008-01-01T00:00:00Z", "defines the starting point of the issues been watched (format: 2006-01-02T15:04:05Z). defaults to no filter")

	p.FlagSet.BoolVar(&debug, "debug", false, "enable debug logging")
	p.FlagSet.BoolVar(&debug, "d", false, "enable debug logging")

	// Set the before function.
	p.Before = func(ctx context.Context) error {
		// Set the log level.
		if debug {
			logrus.SetLevel(logrus.DebugLevel)
		}

		if len(githubToken) < 1 {
			return errors.New("gitHub token cannot be empty")
		}

		if len(airtableAPIKey) < 1 {
			return errors.New("airtable API Key cannot be empty")
		}

		if len(airtableBaseID) < 1 {
			return errors.New("airtable Base ID cannot be empty")
		}

		if len(airtableTableName) < 1 {
			return errors.New("airtable Table cannot be empty")
		}

		return nil
	}

	// Set the main program action.
	p.Action = func(ctx context.Context, args []string) error {
		ticker := time.NewTicker(interval)

		// On ^C, or SIGTERM handle exit.
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		signal.Notify(signals, syscall.SIGTERM)
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		go func() {
			for sig := range signals {
				logrus.Infof("Received %s, exiting.", sig.String())
				ticker.Stop()
				cancel()
				os.Exit(0)
			}
		}()

		// Create the http client.
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)

		// Create the HTTP cache.
		cachePath := "/tmp/cache"
		if err := os.MkdirAll(cachePath, 0777); err != nil {
			logrus.Fatal(err)
		}
		cache := diskcache.New(cachePath)
		tr := httpcache.NewTransport(cache)
		c := &http.Client{Transport: tr}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, c)

		// Create the github client.
		tc := oauth2.NewClient(ctx, ts)
		client := github.NewClient(tc)
		if enturl != "" {
			var err error
			client.BaseURL, err = url.Parse(enturl + "/api/v3/")
			if err != nil {
				logrus.Fatal(err)
			}
		}

		// Create the airtable client.
		airtableClient, err := airtable.New(airtableAPIKey, airtableBaseID)
		if err != nil {
			logrus.Fatal(err)
		}

		// Affiliation must be set before we add the user to the "orgs".
		affiliation := "owner,collaborator"
		if len(orgs) > 0 {
			affiliation += ",organization_member"
		}

		// If we didn't get any orgs explicitly passed, use the current user.
		if len(orgs) == 0 {
			// Get the current user for the GitHub token.
			user, _, err := client.Users.Get(ctx, "")
			if err != nil {
				logrus.Fatalf("getting current github user for token failed: %v", err)
			}
			// Add the current user to orgs.
			orgs = append(orgs, user.GetLogin())
		}

		// Create our bot type.
		bot := &bot{
			ghClient:       client,
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
		return nil
	}

	// Run our program.
	p.Run()
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
	Typecast bool `json:"typecast,omitempty"`
}

// Fields defines the airtable fields for the data.
type Fields struct {
	Reference  string
	Title      string
	Body       string
	State      string
	Author     string
	Type       string
	Labels     []string
	Comments   int
	URL        string
	Updated    time.Time
	Created    time.Time
	Completed  time.Time
	Project    interface{}
	Repository string
}

func (bot *bot) run(ctx context.Context, affiliation string) error {
	// if we are in autofill mode, get our repositories
	if autofill {
		page := 1
		perPage := 100
		logrus.Infof("getting repositories to be autofilled for org[s]: %s...", strings.Join(orgs, ", "))
		if err := bot.getRepositories(ctx, page, perPage, affiliation); err != nil {
			logrus.Errorf("Failed to get repos, %v\n", err)
			return err
		}
	}

	ghRecords := []githubRecord{}
	if err := bot.airtableClient.ListRecords(airtableTableName, &ghRecords); err != nil {
		return fmt.Errorf("listing records for table %s failed: %v", airtableTableName, err)
	}

	since, err := time.Parse("2006-01-02T15:04:05Z", watchSince)
	if err != nil {
		return err
	}

	for _, record := range ghRecords {
		if record.Fields.Updated.After(since) {
			since = record.Fields.Updated
		}
	}

	// if we are in watching mode, get your watched repositories
	if watched {
		page := 1
		perPage := 100
		logrus.Info("getting repositories watched...")

		if err := bot.getWatchedRepositories(ctx, page, perPage, since); err != nil {
			return err
		}
	}

	// Iterate over the records.
	for _, record := range ghRecords {
		// Parse the reference.
		user, repo, id, err := parseReference(record.Fields.Reference)
		if err != nil {
			logrus.Infof("Reference for %v failed:\n%v\n", record, err)
			continue
		}

		// Get the github issue.
		var issue *github.Issue

		// Check if we already have it from autofill or watched.
		if autofill || watched {
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
				if strings.Contains(err.Error(), "404 Not Found") {
					// Delete it from the table, the repo has probably moved or something.
					if err := bot.airtableClient.DestroyRecord(airtableTableName, record.ID); err != nil {
						logrus.Warnf("destroying record %s failed: %v", record.ID, err)
					}
					continue
				}
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
			logrus.Errorf("Failed to apply record to table for reference %s because %v\n", key, err)
			continue
		}
	}

	return nil
}

func (bot *bot) applyRecordToTable(ctx context.Context, issue *github.Issue, key, id string) error {
	// Trim surrounding quotes from ID string.
	id = strings.Trim(id, "\"")

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
		ID: id,
		Fields: Fields{
			Reference:  key,
			Title:      issue.GetTitle(),
			Body:       issue.GetBody(),
			State:      issue.GetState(),
			Author:     issue.GetUser().GetLogin(),
			Type:       issueType,
			Comments:   issue.GetComments(),
			URL:        issue.GetHTMLURL(),
			Updated:    issue.GetUpdatedAt(),
			Created:    issue.GetCreatedAt(),
			Completed:  issue.GetClosedAt(),
			Repository: repo,
		},
		Typecast: true,
	}

	// Update the record fields.
	fields := map[string]interface{}{
		"Reference":  record.Fields.Reference,
		"Title":      record.Fields.Title,
		"Body":       record.Fields.Body,
		"State":      record.Fields.State,
		"Author":     record.Fields.Author,
		"Type":       record.Fields.Type,
		"Comments":   record.Fields.Comments,
		"URL":        record.Fields.URL,
		"Updated":    record.Fields.Updated,
		"Created":    record.Fields.Created,
		"Completed":  record.Fields.Completed,
		"Repository": record.Fields.Repository,
	}

	if id != "" {
		// If we were passed a record ID, update the record instead of create.
		logrus.Debugf("updating record %s for issue %s", id, key)
		if err := bot.airtableClient.UpdateRecord(airtableTableName, id, fields, &record, true); err != nil {
			logrus.Warnf("updating record %s for issue %s failed: %v", id, key, err)
			return nil
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
	if err := bot.airtableClient.UpdateRecord(airtableTableName, record.ID, fields, &record, true); err != nil {
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
		// logrus.Debugf("checking if %s is in (%s)", repo.GetOwner().GetLogin(), strings.Join(orgs, " | "))
		if in(orgs, repo.GetOwner().GetLogin()) {
			logrus.Debugf("getting issues for repo %s...", repo.GetFullName())
			ipage := 0
			since, err := time.Parse("2006-01-02T15:04:05Z", "1900-01-02T15:04:05Z")
			if err != nil {
				return err
			}
			if !autofill {
				since = repo.UpdatedAt.Time
			}
			if err := bot.getIssues(ctx, ipage, perPage, repo.GetOwner().GetLogin(), repo.GetName(), since); err != nil {
				logrus.Debugf("Failed to get issues for repo %s - %v\n", repo.GetName(), err)
				return err
			}
			logrus.Debugf("Total issues: %d...", len(bot.issues))
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return bot.getRepositories(ctx, page, perPage, affiliation)
}

func (bot *bot) getWatchedRepositories(ctx context.Context, page, perPage int, since time.Time) error {
	opt := &github.ListOptions{
		Page:    page,
		PerPage: perPage,
	}

	repos, resp, err := bot.ghClient.Activity.ListWatched(ctx, "", opt)
	if err != nil {
		return err
	}

	for _, repo := range repos {
		// logrus.Debugf("checking if %s is in (%s)", repo.GetOwner().GetLogin(), strings.Join(orgs, " | "))
		// logrus.Debugf("getting issues for repo %s...", repo.GetFullName())
		ipage := 0
		if err := bot.getIssues(ctx, ipage, perPage, repo.GetOwner().GetLogin(), repo.GetName(), since); err != nil {
			return err
		}
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return bot.getWatchedRepositories(ctx, page, perPage, since)
}

func (bot *bot) getIssues(ctx context.Context, page, perPage int, owner, repo string, since time.Time) error {
	opt := &github.IssueListByRepoOptions{
		State: "all",
		Since: since,
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

		if verboseKeys {
			key = fmt.Sprintf("%s/%s#%d - %s", owner, repo, issue.GetNumber(), issue.GetTitle())
		}

		// logrus.Debugf("handling issue %s...", key)
		bot.issues[key] = issue
	}

	// Return early if we are on the last page.
	if page == resp.LastPage || resp.NextPage == 0 {
		return nil
	}

	page = resp.NextPage
	return bot.getIssues(ctx, page, perPage, owner, repo, since)
}

func parseReference(ref string) (string, string, int, error) {
	// Split the reference into repository and issue number.
	parts := strings.SplitN(ref, "#", 2)
	verboseParts := strings.SplitN(parts[1], " - ", 2)
	parts[1] = verboseParts[0]

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

func in(a stringSlice, s string) bool {
	for _, b := range a {
		if b == s {
			return true
		}
	}
	return false
}
