package backend

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	// ErrEmailRateLimit too many send requests on a particular email
	ErrEmailRateLimit = errors.New("too many emails sent to this address in a short time")
)

// Role auth roles/perms
type Role = int64

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
		created: DATE_NOW(),
		logins: [],
		sessions: [],
		roles: @roles
	} INTO users OPTIONS {waitForSync: true} RETURN NEW`
	FindUSERByUsername = `FOR u IN users FILTER u.username == @username RETURN u`
	FindUSERByEmail    = `FOR u IN users FILTER u.email == @email RETURN u`
	FindUserByDetails  = `FOR u IN users FILTER u.email == @email && u.username == @username RETURN u`
)

// User struct describing a user account
type User struct {
	Key         string   `json:"_key,omitempty"`
	Email       string   `json:"email"`
	EmailMD5    string   `json:"emailmd5"`
	Username    string   `json:"username"`
	Description string   `json:"description,omitempty"`
	Verifier    string   `json:"verifier,omitempty"`
	Created     int64    `json:"created,omitempty"`
	Logins      []int64  `json:"logins,omitempty"`
	Sessions    []int64  `json:"sessions,omitempty"`
	Roles       []Role   `json:"roles,omitempty"`
	Friends     []string `json:"friends,omitempty"`
	Exp         int64    `json:"exp,omitempty"`
	Subscriber  bool     `json:"subscriber,omitempty"`
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
	if err == nil {
		defer cursor.Close()
		_, err = cursor.ReadDocument(ctx, user)
		if err != nil && DevMode {
			fmt.Println("error updating user: ", err)
		}
	}
	return err
}

// HasRole check that a user has a particular auth role
func (user *User) HasRole(role Role) bool {
	for _, val := range user.Roles {
		if val == role {
			return true
		}
	}
	return false
}

// HasRoles check that a user has particular auth roles
func (user *User) HasRoles(roles []Role) bool {
	milestones := 0
	requires := len(roles)
	for _, val := range user.Roles {
		for _, role := range roles {
			if val == role {
				milestones++
			}
		}
	}
	return milestones == requires
}

// Verified check that a user has verified their email at least once
func (user *User) Verified() bool {
	return user.HasRole(VerifiedUser)
}

// Verified check that a user has verified their email at least once
func (user *User) isAdmin() bool {
	return user.HasRole(Admin)
}

// SetupVerifier initiate verification process with verifier and db update
func (user *User) SetupVerifier() error {
	return user.Update("{verifier: @verifier}", obj{
		"verifier": GenerateVerifier(user.Key),
	})
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
	if !validUsername(username) {
		return user, ErrInvalidUsernameOrEmail
	}
	err := QueryOne(FindUSERByUsername, obj{"username": username}, &user)
	return user, err
}

// UserByEmail get user with a certain email
func UserByEmail(email string) (User, error) {
	var user User
	if !validEmail(email) {
		return user, ErrInvalidUsernameOrEmail
	}
	err := QueryOne(FindUSERByEmail, obj{"email": email}, &user)
	return user, err
}

// UserByDetails attempt to get a user via their email/username combo
func UserByDetails(email, username string) (User, error) {
	var user User
	if !validEmail(email) || !validUsername(username) {
		return user, ErrInvalidUsernameOrEmail
	}
	err := QueryOne(FindUserByDetails, obj{
		"email":    email,
		"username": username,
	}, &user)
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

// AuthenticateUser create and/or authenticate a user
func AuthenticateUser(email, username string) (User, error) {
	user, err := UserByDetails(email, username)
	if err != nil {
		if DevMode {
			fmt.Println("Authentication no user with those details - error: ", err)
		}

		if IsUsernameAvailable(username) && !validEmail(email) {
			return user, ErrInvalidUsernameOrEmail
		}

		user = User{}
		err = QueryOne(CreateUser, obj{
			"email":    email,
			"emailmd5": GetMD5Hash(email),
			"username": username,
			"roles":    []Role{UnverifiedUser},
		}, &user)
		if err != nil {
			if DevMode {
				fmt.Println("\nAutentication - error: ", err, "\nuser:\t\n", user, "\n\t")
			}
			return user, err
		}
	}

	if !ratelimitEmail(email, 3, time.Minute*5) {
		return user, ErrEmailRateLimit
	}

	err = user.SetupVerifier()
	if err != nil {
		if DevMode {
			fmt.Println("Autentication verifier setup troubles - error: ", err)
		}
		return user, err
	}

	link := "https://" + AppDomain + "/auth/" + user.Verifier
	if DevMode {
		link = "https://localhost:" + Config.Get("devPort").String() + "/auth/" + user.Verifier
	}

	vars := obj{
		"AppName":  AppName,
		"Username": user.Username,
		"Link":     link,
		"Verifier": user.Verifier,
		"Domain":   AppDomain,
	}
	emailtxt, err := execTemplate(AuthEmailTXT, vars)
	if err != nil {
		if DevMode {
			fmt.Println("Autentication email text template - error: ", err)
		}
		return user, err
	}
	emailhtml, err := execTemplate(AuthEmailHTML, vars)
	if err != nil {
		if DevMode {
			fmt.Println("Autentication email html template - error: ", err)
		}
		return user, err
	}

	mail := MakeEmail()

	mail.To(user.Email)
	if user.Verified() {
		mail.Subject(VerifiedSubject)
	} else {
		mail.Subject(UnverifiedSubject)
	}

	mail.HTML().Set(string(emailhtml[:len(emailhtml)]))
	mail.Plain().Set(string(emailtxt[:len(emailtxt)]))

	err = SendEmail(mail)
	if err != nil && DevMode {
		fmt.Println(`Could not send email to `+user.Email+` because: `, err)
	}
	return user, err
}

// GenerateVerifier create a branca token
func GenerateVerifier(key string) string {
	token, err := Verinator.Encode(key)
	if err != nil {
		panic(err)
	}
	return token
}

// VerifyUser from a verifier token, check that a user has verified their email at least once
func VerifyUser(verifier string) (*User, error) {
	var user *User
	tk, err := Verinator.Decode(verifier)
	if err != nil {
		if DevMode {
			fmt.Println(`VerifyUser Decoding Error: `, err)
		}
		return user, ErrUnauthorized
	}
	usr, err := UserByKey(tk.Payload)
	user = &usr
	if err != nil || user.Verifier != verifier {
		if DevMode {
			fmt.Println(`VerifyUser Error - either no such user or the verifier didn't match: `, err)
		}
		return user, ErrUnauthorized
	}

	if user.Verified() {
		err = user.Update(`{verifier: null}`, obj{})
	} else {
		err = user.Update(`{
			verifier: null,
			roles: PUSH(REMOVE_VALUE(u.roles, @unverified), @verified, true)
		}`, obj{
			"unverified": UnverifiedUser,
			"verified":   VerifiedUser,
		})
	}

	if err != nil && DevMode {
		fmt.Println(`VerifyUser Error: `, err)
		panic(err)
	}
	return user, err
}

