package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/SaulDoesCode/branca"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type obj = map[string]interface{}
type ctx = echo.Context

var (
	// AppName name of this application
	AppName string
	// Config file data as gjson result
	Config obj
	// Server the echo instance running the show
	Server *echo.Echo
	// DevMode run the app in production or dev-mode
	DevMode = false
	// DB mongodb wrapper struct
	DB = &Database{}
	// Branca token generator/decoder
	Branca *branca.Branca
	// VerifierSize size of pre-token verification code
	VerifierSize = 14
)

// Init start the backend server
func Init(configfile string) {
	raw, err := ioutil.ReadFile(configfile)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(raw, &Config); err != nil {
		panic(err)
	}

	DevMode = os.Getenv("SAULAPP_DEVMODE") == "true"

	Server = echo.New()

	Server.Use(middleware.Static(Config["assets"].(string)))
	Server.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9}))
	Server.Use(middleware.Recover())
	Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
	}))

	AppName = Config["appname"].(string)

	fmt.Println("Firing up: ", AppName+"...")
	fmt.Println("DevMode: ", DevMode)

	insecurePort := ":"
	if DevMode {
		insecurePort += Config["devInsecurePort"].(string)
	} else {
		insecurePort += Config["insecurePort"].(string)
	}

	go http.ListenAndServe(insecurePort, http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		target := "https://" + req.Host + req.URL.Path
		if len(req.URL.RawQuery) > 0 {
			target += "?" + req.URL.RawQuery
		}
		if DevMode {
			fmt.Printf("\nredirect to: %s \n", target)
			fmt.Println(req.RemoteAddr)
		}
		http.Redirect(res, req, target, http.StatusTemporaryRedirect)
	}))

	DB.Open(&DialInfo{
		Addrs:   []string{Config["db_address"].(string)},
		AppName: Config["appname"].(string),
		Timeout: 60 * time.Second,
	}, Config["db_name"].(string))
	defer DB.Close()

	EmailConf.Email = Config["admin_email.email"].(string)
	EmailConf.Server = Config["admin_email.server"].(string)
	EmailConf.Port = Config["admin_email.port"].(string)
	EmailConf.Password = Config["admin_email.password"].(string)
	EmailConf.FromTxt = Config["admin_email.fromtxt"].(string)
	EmailConf.Address = EmailConf.Server + ":" + EmailConf.Port

	startEmailer()
	defer stopEmailer()

	Branca = branca.NewBranca(Config["token_secret"].(string))
	Branca.SetTTL(900)

	if DevMode {
		Server.Logger.Fatal(Server.StartTLS(":"+Config["devPort"].(string), Config["https_cert"].(string), Config["https_key"].(string)))
	} else {
		Server.AutoTLSManager.HostPolicy = autocert.HostWhitelist(Config["domain"].(string))
		Server.AutoTLSManager.Cache = autocert.DirCache(Config["privates"].(string))
		Server.Logger.Fatal(Server.StartAutoTLS(":" + Config["port"].(string)))
	}
}
