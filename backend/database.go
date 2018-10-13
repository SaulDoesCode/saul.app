package backend

import (
	"context"
	"fmt"
	"log"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
)

var (
	// DB central arangodb database for querying
	DB driver.Database
	// Users arangodb collection containing user data
	Users driver.Collection
)

func setupDB(endpoints []string, dbname, username, password string) {
	fmt.Println(`Attempting ArangoDB connection...
		DB: ` + dbname + `
	`)

	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.JWTAuthentication(username, password),
	})
	if err != nil {
		fmt.Println("Could not get proper arangodb client:")
		panic(err)
	}

	db, err := client.Database(nil, dbname)
	if err != nil {
		fmt.Println("Could not get database object:")
		panic(err)
	}

	DB = db
	users, err := DB.Collection(nil, "users")
	Users = users
	if err != nil {
		fmt.Println("Could not get users collection from db:")
		panic(err)
	}

	fmt.Println(`ArangoDB Connected. So far so good.`)
}

// Query query the app's DB with AQL, bindvars, and map that to an output
func Query(query string, vars obj) ([]obj, error) {
	var objects []obj
	ctx := driver.WithQueryCount(context.Background())
	cursor, err := DB.Query(ctx, query, vars)
	if err == nil {
		defer cursor.Close()
		objects = []obj{}
		for {
			var doc obj
			_, err := cursor.ReadDocument(ctx, &doc)
			if driver.IsNoMoreDocuments(err) || err != nil {
				break
			}
			objects = append(objects, doc)
		}
	}
	return objects, err
}

// QueryOne query the app's DB with AQL, bindvars, and map that to an output
func QueryOne(query string, vars obj, result interface{}) error {
	ctx := driver.WithQueryCount(context.Background())
	cursor, err := DB.Query(ctx, query, vars)
	if err == nil {
		_, err = cursor.ReadDocument(ctx, result)
		cursor.Close()
	}
	return err
}
