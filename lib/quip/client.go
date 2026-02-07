package quip

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	goquip "github.com/mduvall/go-quip"
)

type Client struct {
	quipClient *goquip.Client
}

func NewClient(accessToken string) *Client {
	return &Client{
		quipClient: goquip.NewClient(accessToken),
	}
}

func normalizeThreadID(id string) string {
	id = strings.TrimSpace(id)
	id = strings.TrimPrefix(id, "https://quip.com/")
	id = strings.TrimPrefix(id, "http://quip.com/")
	return id
}

func (c *Client) CreateDocument(title, content string) (string, error) {
	thread := c.quipClient.NewDocument(&goquip.NewDocumentParams{
		Title:   title,
		Content: content,
		Format:  "markdown",
	})

	if thread == nil {
		return "", fmt.Errorf("failed to create document")
	}

	return fmt.Sprintf("https://quip.com/%s", thread.Thread["id"]), nil
}

func (c *Client) UpdateDocument(threadID, content string) error {
	threadID = normalizeThreadID(threadID)

	result := c.quipClient.EditDocument(&goquip.EditDocumentParams{
		ThreadId: threadID,
		Content:  content,
		Format:   "markdown",
		// ReplaceAll: true,
	})

	if result == nil {
		return fmt.Errorf("failed to update document")
	}

	return nil
}

func (c *Client) AppendToDocument(threadID, content string) error {
	threadID = normalizeThreadID(threadID)

	result := c.quipClient.EditDocument(&goquip.EditDocumentParams{
		ThreadId: threadID,
		Content:  "\n" + content,
		Format:   "markdown",
	})

	if result == nil {
		return fmt.Errorf("failed to append to document")
	}

	return nil
}

func (c *Client) AppendTableToDocument(threadID, htmlTable string) error {
	threadID = normalizeThreadID(threadID)

	thread := c.quipClient.GetThread(threadID)
	if thread == nil {
		return fmt.Errorf("failed to get thread")
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(thread.Html))
	if err != nil {
		return err
	}

	var lastSectionID string
	doc.Find("[id]").Last().Each(func(i int, s *goquery.Selection) {
		if id, ok := s.Attr("id"); ok {
			lastSectionID = id
		}
	})

	if !strings.Contains(htmlTable, "<table") {
		htmlTable = "<table>" + htmlTable + "</table>"
	}

	params := &goquip.EditDocumentParams{
		ThreadId: threadID,
		Content:  htmlTable,
		Format:   "html",
	}

	if lastSectionID != "" {
		params.Location = lastSectionID
	}

	result := c.quipClient.EditDocument(params)
	if result == nil {
		return fmt.Errorf("failed to append table")
	}

	return nil
}

func (c *Client) AppendHTMLTableAsSpreadsheet(threadID, htmlTable string) error {
	return c.AppendTableToDocument(threadID, htmlTable)
}
