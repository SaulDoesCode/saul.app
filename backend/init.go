package backend

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"text/template"
	// "os"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
	"github.com/integrii/flaggy"
)

type obj = map[string]interface{}
type ctx = echo.Context

const oneweek = 7 * 24 * time.Hour

var (
	// AuthEmailHTML html template for authentication emails
	AuthEmailHTML *template.Template
	// AuthEmailTXT html template for authentication emails
	AuthEmailTXT *template.Template
	// PostTemplate html template for post pages
	PostTemplate *template.Template
	// AppName name of this application
	AppName string
	// AppDomain web domain of this application
	AppDomain string
	// Config file data as gjson result
	Config gjson.Result
	// DKIMKey private dkim key used for email signing
	DKIMKey []byte
	// Server the echo instance running the show
	Server *echo.Echo
	// DevMode run the app in production or dev-mode
	DevMode = false
	// Tokenator token generator/decoder
	Tokenator *Branca
	// Verinator token generator/decoder for verification codes only
	Verinator *Branca
	// RateLimiter restrict spammy trafic with a tollbooth limiter
	RateLimiter *limiter.Limiter
	// MaintainerEmails the list of people to email if all hell breaks loose
	MaintainerEmails []string
	insecurePort string
	// AssetsFolder path to all the servable static assets
	AssetsFolder string
)

// Init start the backend server
func Init(configfile string) {
	conf, err := ReadJSONFile(configfile)
	critCheck(err)
	Config = conf

	// os.Getenv("SAULAPP_DEVMODE") == "true" || 
	if Config.Get("devmode").Bool() {
		DevMode = true
	} else {
		flaggy.Bool(&DevMode, "dev", "devmode", "putt the server into dev mode for extra logging and checks")
	}

	flaggy.Parse()

	Server = echo.New()

	Server.Use(middleware.Recover())
	Server.Use(middleware.BodyLimit("3M"))

	Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
	}))

	RateLimiter = tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	RateLimiter.SetMethods([]string{"GET", "POST"})

	Server.Use(LimitMiddleware(RateLimiter))

	AssetsFolder = Config.Get("assets").String()

	Server.Static("/", AssetsFolder)

	AppName = Config.Get("appname").String()
	AppDomain = Config.Get("domain").String()

	maintainerEmails := Config.Get("maintainer_emails").Array()
	if len(maintainerEmails) >= 1 {
		for _, me := range maintainerEmails {
			MaintainerEmails = append(MaintainerEmails, me.String())
		}
	}

	DKIMKey, err = ioutil.ReadFile(Config.Get("dkim_key").String())
	if err != nil {
		fmt.Println("dkim_key is missing, generate one")
		panic(err)
	}

	fmt.Println("Firing up: ", AppName+"...")
	fmt.Println("DevMode: ", DevMode)

	EmailConf.Email = Config.Get("admin_email.email").String()
	EmailConf.Server = Config.Get("admin_email.server").String()
	EmailConf.Port = Config.Get("admin_email.port").String()
	EmailConf.Password = Config.Get("admin_email.password").String()
	EmailConf.FromName = Config.Get("admin_email.name").String()
	EmailConf.Address = EmailConf.Server + ":" + EmailConf.Port

	fmt.Println(EmailConf.Address, EmailConf.Email, EmailConf.FromName)
	startEmailer()

	insecurePort = ":"
	if DevMode {
		insecurePort += Config.Get("devInsecurePort").String()
	} else {
		insecurePort += Config.Get("insecurePort").String()
	}

	addrs := []string{}
	rawlocaladdrs := Config.Get("db_local_address").Array()
	if len(rawlocaladdrs) >= 1 {
		for _, val := range rawlocaladdrs {
			addrs = append(addrs, val.String())
		}
	}

	err = setupDB(
		addrs,
		Config.Get("db_name").String(),
		Config.Get("db_username").String(),
		Config.Get("db_password").String(),
	)
	if err != nil {
		fmt.Println("couldn't connect to DB locally, trying remote connection now...")

		addrs = []string{}
		rawaddrs := Config.Get("db_address").Array()
		for _, val := range rawaddrs {
			addrs = append(addrs, val.String())
		}

		err = setupDB(
			addrs,
			Config.Get("db_name").String(),
			Config.Get("db_username").String(),
			Config.Get("db_password").String(),
		)
		if err != nil {
			fmt.Println("couldn't get DB connection going: ", err)
			panic(err)
		}
	}

	startDBHealthCheck()
	defer DBHealthTicker.Stop()

	AuthEmailHTML = template.Must(template.ParseFiles("./templates/authemail.html"))
	AuthEmailTXT = template.Must(template.ParseFiles("./templates/authemail.txt"))
	PostTemplate = template.Must(template.ParseFiles("./templates/post.html"))

	Tokenator = NewBranca(Config.Get("token_secret").String())
	Tokenator.SetTTL(86400 * 7)
	Verinator = NewBranca(Config.Get("verifier_secret").String())
	Verinator.SetTTL(925)

	initAuth()
	initWrits()

	startHTTPServer()

	if DevMode {
		err = Server.StartTLS(
			":"+Config.Get("devPort").String(),
			Config.Get("https_cert").String(),
			Config.Get("https_key").String(),
		)
	} else {
		err = Server.StartTLS(
			":"+Config.Get("port").String(),
			"/etc/letsencrypt/live/"+AppDomain+"/cert.pem",
			"/etc/letsencrypt/live/"+AppDomain+"/privkey.pem",
		)
	}

	if err != nil {
		fmt.Println("unable to start app server, something must be misconfigured: ", err)
	}

	time.Sleep(5 * time.Second)
}

// LimitMiddleware tollbooth adapter for echo
//
// from  https://github.com/didip/tollbooth_echo/blob/master/tollbooth_echo.go
// credit goes to @didip
func LimitMiddleware(lmt *limiter.Limiter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return echo.HandlerFunc(func(c echo.Context) error {
			httpError := tollbooth.LimitByRequest(lmt, c.Response(), c.Request())
			if httpError != nil {
				return c.String(httpError.StatusCode, httpError.Message)
			}
			return next(c)
		})
	}
}

var redirectServer *http.Server

func startHTTPServer() {
    redirectServer = &http.Server{Addr: insecurePort}

		redirectServer.Handler = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			target := "https://" + req.Host + req.URL.Path
			if len(req.URL.RawQuery) > 0 {
				target += "?" + req.URL.RawQuery
			}
			if DevMode {
				fmt.Printf("\nredirect to: %s \n", target)
				fmt.Println(req.RemoteAddr)
			}
			http.Redirect(res, req, target, http.StatusTemporaryRedirect)
		})

		go redirectServer.ListenAndServe()
		fmt.Println("insecure to secure redirect server started")
}