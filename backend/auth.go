package backend

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrInvalidUsernameOrEmail bad username or email details
	ErrInvalidUsernameOrEmail = errors.New("bad username and/or email")
	// ErrUnauthorized what ever happened it was not authorized
	ErrUnauthorized = errors.New("unauthorized request")
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

// User struct describing a user account
type User struct {
	ID          ObjID       `json:"-" bson:"_id,omitempty"`
	Email       string      `json:"email" bson:"email"`
	EmailMD5    string      `json:"emailmd5" bson:"emailmd5"`
	Username    string      `json:"username" bson:"username"`
	Description string      `json:"description,omitempty" bson:"description,omitempty"`
	Verifier    string      `json:"-" bson:"verifier,omitempty"`
	Created     time.Time   `json:"created" bson:"created"`
	Logins      []time.Time `json:"logins,omitempty" bson:"logins,omitempty"`
	Roles       []uint64    `json:"-" bson:"roles,omitempty"`
	Friends     []string    `json:"friends,omitempty" bson:"friends,omitempty"`
	Exp         uint64      `json:"exp,omitempty" bson:"exp,omitempty"`
}

// Update user details in the database
func (user *User) Update(Obj obj) error {
	return DB.Users.UpdateId(user.ID, Obj)
}

// IsValid check that the user's username and email are valid
func (user *User) IsValid() bool {
	return validUsernameAndEmail(user.Username, user.Email)
}

// IsUsernameAvailable checks that the username is as of yet unused
func IsUsernameAvailable(username string) bool {
	if validUsername(username) {
		n, err := DB.Users.Find(obj{"username": username}).Count()
		if err == nil && n == 0 {
			return true
		}
	}
	return false
}

func createUser(email, username string) (User, error) {
	var user User

	if !validUsernameAndEmail(username, email) {
		return user, ErrInvalidUsernameOrEmail
	}

	user = User{
		ID:       MakeID(),
		Email:    email,
		EmailMD5: GetMD5Hash(email),
		Username: username,
		Created:  time.Now(),
		Roles:    []uint64{UnverifiedUser},
		Verifier: RandStr(VerifierSize),
	}

	err := DB.Users.Insert(user)
	if DevMode {
		fmt.Println("createUser - error: ", err)
	}

	SendEmail(&Email{
		To: []string{user.Email},
	})

	return user, err
}

func authenticateUser(user *User) error {
	return ErrUnauthorized
}

func verifyUser(user *User, verifier string) error {
	if user.Verifier != verifier {
		return ErrUnauthorized
	}
	err := user.Update(obj{
		"$unset": obj{"verifier": verifier},
		"$pull":  obj{"roles": UnverifiedUser},
		"$addToSet": obj{
			"roles":  VerifiedUser,
			"logins": time.Now(),
		},
	})
	return err
}

// UserByID get user with a certain _id property
func UserByID(id string) (User, error) {
	var user User
	err := DB.Users.FindId(id).One(&user)
	return user, err
}

// UserByUsername get user with a certain username
func UserByUsername(username string) (User, error) {
	var user User
	err := DB.Users.Find(obj{"username": username}).One(&user)
	return user, err
}

// UserByEmail get user with a certain email
func UserByEmail(email string) (User, error) {
	var user User
	err := DB.Users.Find(obj{"email": email}).One(&user)
	return user, err
}

// GenerateAuthToken create a branca token
func GenerateAuthToken(userID string) string {
	token, err := Branca.EncodeToString(userID)
	if err != nil {
		panic(err)
	}
	return token
}

// ValidateAuthToken and return a user if ok
func ValidateAuthToken(token string) (User, bool) {
	var user User
	tkn, err := Branca.DecodeToken(token)
	ok := err == nil
	if ok {
		ok = DB.Users.FindId(tkn.Payload).One(&user) != nil
	}
	return user, ok
}

func initAuth() {
	VerifiedSubject = "Login to " + AppName
	UnverifiedSubject = "Welcome to " + AppName

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
			err = authenticateUser(&user)
		} else {
			user, err = createUser(email, username)
		}

		if err == nil {
			return c.JSON(203, obj{
				"msg": "Thank You " + user.Username + ", we sent you an authentication email.",
			})
		} else if DevMode {
			fmt.Println("Authentication Problem: \n\tusername - ", username, "\n\temail - ", email, "\n\terror - ", err)
		}

		return UnauthorizedError(c)
	})

	Server.GET("/auth/:user/:verifier/:mode", func(c ctx) error {
		username := c.Param("user")
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
			return UnauthorizedError(c)
		}

		token := GenerateAuthToken(user.ID.String())
		if c.Param("mode") == "web" {
			return c.HTML(
				203,
				`<!DOCTYPE html>
					<html>
					<head>
						<meta charset="utf-8">
						<title>`+AppName+` Auth Redirect</title>
	    			<script>
							localStorage.setItem("token", "`+token+`")
							localStorage.setItem("username", "`+user.Username+`")
							location.replace("/")
						</script>
	  			</head>
					</html>`,
			)
		}
		return c.JSON(203, obj{"token": token, "username": username})
	})
}
