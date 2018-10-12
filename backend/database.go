package backend

import (
	"context"
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
	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatalf("Failed to create HTTP connection: %v", err)
	}

	client, err := driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication(username, password),
	})
	if err != nil {
		log.Fatalf("Could not get proper arangodb client: %v", err)
	}

	ctx := context.Background()

	db, err := client.Database(ctx, dbname)
	if err != nil {
		log.Fatalf("Could not get database object: %v", err)
	}
	DB = db
	users, err := DB.Collection(ctx, "users")
	Users = users
	if err != nil {
		log.Fatalf("Could not get users collection from db: %v", err)
	}
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
	defer cursor.Close()
	if err == nil {
		_, err = cursor.ReadDocument(ctx, &result)
	}
	return err
}
