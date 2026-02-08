package quip

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GetThread retrieves a thread by ID
func (q *Client) GetThread(id string) *Thread {
	resp := q.getJson(apiUrlResource("threads/"+id), map[string]interface{}{})
	parsed := parseJsonObject(resp)
	return hydrateThread(parsed)
}

// EditDocument edits a document with the given parameters
func (q *Client) EditDocument(params *EditDocumentParams) *Thread {
	requestParams := make(map[string]interface{})
	required(params.Content, "Content is required for /edit-document")
	requestParams["content"] = params.Content

	required(params.ThreadId, "ThreadId is required for /edit-document")
	requestParams["thread_id"] = params.ThreadId

	setOptional(params.SectionId, "section_id", &requestParams)
	setOptional(params.Format, "format", &requestParams)
	setOptional(params.Location, "location", &requestParams)

	resp := q.postJson(apiUrlResource("threads/edit-document"), requestParams)

	return hydrateThread(resp)
}

// CreateDocument creates a new document (not yet implemented)
func (c *Client) CreateDocument(title, content string) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// UpdateDocument updates a document's content
func (c *Client) UpdateDocument(threadID, content string) error {
	threadID = normalizeThreadID(threadID)

	result := c.EditDocument(&EditDocumentParams{
		ThreadId: threadID,
		Content:  content,
		Format:   "markdown",
	})

	if result == nil {
		return fmt.Errorf("failed to update document")
	}

	return nil
}

// AppendToDocument appends content to a document
func (c *Client) AppendToDocument(threadID, content string) error {
	threadID = normalizeThreadID(threadID)

	result := c.EditDocument(&EditDocumentParams{
		ThreadId: threadID,
		Content:  "\n" + content,
		Format:   "markdown",
	})

	if result == nil {
		return fmt.Errorf("failed to append to document")
	}

	return nil
}

// AppendTableToDocument appends an HTML table to the end of a document
func (c *Client) AppendTableToDocument(threadID, htmlTable string) error {
	return c.AppendTableToDocumentAfter(threadID, htmlTable, "")
}

// AppendTableToDocumentAfter appends an HTML table after a specific section
func (c *Client) AppendTableToDocumentAfter(threadID, htmlTable, after string) error {
	threadID = normalizeThreadID(threadID)

	thread := c.GetThread(threadID)
	if thread == nil {
		return fmt.Errorf("failed to get thread")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(thread.Html))
	if err != nil {
		return err
	}

	var lastSectionID string
	after = strings.ToLower(after)
	if after != "" {
		// Find the specified section ID to append after
		sectionFound := false
		doc.Find("[id]").Each(func(i int, s *goquery.Selection) {
			if id, ok := s.Attr("id"); ok {
				if strings.ToLower(s.Text()) == after {
					sectionFound = true
					lastSectionID = id
					fmt.Printf("Found specified section ID to append after: %s\n", id)
				}
			}
		})
		if !sectionFound {
			return fmt.Errorf("specified section ID to append after not found: %s", after)
		}
	} else {
		// find the last section ID in the document if "after" not defined
		doc.Find("[id]").Last().Each(func(i int, s *goquery.Selection) {
			if id, ok := s.Attr("id"); ok {
				lastSectionID = id
				fmt.Printf("Found section ID: %s, content %s\n", id, s.Text())
			}
		})
	}

	if !strings.Contains(htmlTable, "<table") {
		htmlTable = "<table>" + htmlTable + "</table>"
	}

	params := &EditDocumentParams{
		ThreadId: threadID,
		Content:  htmlTable,
		Format:   "html",
		Location: AFTER_SECTION,
	}

	if lastSectionID != "" {
		params.SectionId = lastSectionID
	}

	result := c.EditDocument(params)
	if result == nil {
		return fmt.Errorf("failed to append table")
	}

	return nil
}

// AppendHTMLTableAsSpreadsheet appends an HTML table as a spreadsheet to a document
func (c *Client) AppendHTMLTableAsSpreadsheet(threadID, htmlTable string) error {
	return c.AppendTableToDocument(threadID, htmlTable)
}
