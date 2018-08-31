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
	if err != nil {
		if DevMode {
			fmt.Println("createUser - error: ", err)
		}
		return user, err
	}

	magicLink := "https://saul.app/auth/" + user.Username + "/" + user.Verifier + "/web"
	if DevMode {
		magicLink = "https://localhost:" + Config.Get("devPort").String() + "/auth/" + user.Username + "/" + user.Verifier + "/web"
	}

	err = SendEmail(&Email{
		To:      []string{user.Email},
		Subject: UnverifiedSubject,
		Text: []byte(`
Hi, ` + user.Username + `!

To login at ` + AppName + `, just follow this magic link:
` + magicLink + `
Note, this link will expire in about 15 minutes.
If it doesn't work try logging in again from https://saul.app.
		`),
		HTML: []byte(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1.0">
	<title>Verification Email</title>
</head>
<body style="font-family: Nunito, Verdunda, Helvetica, Roboto, sans-serif; text-align: center; color: hsl(0,0%,30%); background: hsl(0,0%,99%);">
	<main style="display: block; position: relative; margin: 15px auto; padding: 5px 15px 15px 15px; max-width: 420px; background: #FFF; box-shadow: 0 2px 8px hsla(0,0%,0%,.12); border-radius: 2.5px;">
		<h3>Hi there ` + user.Username + `!</h3>
		Please follow the verification link to login to ` + AppName + `.
		<br>
		<a href="` + magicLink + `" style="display: block; font-size: 1.2em; font-weight: 600; margin: 10px auto; max-width: 180px; padding: 8px; border-radius: 2.5px; text-decoration: none; color: #fff; background: hsl(0,0%,30%); box-shadow: 0 2px 6px hsla(0,0%,0%,.12); text-shadow: 0 1px 3px hsla(0,0%,0%,.12);">
			Login
		</a>
		<br>
		<footer>
			Note, this link will expire in about 15 minutes, just log in again from <a href="https://saul.app" style="color: inherit;">saul.app</a> if it doesn't work.
		</footer>
	</main>
</body>
</html>`),
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
	err := user.Update(obj{
		"$set": obj{"verifier": user.Verifier},
	})
	if err != nil {
		return err
	}

	magicLink := "https://saul.app/auth/" + user.Username + "/" + user.Verifier + "/web"
	if DevMode {
		magicLink = "https://localhost:" + Config.Get("devPort").String() + "/auth/" + user.Username + "/" + user.Verifier + "/web"
	}

	err = SendEmail(&Email{
		To:      []string{user.Email},
		Subject: UnverifiedSubject,
		Text: []byte(`
Hi, ` + user.Username + `!

To login at ` + AppName + `, just follow this magic link:
` + magicLink + `
Note, this link will expire in about 15 minutes.
If it doesn't work try logging in again from https://saul.app.
		`),
		HTML: []byte(`
<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width,initial-scale=1.0">
	<title>Verification Email</title>
</head>
<body style="font-family: Nunito, Verdunda, Helvetica, Roboto, sans-serif; text-align: center; color: hsl(0,0%,30%); background: hsl(0,0%,99%);">
	<main style="display: block; position: relative; margin: 15px auto; padding: 5px 15px 15px 15px; max-width: 420px; background: #FFF; box-shadow: 0 2px 8px hsla(0,0%,0%,.12); border-radius: 2.5px;">
		<h3>Hi there ` + user.Username + `!</h3>
		Please follow the verification link to login to ` + AppName + `.
		<br>
		<a href="` + magicLink + `" style="display: block; font-size: 1.2em; font-weight: 600; margin: 10px auto; max-width: 180px; padding: 8px; border-radius: 2.5px; text-decoration: none; color: #fff; background: hsl(0,0%,30%); box-shadow: 0 2px 6px hsla(0,0%,0%,.12); text-shadow: 0 1px 3px hsla(0,0%,0%,.12);">
			Login
		</a>
		<br>
		<footer>
			Note, this link will expire in about 15 minutes, just log in again from <a href="https://saul.app" style="color: inherit;">saul.app</a> if it doesn't work.
		</footer>
	</main>
</body>
</html>`),
	})

	return err
}

func verifyUser(user *User, verifier string) error {
	if user.Verifier != verifier {
		return ErrUnauthorized
	}
	err := user.Update(obj{
		"$unset": obj{"verifier": verifier},
		"$addToSet": obj{
			"roles":  VerifiedUser,
			"logins": time.Now(),
		},
	})

	for _, val := range user.Roles {
		if val == UnverifiedUser {
			err = user.Update(obj{
				"$pull": obj{"roles": UnverifiedUser},
			})
		}
	}

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

	fmt.Println("Auth Handling Started")
}
