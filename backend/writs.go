package backend

import (
	"time"

	"github.com/Machiel/slugify"
)


// Writ - struct representing a post or document in the database
type Writ struct {
	ID          ObjID       `bson:"_id"`
	Title       string      `bson:"title"`
	Author      string      `bson:"author"`
	Content     string      `bson:"content"`
	Markdown    string      `bson:"markdown"`
	Description string      `bson:"description"`
	Slug        string      `bson:"slug"`
	Tags        []string    `bson:"tags"`
	Edits       []time.Time `bson:"edits"`
	Created     time.Time   `bson:"created"`
	Views       int64       `bson:"views"`
	Likes       int64       `bson:"likes"`
	Published   bool        `bson:"published"`
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
