package backend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Machiel/slugify"
	"github.com/arangodb/go-driver"
)

var (
	// ErrIncompleteWrit someone probably tried to mutate a writ that is invalid or non existing (in db)
	ErrIncompleteWrit = errors.New(`attempted to modify either an invalid writ, or one not in the db`)
	// ErrMissingTags writ is missing tags
	ErrMissingTags = errors.New(`writ doesn't have any tags, add some`)
	// ErrAuthorIsNoUser writ's author is persona non grata
	ErrAuthorIsNoUser = errors.New(`writ author is not a registered user`)
)

// Writ - struct representing a post or document in the database
type Writ struct {
	Key         string   `json:"_key,omitempty"`
	Type        string   `json:"type,omitempty"`
	Title       string   `json:"title,omitempty"`
	AuthorKey   string   `json:"authorkey,omitempty"`
	Author      string   `json:"author,omitempty"`
	Content     string   `json:"content,omitempty"`
	Injection   string   `json:"injection,omitempty"`
	Markdown    string   `json:"markdown,omitempty"`
	Description string   `json:"description,omitempty"`
	Slug        string   `json:"slug,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Edits       []int64  `json:"edits,omitempty"`
	Created     int64    `json:"created,omitempty"`
	Views       int64    `json:"views,omitempty"`
	ViewedBy    []string `json:"viewedby,omitempty"`
	LikedBy     []string `json:"likedby,omitempty"`
	Public      bool     `json:"public,omitempty"`
	MembersOnly bool     `json:"membersonly,omitempty"`
	NoComments  bool     `json:"nocomments,omitempty"`
	Roles       []Role   `json:"roles,omitempty"`
}

// Timeframe a distance of time between a .Start time and a .Finish time
type Timeframe struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

type writQuery struct {
	One         bool      `json:"one,omitempty"`
	Public      bool      `json:"public,omitempty"`
	EditorMode  bool      `json:"editormode,omitempty"`
	Extensive   bool      `json:"extensive,omitempty"`
	UpdateViews bool      `json:"updateviews,omitempty"`
	Comments    bool      `json:"comments,omitempty"`
	MembersOnly bool      `json:"membersonly,omitempty"`
	DontSort    bool      `json:"dontsort,omitempty"`
	Vars        obj       `json:"vars,omitempty"`
	ViewedBy    string    `json:"viewedby,omitempty"`
	LikedBy     string    `json:"likedby,omitempty"`
	Viewer      string    `json:"viewer,omitempty"`
	Title       string    `json:"title,omitempty"`
	Slug        string    `json:"slug,omitempty"`
	Author      string    `json:"author,omitempty"`
	Created     time.Time `json:"created,omitempty"`
	Between     Timeframe `json:"between,omitempty"`
	Roles       []Role    `json:"roles,omitempty"`
	Limit       []int64   `json:"limit,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Omissions   []string  `json:"omissions,omitempty"`
}

