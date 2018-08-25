package main

import (
	"net/http"
	"github.com/sauldoescode/saul.app/backend"
)

func main() {
	backend.Init("./private/config.json")
}