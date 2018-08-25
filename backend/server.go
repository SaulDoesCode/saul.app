package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"golang.org/x/crypto/acme/autocert"
)

type obj = map[string]interface{}

var (
	// Server the echo instance running the show
	Server *echo.Echo
	// Config file data as gjson result
	Config obj
	// DevMode run the app in production or dev-mode
	DevMode = false
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

	fmt.Println("Firing up: " + Config["appname"].(string) + "...")

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

	if DevMode {
		Server.Logger.Fatal(Server.StartTLS(":"+Config["port"].(string), Config["https_cert"].(string), Config["https_key"].(string)))
	} else {
		Server.AutoTLSManager.HostPolicy = autocert.HostWhitelist(Config["domain"].(string))
		Server.AutoTLSManager.Cache = autocert.DirCache(Config["privates"].(string))
		Server.Logger.Fatal(Server.StartAutoTLS(":" + Config["port"].(string)))
	}
}
