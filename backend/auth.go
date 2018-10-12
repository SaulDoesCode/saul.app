package backend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
)

var (
	// ErrInvalidUsernameOrEmail bad username or email details
	ErrInvalidUsernameOrEmail = errors.New("bad username and/or email")
	// ErrUnauthorized what ever happened it was not authorized
	ErrUnauthorized = errors.New("unauthorized request")
	// ErrIncompleteUser user is half baked, best get them in the DB before doing funny stuff
	ErrIncompleteUser = errors.New("cannot mutate a user that is incomplete or not in database")
)

// Role auth roles/perms
type Role = uint64

const (
	// UnverifiedUser user with unconfirmed email
	UnverifiedUser Role = iota + 1
	// VerifiedUser a verified user
	VerifiedUser
	// Admin boss man with all the perms
	Admin
)

var (
	// VerifiedSubject auth email subject
	VerifiedSubject string
	// UnverifiedSubject auth email subject for first timers
	UnverifiedSubject string
	// UnauthorizedError unauthorized request, cannot proceed
	UnauthorizedError = sendError("unauthorized request, cannot proceed")
	// InvalidDetailsError invalid details, could not authorize user
	InvalidDetailsError = sendError("invalid details, could not authorize user")
	// BadUsernameError invalid username, could not authorize user
	BadUsernameError = sendError("invalid username, could not authorize user")
	// BadEmailError invalid email, could not authorize user
	BadEmailError = sendError("invalid email, could not authorize user")
	// BadRequestError bad request, check details and try again
	BadRequestError = sendError("bad request, check details and try again")
	// ServerJSONError ran into trouble decoding your request
	ServerJSONError = sendError("ran into trouble decoding your request")
	// ServerDBError server error, could not complete your request
	ServerDBError = sendError("server error, could not complete your request")
)

// Common DB Queries
var (
	CreateUser = `INSERT {
		email: @email,
		emailmd5: @emailmd5,
		username: @username,
		description: @description,
		verifier: @verifier,
		created: DATE_NOW(),
		logins: [DATE_NOW()],
		roles: [0]
	} INTO users OPTIONS {
		waitForSync: true
	} RETURN NEW`
	FindUSERByUsername = `FOR u IN users FILTER u.username == @username RETURN u`
	FindUSERByEmail    = `FOR u IN users FILTER u.email == @email RETURN u`
)

// User struct describing a user account
type User struct {
	Key         string      `json:"_key,omitempty"`
	Email       string      `json:"email"`
	EmailMD5    string      `json:"emailmd5"`
	Username    string      `json:"username"`
	Description string      `json:"description,omitempty"`
	Verifier    string      `json:"verifier,omitempty"`
	Created     time.Time   `json:"created"`
	Logins      []time.Time `json:"logins,omitempty"`
	Roles       []uint64    `json:"roles,omitempty"`
	Friends     []string    `json:"friends,omitempty"`
	Exp         uint64      `json:"exp,omitempty"`
}

// IsValid check that the user's username and email are valid
func (user *User) IsValid() bool {
	return validUsernameAndEmail(user.Username, user.Email)
}

// Update update a user's details using a common map
func (user *User) Update(query string, vars obj) error {
	if len(user.Key) < 0 {
		return ErrIncompleteUser
	}
	vars["key"] = user.Key
	query = "FOR u in users FILTER u._key == @key UPDATE u WITH " + query + " IN users OPTIONS {keepNull: false, waitForSync: true} RETURN NEW"
	ctx := driver.WithQueryCount(context.Background())
	cursor, err := DB.Query(ctx, query, vars)
	defer cursor.Close()
	if err == nil {
		_, err = cursor.ReadDocument(ctx, user)
	}
	return err
}

// UserByKey retrieve user using their db document key
func UserByKey(key string) (User, error) {
	var user User
	_, err := Users.ReadDocument(context.Background(), key, &user)
	return user, err
}

// UserByUsername get user with a certain username
func UserByUsername(username string) (User, error) {
	var user User
	err := QueryOne(FindUSERByUsername, obj{"username": username}, &user)
	return user, err
}

// UserByEmail get user with a certain email
func UserByEmail(email string) (User, error) {
	var user User
	err := QueryOne(FindUSERByEmail, obj{"email": email}, &user)
	return user, err
}

// IsUsernameAvailable checks that the username is as of yet unused
func IsUsernameAvailable(username string) bool {
	if validUsername(username) {
		_, err := UserByUsername(username)
		return err != nil
	}
	return false
}