// Exec execute a writQuery to retrieve some/certain writs
func (q *writQuery) Exec() ([]Writ, error) {
	writs := []Writ{}
	if q.Vars == nil {
		q.Vars = obj{}
	}

	query := "FOR writ IN writs "

	filter := ""
	firstfilter := true

	if q.Public {
		filter += `writ.public == true `
		firstfilter = false
	}

	if q.MembersOnly {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		filter += `writ.membersonly == true `
	}

	if &q.Between != nil {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		startzero := q.Between.Start.IsZero()
		endzero := q.Between.End.IsZero()
		if !startzero && !endzero {
			q.Vars["@betweenStart"] = q.Between.Start.Unix()
			q.Vars["@betweenEnd"] = q.Between.Start.Unix()
			filter += "writ.created > @betweenStart && writ.created < @betweenEnd "
		} else if startzero {
			q.Vars["@betweenStart"] = q.Between.Start.Unix()
			filter += "writ.created > @betweenStart "
		} else if endzero {
			q.Vars["@betweenEnd"] = q.Between.Start.Unix()
			filter += "writ.created < @betweenEnd "
		}
	}

	if len(q.Author) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["author"] = q.Author
		filter += `writ.author == @author `
	}

	if len(q.ViewedBy) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["viewedby"] = q.ViewedBy
		filter += `@viewedby IN writ.viewedby `
	}

	if len(q.LikedBy) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["likedby"] = q.LikedBy
		filter += `@likedby IN writ.likedby `
	}

	if len(q.Roles) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["roles"] = q.Roles
		filter += `@roles ALL IN writ.roles `
	}

	if len(q.Tags) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["tags"] = q.Tags
		filter += `@tags ALL IN writ.tags `
	}

	if !firstfilter {
		query += "FILTER " + filter
	}

	if len(q.Limit) > 0 {
		q.Vars["pagenum"] = q.Limit[0]
		query += `LIMIT @pagenum `
		if len(q.Limit) == 2 {
			q.Vars["pagesize"] = q.Limit[1]
			query += `, @pagesize `
		}
	}

	if !q.DontSort {
		query += "SORT writ.created DESC "
	}

	query += "RETURN "

	final := "MERGE(writ, {likes: writ.likes + LENGTH(writ.likedby), views: writ.views + LENGTH(writ.viewedby)})"

	if !q.EditorMode {
		q.Omissions = append(q.Omissions, "_key", "markdown", "edits", "public", "roles")
	}

	if !q.Extensive {
		q.Omissions = append(q.Omissions, "likedby", "viewedby")
	}

	if len(q.Omissions) > 0 {
		q.Vars["omissions"] = q.Omissions
		final = "UNSET(" + final + ", @omissions)"
	}

	query += final

	if DevMode {
		fmt.Println("\n You're trying this query now: \n", query, "\n\t")
	}

	ctx := driver.WithQueryCount(context.Background())
	cursor, err := DB.Query(ctx, query, q.Vars)
	if err == nil {
		defer cursor.Close()
		for {
			var writ Writ
			_, err := cursor.ReadDocument(ctx, &writ)
			if driver.IsNoMoreDocuments(err) {
				break
			} else if err != nil {
				if DevMode {
					fmt.Println("DB Multiple Query - something strange happened: ", err)
				}
				panic(err)
			}
			writs = append(writs, writ)
		}
	} else if driver.IsNoMoreDocuments(err) {
		fmt.Println(`No more docs? Awww :( - `, err)
	} else if DevMode {
		fmt.Println("\n... And, it would seem that it has failed: \n", err, "\n\t")
	}
	return writs, err
}

// ExecOne execute a writQuery to retrieve a single writ
func (q *writQuery) ExecOne() (Writ, error) {
	var writ Writ

	if q.Vars == nil {
		q.Vars = obj{}
	}

	query := "FOR writ IN writs "

	filter := ""
	firstfilter := true

	if q.Public {
		filter += `writ.public == true `
		firstfilter = false
	}

	if q.MembersOnly {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		filter += `writ.membersonly == true `
	}

	if !q.Created.IsZero() {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["@created"] = q.Created.Unix()
		filter += "writ.created == @created "
	}

	if len(q.Slug) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["slug"] = q.Slug
		filter += `writ.slug == @slug `
	}

	if len(q.Title) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["title"] = q.Title
		filter += `writ.title == @title `
	}

	if len(q.Author) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["author"] = q.Author
		filter += `writ.author == @author `
	}

	if len(q.ViewedBy) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["viewedby"] = q.ViewedBy
		filter += `@viewedby IN writ.viewedby `
	}

	if len(q.LikedBy) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["likedby"] = q.LikedBy
		filter += `@likedby IN writ.likedby `
	}

	if len(q.Roles) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["roles"] = q.Roles
		filter += `@roles ALL IN writ.roles `
	}

	if len(q.Tags) > 0 {
		if !firstfilter {
			filter += "&& "
		}
		firstfilter = false
		q.Vars["tags"] = q.Tags
		filter += `@tags ALL IN writ.tags `
	}

	if !firstfilter {
		query += "FILTER " + filter
	}

	query += "RETURN "

	final := "MERGE(writ, {likes: writ.likes + LENGTH(writ.likedby), views: writ.views + LENGTH(writ.viewedby)})"

	if !q.EditorMode {
		q.Omissions = append(q.Omissions, "_key", "markdown", "edits", "public", "roles")
	}

	if !q.Extensive {
		q.Omissions = append(q.Omissions, "likedby", "viewedby")
	}

	if len(q.Omissions) > 0 {
		q.Vars["omissions"] = q.Omissions
		final = "UNSET(" + final + ", @omissions)"
	}

	query += final

	if DevMode {
		fmt.Println("\n You're trying this query now: \n", query, "\n\t")
	}

	err := QueryOne(query, q.Vars, &writ)

	if DevMode && err != nil {
		fmt.Println("\n... And, it would seem that it has failed: \n", err, "\n\t")
	}

	return writ, err
}

// Slugify generate and set .Slug from .Title
func (writ *Writ) Slugify() {
	writ.Slug = slugify.Slugify(writ.Title)
}

