package backend

import (
	"time"

	"github.com/Machiel/slugify"
)

// Writ - struct representing a post or document in the database
type Writ struct {
	Key         string      `json:"_key"`
	Title       string      `json:"title"`
	Author      string      `json:"author"`
	Content     string      `json:"content"`
	Markdown    string      `json:"markdown"`
	Description string      `json:"description"`
	Slug        string      `json:"slug"`
	Tags        []string    `json:"tags"`
	Edits       []time.Time `json:"edits"`
	Created     time.Time   `json:"created"`
	Views       int64       `json:"views"`
	Likes       int64       `json:"likes"`
	Published   bool        `json:"published"`
}

// Slugify generate and set .Slug from .Title
func (writ *Writ) Slugify() {
	writ.Slug = slugify.Slugify(writ.Title)
}

// RenderContent from .Markdown generate html and set .Content
func (writ *Writ) RenderContent() {
	writ.Content = string(renderMarkdown([]byte(writ.Markdown))[:])
}

func initWrit(w *Writ) {
	w.Slugify()
	w.RenderContent()
}

func initWrits() {
	Server.GET("/writ", func(c ctx) error {
		return c.JSON(200, obj{
			"writs": "not yet but soon",
		})
	})
}