func createUser(email, username string) (User, error) {
	var user User

	if IsUsernameAvailable(username) && !validEmail(email) {
		return user, ErrInvalidUsernameOrEmail
	}

	err := QueryOne(CreateUser, obj{
		"email":    email,
		"emailMD5": GetMD5Hash(email),
		"username": username,
		"roles":    []uint64{UnverifiedUser},
		"verifier": RandStr(VerifierSize),
	}, &user)

	if err != nil {
		if DevMode {
			fmt.Println("createUser - error: ", err)
		}
		return user, err
	}

	link := "https://saul.app/auth/" + user.Username + "/" + user.Verifier + "/web"
	if DevMode {
		link = "https://localhost:" + Config.Get("devPort").String() + "/auth/" + user.Username + "/" + user.Verifier + "/web"
	}

	vars := obj{
		"AppName":  AppName,
		"Username": user.Username,
		"Link":     link,
		"Verifier": user.Verifier,
	}
	emailtxt, err := execTemplate(AuthEmailTXT, vars)
	if err != nil {
		return user, err
	}
	emailhtml, err := execTemplate(AuthEmailHTML, vars)
	if err != nil {
		return user, err
	}

	err = SendEmail(&Email{
		To:      []string{user.Email},
		Subject: UnverifiedSubject,
		Text:    emailtxt,
		HTML:    emailhtml,
	})

	if err != nil {
		if DevMode {
			fmt.Println("createUser - emailing error: ", err)
		}
	}

	return user, err
}

func authenticateUser(user *User) error {
	user.Verifier = RandStr(VerifierSize)
	err := user.Update(`{verifier: @verifier}`, obj{"verifier": user.Verifier})
	if err != nil {
		return err
	}

	link := "https://saul.app/auth/" + user.Username + "/" + user.Verifier + "/web"
	if DevMode {
		link = "https://localhost:" + Config.Get("devPort").String() + "/auth/" + user.Username + "/" + user.Verifier + "/web"
	}

	vars := obj{
		"AppName":  AppName,
		"Username": user.Username,
		"Link":     link,
		"Verifier": user.Verifier,
	}
	emailtxt, err := execTemplate(AuthEmailTXT, vars)
	if err != nil {
		return err
	}
	emailhtml, err := execTemplate(AuthEmailHTML, vars)
	if err != nil {
		return err
	}

	err = SendEmail(&Email{
		To:      []string{user.Email},
		Subject: VerifiedSubject,
		Text:    emailtxt,
		HTML:    emailhtml,
	})

	return err
}

func verifyUser(user *User, verifier string) error {
	if user.Verifier != verifier {
		return ErrUnauthorized
	}
	return user.Update(`{
		verifier: null,
		roles: PUSH(REMOVE_VALUE(u.roles, @unverified), @verified)
	}`, obj{
		"unverified": UnverifiedUser,
		"verified":   VerifiedUser,
	})
}

// GenerateAuthToken create a branca token
func GenerateAuthToken(payload string) string {
	token, err := Tokenator.Encode(payload)
	if err != nil {
		panic(err)
	}
	return token
}

// ValidateAuthToken and return a user if ok
func ValidateAuthToken(token string) (User, bool) {
	var user User
	payload, _, err := Tokenator.Decode(token)
	ok := err == nil
	if ok {
		user, err = UserByKey(payload)
		ok = err == nil
	}
	return user, ok
}

func initAuth() {
	VerifiedSubject = "Login to " + AppName
	UnverifiedSubject = "Welcome to " + AppName

	Server.GET("/check-username/:username", func(c ctx) error {
		return c.JSON(200, obj{"ok": IsUsernameAvailable(c.Param("username"))})
	})

	Server.POST("/auth", func(c ctx) error {
		body, err := JSONbody(c)
		if err != nil {
			return BadRequestError(c)
		}

		email := body.Get("email").String()
		if !validEmail(email) {
			return BadEmailError(c)
		}

		username := body.Get("username").String()
		if !validUsername(username) {
			return BadUsernameError(c)
		}

		user, err := UserByEmail(email)
		if err == nil {
			if user.Username != username {
				return InvalidDetailsError(c)
			}
			err = authenticateUser(&user)
		} else {
			user, err = createUser(email, username)
		}

		if err == nil {
			return c.JSONBlob(203, []byte(`{"msg": "Thanks `+user.Username+`, we sent you an authentication email."}`))
		} else if DevMode {
			fmt.Println("Authentication Problem: \n\tusername - ", username, "\n\temail - ", email, "\n\terror - ", err)
		}

		return UnauthorizedError(c)
	})

	Server.GET("/auth/:username/:verifier/:mode", func(c ctx) error {
		username := c.Param("username")
		verifier := c.Param("verifier")

		if len(verifier) != VerifierSize {
			return BadRequestError(c)
		}

		if !validUsername(username) {
			return BadUsernameError(c)
		}

		user, err := UserByUsername(username)
		if err != nil {
			return UnauthorizedError(c)
		}

		err = verifyUser(&user, verifier)
		if err != nil {
			if DevMode {
				fmt.Println("verifyUser: ", err)
			}
			return UnauthorizedError(c)
		}

		token := GenerateAuthToken(user.Key)
		if c.Param("mode") == "web" {
			return c.HTML(203, `<!DOCTYPE html><html><head><meta charset="utf-8"><title>`+AppName+` Auth Redirect</title><script>localStorage.setItem("token", "`+token+`"); localStorage.setItem("username", "`+user.Username+`");location.replace("/")</script></head></html>`)
		}
		return c.JSON(203, obj{"token": token, "username": username})
	})

	fmt.Println("Auth Handling Started")
}