// GenerateAuthToken create a branca token
func GenerateAuthToken(user *User, renew bool) (string, error) {
	now := time.Now()
	nowunix := now.Unix()
	token, err := Tokenator.EncodeWithTime(user.Key, now)
	if err != nil {
		panic(err)
	}
	vars := obj{}

	if len(user.Sessions) > 0 {
		for i, session := range user.Sessions {
			if time.Unix(session, 0).Add(oneweek).After(now) {
				user.Sessions = append(user.Sessions[:i], user.Sessions[i+1:]...)
			}
		}
	}
	vars["sessions"] = append(user.Sessions, nowunix)
	if renew {
		err = user.Update(`{sessions: @sessions}`, vars)
	} else {
		vars["now"] = nowunix
		err = user.Update(`{sessions: @sessions, logins: PUSH(u.logins, @now)}`, vars)
	}
	return token, err
}

// ValidateAuthToken and return a user if ok
func ValidateAuthToken(token string) (User, bool) {
	var user User
	tk, err := Tokenator.Decode(token)
	ok := err == nil
	if !ok {
		return user, ok
	}
	user, err = UserByKey(tk.Payload)
	ok = err == nil && len(user.Sessions) < 1
	if !ok {
		return user, ok
	}
	// guilty until proven innocent here unfortunately
	ok = false
	now := time.Now()
	for i, session := range user.Sessions {
		if time.Unix(session, 0).Add(oneweek).After(now) {
			user.Sessions = append(user.Sessions[:i], user.Sessions[i+1:]...)
		} else if tk.Timestamp == session {
			ok = true
		}
	}
	ok = user.Update(`{sessions: @sessions}`, obj{"sessions": user.Sessions}) == nil
	return user, ok
}