// RenderContent from .Markdown generate html and set .Content
func (writ *Writ) RenderContent() {
	writ.Content = string(renderMarkdown([]byte(writ.Markdown), false)[:])
}

// ToObj convert writ into map[string]interface{}
func (writ *Writ) ToObj(omissions ...string) obj {
	output := obj{}

	if len(writ.Key) != 0 {
		output["_key"] = writ.Key
	}
	if len(writ.Type) != 0 {
		output["type"] = writ.Type
	}
	if len(writ.Title) != 0 {
		output["title"] = writ.Title
	}
	if len(writ.AuthorKey) != 0 {
		output["authorkey"] = writ.AuthorKey
	}
	if len(writ.Author) != 0 {
		output["author"] = writ.Author
	}
	if len(writ.Content) != 0 {
		output["content"] = writ.Content
	}
	if len(writ.Injection) != 0 {
		output["injection"] = writ.Injection
	}
	if len(writ.Markdown) != 0 {
		output["markdown"] = writ.Markdown
	}
	if len(writ.Description) != 0 {
		output["description"] = writ.Description
	}
	if len(writ.Slug) != 0 {
		output["slug"] = writ.Slug
	}
	if len(writ.Tags) != 0 {
		output["tags"] = writ.Tags
	}
	if len(writ.Edits) != 0 {
		output["edits"] = writ.Edits
	}
	if writ.Created != 0 {
		output["created"] = writ.Created
	}
	if writ.Views != 0 {
		output["views"] = writ.Views
	}
	if len(writ.ViewedBy) != 0 {
		output["viewedby"] = writ.ViewedBy
	}
	if len(writ.LikedBy) != 0 {
		output["likedby"] = writ.LikedBy
	}
	if &writ.Public != nil {
		output["public"] = writ.Public
	}
	if &writ.MembersOnly != nil {
		output["membersonly"] = writ.MembersOnly
	}
	if &writ.NoComments != nil {
		output["nocomments"] = writ.NoComments
	}
	if len(writ.Roles) != 0 {
		output["roles"] = writ.Roles
	}

	if len(omissions) != 0 {
		for _, omission := range omissions {
			delete(output, omission)
		}
	}

	return output
}

// Update update a writ's details using a map[string]interface{}
func (writ *Writ) Update(query string, vars obj) error {
	if len(writ.Key) < 0 {
		return ErrIncompleteWrit
	}
	vars["key"] = writ.Key
	query = "FOR u in writs FILTER u._key == @key UPDATE u WITH " + query + " IN writs OPTIONS {keepNull: false, waitForSync: true} RETURN NEW"
	ctx := driver.WithQueryCount(context.Background())
	cursor, err := DB.Query(ctx, query, vars)
	defer cursor.Close()
	if err == nil {
		_, err = cursor.ReadDocument(ctx, writ)
	}
	return err
}

// WritByKey retrieve user using their db document key
func WritByKey(key string) (Writ, error) {
	var writ Writ
	_, err := Writs.ReadDocument(context.Background(), key, &writ)
	return writ, err
}

// InitWrit initialize a new writ
func InitWrit(w *Writ) error {
	if len(w.Tags) < 1 {
		return ErrMissingTags
	}

	ctx := driver.WithWaitForSync(context.Background(), true)

	exists := true
	var err error
	var currentWrit Writ
	if len(w.Key) == 0 {
		if DevMode {
			fmt.Println("Searching For: ", w.Title)
		}
		currentWrit, err = (&writQuery{
			EditorMode: true,
			Title:      w.Title,
		}).ExecOne()
		exists = err == nil
		err = nil
	}

	if !exists {
		w.Created = time.Now().Unix()
		if len(w.Markdown) < 1 || len(w.Title) < 1 || len(w.Author) < 1 {
			if DevMode {
				fmt.Println("InitWrit - it's horribly incomplete, fix it, add in author, title, and markdown")
			}
			return ErrIncompleteWrit
		}

		user, err := UserByUsername(w.Author)
		if err != nil {
			if DevMode {
				fmt.Println("InitWrit - author ("+w.Author+") is invalid or MIA: ", err)
			}
			return ErrAuthorIsNoUser
		}
		w.AuthorKey = user.Key

		w.RenderContent()
		if len(w.Slug) < 1 {
			w.Slugify()
		}

		meta, err := Writs.CreateDocument(ctx, w)
		if err != nil {
			if DevMode {
				fmt.Println(`InitWrit - creating a writ in the db: `, err)
			}
			return err
		}
		w.Key = meta.Key
	} else {
		if len(w.Key) == 0 {
			w.Key = currentWrit.Key
		}
		if len(w.Title) != 0 && currentWrit.Title == w.Title {
			w.Title = ""
			w.Slug = ""
		} else {
			w.Slugify()
		}
		if len(w.Content) != 0 && currentWrit.Content == w.Content {
			w.Content = ""
			w.Markdown = ""
		} else {
			w.RenderContent()
		}
		w.Edits = append(w.Edits, time.Now().Unix())
		ctx = driver.WithMergeObjects(ctx, true)
		_, err := Writs.UpdateDocument(ctx, w.Key, w.ToObj("_key"))
		if err != nil {
			if DevMode {
				fmt.Println(`InitWrit - error updating a writ in the db: `, err)
			}
			return err
		}
		if !currentWrit.Public && w.Public {
			go notifySubscribers(w.Key)
		}
	}

	return nil
}

