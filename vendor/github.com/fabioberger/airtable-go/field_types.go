package airtable

// Attachment models the response returned by the Airtable API for an attachment field type. It can be
// used in your record type declarations that include attachment fields.
type Attachment struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	FileName   string `json:"filename"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
	Thumbnails struct {
		Small thumbnail `json:"small"`
		Large thumbnail `json:"large"`
	} `json:"thumbnails"`
}

type thumbnail struct {
	URL    string `json:"url"`
	Width  int64  `json:"width"`
	Height int64  `json:"height"`
}
