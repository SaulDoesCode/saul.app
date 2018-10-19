package backend

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/labstack/echo"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
	"github.com/tidwall/gjson"
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

var (
	bmPolicy = bluemonday.UGCPolicy()
)

func renderMarkdown(input []byte, sanitize bool) []byte {
	if sanitize {
		return bmPolicy.SanitizeBytes(blackfriday.Run(input))
	}
	return blackfriday.Run(input)
}

func sendError(errorStr string) func(c ctx) error {
	return func(c ctx) error {
		return JSONErr(c, 400, errorStr)
	}
}

// JSONErr helper to send a simple {"err": msg} json with an arbitrary code
func JSONErr(c ctx, code int, err string) error {
	return c.JSON(code, obj{"err": err, "ok": false})
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
	var res gjson.Result
	reqbody := c.Request().Body
	if reqbody == nil {
		return res, echo.ErrUnsupportedMediaType
	}
	body, err := ioutil.ReadAll(reqbody)
	if err != nil {
		return res, err
	}
	res = gjson.ParseBytes(body)
	return res, err
}

// UnmarshalJSONBody unmarshal json data straight to struct and such
func UnmarshalJSONBody(c ctx, result interface{}) error {
	reqbody := c.Request().Body
	if reqbody == nil {
		return echo.ErrUnsupportedMediaType
	}
	return json.NewDecoder(reqbody).Decode(result)
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

func execTemplate(temp *template.Template, vars interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := temp.Execute(&buf, vars)
	return buf.Bytes(), err
}

func unix2time(unix string) (time.Time, error) {
	var tm time.Time
	i, err := strconv.ParseInt(unix, 10, 64)
	if err == nil {
		tm = time.Unix(i, 0)
	}
	return tm, err
}
