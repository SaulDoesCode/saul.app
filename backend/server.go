package backend

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/SaulDoesCode/branca"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
	"golang.org/x/crypto/acme/autocert"
)

type obj = map[string]interface{}
type ctx = echo.Context

var (
	// AppName name of this application
	AppName string
	// Config file data as gjson result
	Config gjson.Result
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
	conf, err := ReadJSONFile(configfile)
	critCheck(err)
	Config = conf

	DevMode = os.Getenv("SAULAPP_DEVMODE") == "true"

	Server = echo.New()

	Server.Use(middleware.Static(Config.Get("assets").String()))
	Server.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 9}))
	Server.Use(middleware.Recover())
	Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
	}))

	AppName = Config.Get("appname").String()

	fmt.Println("Firing up: ", AppName+"...")
	fmt.Println("DevMode: ", DevMode)

	insecurePort := ":"
	if DevMode {
		insecurePort += Config.Get("devInsecurePort").String()
	} else {
		insecurePort += Config.Get("insecurePort").String()
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

	addrs := []string{}
	for _, val := range Config.Get("db_address").Array() {
		addrs = append(addrs, val.String())
	}

	DB.Open(&DialInfo{
		Addrs:   addrs,
		AppName: AppName,
		Timeout: 60 * time.Second,
	}, Config.Get("db_name").String())
	defer DB.Close()

	EmailConf.Email = Config.Get("admin-email.email").String()
	EmailConf.Server = Config.Get("admin-email.server").String()
	EmailConf.Port = Config.Get("admin-email.port").String()
	EmailConf.Password = Config.Get("admin-email.password").String()
	EmailConf.FromTxt = Config.Get("admin-email.fromtxt").String()
	EmailConf.Address = EmailConf.Server + ":" + EmailConf.Port

	startEmailer()
	defer stopEmailer()

	Branca = branca.NewBranca(Config.Get("token_secret").String())
	Branca.SetTTL(900)

	initAuth()

	if DevMode {
		Server.Logger.Fatal(Server.StartTLS(
			":"+Config.Get("devPort").String(),
			Config.Get("https_cert").String(),
			Config.Get("https_key").String(),
		))

		Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
			Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
		}))
	} else {
		Server.AutoTLSManager.HostPolicy = autocert.HostWhitelist(Config.Get("domain").String())
		Server.AutoTLSManager.Cache = autocert.DirCache(Config.Get("privates").String())
		Server.Logger.Fatal(Server.StartAutoTLS(":" + Config.Get("port").String()))
	}
}
