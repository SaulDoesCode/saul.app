package backend

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/asaskevich/govalidator"
	"github.com/microcosm-cc/bluemonday"
	"github.com/tidwall/gjson"
	"gopkg.in/russross/blackfriday.v2"
)

var (
	// RandomDictionary the character range of the randomBytes and randomString functions
	RandomDictionary = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

func validUsername(username string) bool {
	return govalidator.Matches(username, `^[a-zA-Z0-9._-]{3,50}$`)
}

func validEmail(email string) bool {
	return govalidator.IsEmail(email)
}

func validUsernameAndEmail(username string, email string) bool {
	return validEmail(email) && validUsername(username)
}

func check(err error) error {
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func critCheck(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// RandBytes generate a random a []byte of a specific size
func RandBytes(size int) []byte {
	bits := make([]byte, size)
	rand.Read(bits)
	for k, v := range bits {
		bits[k] = RandomDictionary[v%byte(len(RandomDictionary))]
	}
	return bits
}

// RandStr generate a random string of a specific size
func RandStr(size int) string {
	return string(RandBytes(size))
}

// GetMD5Hash turn []byte into a MD5 hashed string
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

// MD5Hash turn []byte into a MD5 hashed string
func MD5Hash(data []byte) string {
	hasher := md5.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func renderMarkdown(input []byte) []byte {
	return bluemonday.UGCPolicy().SanitizeBytes(blackfriday.Run(input))
}

func sendError(errorStr string) func(c ctx) error {
	return func(c ctx) error {
		return JSONErr(c, 400, errorStr)
	}
}

// JSONErr helper to send a simple {"err": msg} json with an arbitrary code
func JSONErr(c ctx, code int, err string) error {
	return c.JSONBlob(code, []byte(`{"err":"`+err+`"}`))
}

// ReadJSONFile read a json file and get a gjson result for easy use
func ReadJSONFile(location string) (gjson.Result, error) {
	var result gjson.Result
	data, err := ioutil.ReadFile(location)
	if err != nil {
		return result, err
	}
	result = gjson.ParseBytes(data)
	return result, nil
}

// JSONbody get echo.Context's body as a gjson.Result
func JSONbody(c ctx) (gjson.Result, error) {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return gjson.Result{}, err
	}
	return gjson.ParseBytes(body), err
}

// UnmarshalJSONFile read json files and go straight to unmarshalling
func UnmarshalJSONFile(location string, marshaled interface{}) error {
	data, err := ioutil.ReadFile(location)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, marshaled)
}

// Int64ToString convert int64 to strings (for ports and stuff when you want to make json less stringy)
func Int64ToString(n int64) string {
	return strconv.FormatInt(n, 10)
}