func notifySubscribers(writKey string) {
	writ, err := WritByKey(writKey)
	if err != nil {
		return
	}

	query := `FOR u IN users FILTER u.subscriber == true RETURN u`
	ctx := driver.WithQueryCount(context.Background())
	var users []User
	cursor, err := DB.Query(ctx, query, obj{})
	if err != nil {
		return
	}
	defer cursor.Close()
	users = []User{}
	for {
		var doc User
		_, err = cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		} else if err != nil {
			return
		}
		users = append(users, doc)
	}

	mail := MakeEmail()
	mail.Subject("Subscriber Update: Newly Published Writ")
	domain := AppDomain
	if DevMode {
		domain = "localhost:2443"
	}
	mail.HTML().Set(`
		<h4>There's a new writ: ` + writ.Title + `</h4>
		<p><a href="https://` + domain + "/writ/" + writ.Slug + `">check it out</a></p>
		<sub><a href="https://` + domain + `/subscribe-toggle">unsubcribe</a></sub>
	`)
	for _, user := range users {
		mail.Bcc(user.Email)
	}
	go SendEmail(mail)
}

func initWrits() {
	Server.GET("/writ/:slug", func(c ctx) error {
		slug := c.Param("slug")

		wq := writQuery{
			Public:      true,
			UpdateViews: true,
			Slug:        slug,
		}

		user, err := CredentialCheck(c)
		isuser := true
		if err == nil {
			wq.Viewer = user.Key
		} else {
			isuser = false
			err = nil
		}

		writ, err := wq.ExecOne()

		if driver.IsNotFound(err) {
			return JSONErr(c, 404, "couldn't find a writ like that")
		} else if err != nil {
			return JSONErr(c, 503, err.Error())
		} else if writ.MembersOnly && !isuser {
			return UnauthorizedError(c)
		}

		writdata := writ.ToObj()

		createdate := time.Unix(writ.Created, 0)
		writdata["Created"] = createdate.Format("1 Jan 2006")
		writdata["CreateDate"] = createdate

		editslen := len(writ.Edits)
		if editslen != 0 {
			writdata["ModifiedDate"] = time.Unix(writ.Edits[editslen-1], 0)
		}

		writdata["URL"] = "https://" + AppDomain + "/writ/" + writ.Slug

		c.Response().Header().Set("Content-Type", "text/html")
		err = PostTemplate.Execute(c.Response(), &writdata)
		if err != nil {
			if DevMode {
				fmt.Println("GET /writ/:slug - error executing the post template: ", err)
			}
		}
		return err
	})

	Server.GET("/writs/", func(c ctx) error {
		return c.JSON(404, []obj{})
	})

	Server.POST("/writ", AdminHandle(func(c ctx, user *User) error {
		var writ Writ
		err := UnmarshalJSONBody(c, &writ)
		if err != nil {
			return BadRequestError(c)
		}

		err = InitWrit(&writ)
		if err != nil {
			if !driver.IsNoMoreDocuments(err) {
				return c.JSON(503, obj{"ok": false, "error": err})
			}
		}

		fmt.Println(`Baking Writs! - `, writ.Title)

		return c.JSON(203, obj{"ok": true, "msg": "sucess!"})
	}))

	Server.POST("/writ/query", AdminHandle(func(c ctx, user *User) error {
		var q writQuery
		err := UnmarshalJSONBody(c, &q)
		if err != nil {
			return BadRequestError(c)
		}

		var output interface{}
		if q.One {
			output, err = q.ExecOne()
		} else {
			output, err = q.Exec()
		}
		if err != nil {
			return c.JSON(503, obj{"ok": false, "error": err})
		}
		return c.JSON(200, output)
	}))
}