// CredentialCheck get an authorized user from a route handler's context
func CredentialCheck(c ctx) (*User, error) {
	cookie, err := c.Cookie("Auth")
	if err != nil || cookie == nil {
		if DevMode {
			fmt.Println("CredentialCheck cookie troubles - error: ", err)
		}
		return nil, ErrUnauthorized
	}

	tk, err := Tokenator.Decode(cookie.Value)
	if err != nil {
		if DevMode {
			fmt.Println("CredentialCheck Decoding - error: ", err)
		}
		return nil, ErrUnauthorized
	}

	user, err := UserByKey(tk.Payload)
	if err != nil {
		if DevMode {
			fmt.Println("CredentialCheck User retrieval - error: ", err)
		}
		return nil, ErrUnauthorized
	}

	if tk.ExpiresBefore(time.Now().Add(time.Hour * 48)) {
		// refresh the auth token if it's about to go bad

		newtoken, err := GenerateAuthToken(&user, true)
		if err == nil {
			authCookie := &http.Cookie{
				Name:     "Auth",
				Value:    newtoken,
				Expires:  time.Now().Add(time.Hour * (24 * 7)),
				MaxAge:   60 * 60 * 24 * 7,
				Path:     "/",
				HttpOnly: true,
			}
			if !DevMode {
				authCookie.Domain = AppDomain
				authCookie.SameSite = http.SameSiteStrictMode
			}
			c.SetCookie(authCookie)
		} else {
			if DevMode {
				fmt.Println(`error Renewing Auth Token, it probably has something to do with the db`)
			}
		}
	}

	return &user, err
}

// AuthHandle create a GET route, accessible only to authenticated users
func AuthHandle(handle func(ctx, *User) error) func(ctx) error {
	return func(c ctx) error {
		user, err := CredentialCheck(c)
		if err != nil || user == nil {
			return UnauthorizedError(c)
		}
		return handle(c, user)
	}
}

// AdminHandle create a GET route, accessible only to admin users
func AdminHandle(handle func(ctx, *User) error) func(ctx) error {
	return func(c ctx) error {
		user, err := CredentialCheck(c)
		if err != nil || user == nil || !user.isAdmin() {
			if DevMode {
				fmt.Println(`AdminHandle for didn't go through: `, err)
			}
			return UnauthorizedError(c)
		}
		return handle(c, user)
	}
}

// RoleHandle create a GET route, accessible only to users with certain Roles
func RoleHandle(roles []Role, handle func(ctx, *User) error) func(ctx) error {
	return func(c ctx) error {
		user, err := CredentialCheck(c)
		if err != nil {
			return UnauthorizedError(c)
		}

		milestones := 0
		for _, authrole := range roles {
			for _, urole := range user.Roles {
				if urole == authrole {
					milestones++
				}
			}
		}

		if milestones == len(roles) {
			return handle(c, user)
		}
		return UnauthorizedError(c)
	}
}

