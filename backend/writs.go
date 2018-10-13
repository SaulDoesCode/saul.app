package backend

import (
	"github.com/Machiel/slugify"
)

// Writ - struct representing a post or document in the database
type Writ struct {
	Key         string   `json:"_key"`
	Title       string   `json:"title"`
	Author      string   `json:"author"`
	Content     string   `json:"content"`
	Markdown    string   `json:"markdown,omitempty"`
	Description string   `json:"description"`
	Slug        string   `json:"slug"`
	Tags        []string `json:"tags"`
	Edits       []int64  `json:"edits"`
	Created     int64    `json:"created"`
	Views       int64    `json:"views"`
	Likes       int64    `json:"likes"`
	Public      bool     `json:"public"`
	MembersOnly bool     `json:"membersonly"`
	Roles       []int64  `json:"roles"`
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
