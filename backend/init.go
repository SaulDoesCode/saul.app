package backend

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/SaulDoesCode/echo-memfile"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/tidwall/gjson"
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
)

// Init start the backend server
func Init(configfile string) {
	conf, err := ReadJSONFile(configfile)
	critCheck(err)
	Config = conf

	DevMode = os.Getenv("SAULAPP_DEVMODE") == "true"

	Server = echo.New()

	Server.Use(middleware.Recover())
	Server.Use(middleware.BodyLimit("3M"))

	Server.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: "${method}::${status} ${host}${uri}  \tlag=${latency_human}\n",
	}))

	RateLimiter = tollbooth.NewLimiter(1, &limiter.ExpirableOptions{DefaultExpirationTTL: time.Hour})

	RateLimiter.SetMethods([]string{"GET", "POST"})

	Server.Use(LimitMiddleware(RateLimiter))

	mfi := memfile.New(Server, Config.Get("assets").String(), false)
	if DevMode {
		mfi.UpdateOnInterval(time.Millisecond * 400)
	} else {
		mfi.UpdateOnInterval(time.Second * 5)
	}
	MFI = &mfi

	AppName = Config.Get("appname").String()
	AppDomain = Config.Get("domain").String()

	DKIMKey, err = ioutil.ReadFile(Config.Get("dkim_key").String())
	if err != nil {
		fmt.Println("dkim_key is missing, generate one")
		panic(err)
	}

	fmt.Println("Firing up: ", AppName+"...")
	fmt.Println("DevMode: ", DevMode)

	insecurePort := ":"
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
	if err != nil && err == ErrBadDBConnection {
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

	AuthEmailHTML = template.Must(template.ParseFiles("./templates/authemail.html"))
	AuthEmailTXT = template.Must(template.ParseFiles("./templates/authemail.txt"))
	PostTemplate = template.Must(template.ParseFiles("./templates/post.html"))

	EmailConf.Email = Config.Get("admin_email.email").String()
	EmailConf.Server = Config.Get("admin_email.server").String()
	EmailConf.Port = Config.Get("admin_email.port").String()
	EmailConf.Password = Config.Get("admin_email.password").String()
	EmailConf.FromName = Config.Get("admin_email.name").String()
	EmailConf.Address = EmailConf.Server + ":" + EmailConf.Port

	fmt.Println(EmailConf.Address, EmailConf.Email, EmailConf.FromName)

	startEmailer()
	defer stopEmailer()

	Tokenator = NewBranca(Config.Get("token_secret").String())
	Tokenator.SetTTL(86400 * 7)
	Verinator = NewBranca(Config.Get("verifier_secret").String())
	Verinator.SetTTL(925)

	initAuth()
	initWrits()

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

	if DevMode {
		Server.Logger.Fatal(Server.StartTLS(
			":"+Config.Get("devPort").String(),
			Config.Get("https_cert").String(),
			Config.Get("https_key").String(),
		))
	} else {
		Server.Logger.Fatal(Server.StartTLS(
			":"+Config.Get("port").String(),
			"/etc/letsencrypt/live/"+AppDomain+"/cert.pem",
			"/etc/letsencrypt/live/"+AppDomain+"/privkey.pem",
		))
	}
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