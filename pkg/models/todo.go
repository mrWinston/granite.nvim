package models

import (
	"fmt"
	"regexp"
)

var STATE_MAP = map[string]string{
	"[]":  "OPEN",
	"[ ]": "OPEN",
	"[-]": "IN_PROGRESS",
	"[/]": "IN_PROGRESS",
	"[x]": "DONE",
	"[X]": "DONE",
}

type Todo struct {
	// RawLine is the complete unparse line from the markdown file
	RawLine string
	// Text contains the todo string without leading whitespace and -
	Text string `json:"text"`
	// Tags are the tags assigned to the todo
	Tags []string `json:"tags"`
	// LineNumber is the line in the file where the todo was found
	LineNumber int `json:"lnum"`
	// FilePath is the path to the file where the todo was found
	FilePath string `json:"filename"`
	// DueDate is the date when the todo is due. The format must be YYYY-MM-DD
	DueDate string `json:"due_date"`
	// StateString is a string representation of the State of the ToDo, eg: OPEN, DONE, IN_PROGRESS
	StateString string `json:"state"`
}

// Parse parses the RawLine set in the todo and populates all other fields based on what it finds there.
//
// Returns an error when RawLine, LineNumber or FilePath are unset or the RawLine can't be parsed into a todo
func (t *Todo) Parse() error {
	stateRegex := regexp.MustCompile(`(\[.?\])`)
	tagsRegex := regexp.MustCompile(`(#\w+)`)
	dateRegex := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	textRegex := regexp.MustCompile(`^.*(\[.?\].*)$`)

	stateRaw := stateRegex.FindString(t.RawLine)
	stateString, ok := STATE_MAP[stateRaw]
	if !ok {
		return fmt.Errorf("Couldn't parse todo state: '%s', line is: '%s'", stateRaw, t.RawLine)
	}

	t.StateString = stateString
	t.Tags = tagsRegex.FindAllString(t.RawLine, 100)
	if len(t.Tags) == 0 {
		return fmt.Errorf("Couldn't parse todo Tags: %s", t.RawLine)
	}

	t.DueDate = dateRegex.FindString(t.RawLine)
	t.Text = textRegex.FindStringSubmatch(t.RawLine)[1]
	return nil
}