func initAuth() {
	VerifiedSubject = "Login to " + AppName
	UnverifiedSubject = "Welcome to " + AppName

	Server.GET("/check-username/:username", func(c ctx) error {
		return c.JSON(200, obj{"ok": IsUsernameAvailable(c.Param("username"))})
	})

	Server.POST("/auth", func(c ctx) error {
		if _, err := CredentialCheck(c); err == nil {
			return c.JSON(203, obj{
				"msg": "You're already logged in :D",
				"ok":  true,
			})
		}

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

		user, err := AuthenticateUser(email, username)
		if err == nil {
			return c.JSON(203, obj{
				"msg": "Thanks" + user.Username + ", we sent you an authentication email.",
				"ok":  true,
			})
		} else if DevMode {
			fmt.Println("\nAuthentication Problem: \n\tusername - ", username, "\n\temail - ", email, "\n\terror - ", err, "\n\t")
		}

		if err == ErrEmailRateLimit {
			return c.JSON(429, obj{
				"msg": "too many auth requests/emails, wait a bit and try again",
				"ok":  false,
			})
		}

		return UnauthorizedError(c)
	})

	Server.GET("/auth/logout", func(c ctx) error {
		token := ""
		cookie, err := c.Cookie("Auth")
		if err == nil {
			token = cookie.Value
		}

		c.SetCookie(&http.Cookie{
			Name:     "Auth",
			Value:    "",
			Expires:  time.Now().Truncate(time.Hour),
			MaxAge:   1,
			Path:     "/",
			HttpOnly: true,
		})

		if len(token) > 0 {
			go func() {
				tk, err := Tokenator.Decode(token)
				if err != nil {
					return
				}

				user, err := UserByKey(tk.Payload)
				if err != nil {
					return
				}

				user.Update(
					`{session: REMOVE_VALUE(u.sessions, @session)}`,
					obj{"session": tk.Timestamp},
				)
			}()
		}

		return nil
	})

	Server.GET("/auth/:verifier", func(c ctx) error {
		user, err := VerifyUser(c.Param("verifier"))
		if err != nil || user == nil {
			if DevMode {
				fmt.Println("Unable to Authenticate user: ", err)
			}
			return UnauthorizedError(c)
		}

		newtoken, err := GenerateAuthToken(user, false)
		if err == nil {
			authCookie := &http.Cookie{
				Name:     "Auth",
				Value:    newtoken,
				Expires:  time.Now().Add(time.Hour * (24 * 7)),
				MaxAge:   60 * 60 * 24 * 7,
				Path:     "/",
				HttpOnly: true,
			}
			if !DevMode {
				authCookie.Domain = AppDomain
				authCookie.SameSite = http.SameSiteStrictMode
			}
			c.SetCookie(authCookie)
		} else {
			if DevMode {
				fmt.Println("error verifying (email) the user, GenerateAuthToken db problem: ", err)
			}
		}

		if user.isAdmin() {
			return c.Redirect(301, "/admin")
		}
		return c.Redirect(301, "/")
	})

	Server.GET("/subscribe-toggle", AuthHandle(func(c ctx, user *User) error {
		err := user.Update("{subscriber: @subscriber}", obj{"subscriber": !user.Subscriber})
		if err != nil {
			mail := MakeEmail()
			mail.To("saulvdw@gmail.com")
			mail.Subject("Subscriber State Toggle Error: " + user.Username)
			mail.HTML().Set(`
				<h4>There's been a problem updating user ` + user.Username + `'s subscriber status</h4>
				<p>err:<br>` + err.Error() + `</p>
			`)
			go SendEmail(mail)
			return c.JSON(203, obj{"msg": "something happened, don't worry, we'll figure it out", "ok": false})
		}
		msg := "success, you are "
		if user.Subscriber {
			msg += "subscribed for new writs and updates"
		} else {
			msg += "unsubscribed and will no longer receive any (non auth related) emails from grimstack.io."
		}
		return c.JSON(203, obj{"msg": msg, "ok": true})
	}))

	fmt.Println("Authentication Services Started")
	initAdmin()
}
