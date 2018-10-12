package backend

import (
	"fmt"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/SaulDoesCode/echo-memfile"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
	//	"golang.org/x/crypto/acme/autocert"
)

type obj = map[string]interface{}
type ctx = echo.Context

var (
	// MFI instance of memfile static file memory caching
	MFI *memfile.MemFileInstance
	// AuthEmailHTML html template for authentication emails
	AuthEmailHTML *template.Template
	// AuthEmailTXT html template for authentication emails
	AuthEmailTXT *template.Template
	// AppName name of this application
	AppName string
	// Config file data as gjson result
	Config gjson.Result
	// Server the echo instance running the show
	Server *echo.Echo
	// DevMode run the app in production or dev-mode
	DevMode = false
	// Tokenator token generator/decoder
	Tokenator *Branca
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

	Server.Use(middleware.Recover())
	Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
	}))

	MFI := memfile.New(Server, Config.Get("assets").String(), DevMode)

	if DevMode {
		MFI.UpdateOnInterval(time.Millisecond * 400)
	} else {
		MFI.UpdateOnInterval(time.Second * 5)
	}

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

	setupDB(
		addrs,
		Config.Get("db_name").String(),
		Config.Get("db_username").String(),
		Config.Get("db_password").String(),
	)

	AuthEmailHTML = template.Must(template.ParseFiles("./templates/authemail.html"))
	AuthEmailTXT = template.Must(template.ParseFiles("./templates/authemail.txt"))

	EmailConf.Email = Config.Get("admin_email.email").String()
	EmailConf.Server = Config.Get("admin_email.server").String()
	EmailConf.Port = Config.Get("admin_email.port").String()
	EmailConf.Password = Config.Get("admin_email.password").String()
	EmailConf.FromTxt = Config.Get("admin_email.fromtxt").String()
	EmailConf.Address = EmailConf.Server + ":" + EmailConf.Port

	fmt.Println(EmailConf.Address, EmailConf.FromTxt)
	startEmailer()
	defer stopEmailer()

	Tokenator = NewBranca(Config.Get("token_secret").String())
	Tokenator.SetTTL(900)

	initAuth()

	if DevMode {
		Server.Logger.Fatal(Server.StartTLS(
			":"+Config.Get("devPort").String(),
			Config.Get("https_cert").String(),
			Config.Get("https_key").String(),
		))
	} else {
		//Server.AutoTLSManager.HostPolicy = autocert.HostWhitelist(Config.Get("domain").String())
		//Server.AutoTLSManager.Cache = autocert.DirCache(Config.Get("privates").String())
		//Server.Logger.Fatal(Server.StartAutoTLS(":" + Config.Get("port").String())
		Server.Logger.Fatal(Server.StartTLS(
			":"+Config.Get("port").String(),
			"/etc/letsencrypt/live/"+Config.Get("domain").String()+"/cert.pem",
			"/etc/letsencrypt/live/"+Config.Get("domain").String()+"/privkey.pem",
		))
	}
}
