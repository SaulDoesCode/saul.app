package backend

import (
	"context"
	"crypto/tls"
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
	// Writs arangodb writ collection containing writs
	Writs driver.Collection
	// RateLimits arangodb ratelimits collection
	RateLimits driver.Collection
)

func setupDB(endpoints []string, dbname, username, password string) {
	fmt.Println(`Attempting ArangoDB connection...
		DB: ` + dbname + `
	`)

	// Create an HTTP connection to the database
	conn, err := http.NewConnection(http.ConnectionConfig{
		Endpoints: endpoints,
		TLSConfig: &tls.Config{
			ServerName:         AppDomain,
			InsecureSkipVerify: true,
		},
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
	if err != nil {
		fmt.Println("Could not get users collection from db:")
		panic(err)
	}
	Users = users

	writs, err := DB.Collection(nil, "writs")
	if err != nil {
		fmt.Println("Could not get users collection from db:")
		panic(err)
	}
	Writs = writs

	_, _, err = Users.EnsureHashIndex(
		nil,
		[]string{"username", "email", "emailmd5"},
		&driver.EnsureHashIndexOptions{Unique: true},
	)
	if err != nil {
		panic(err)
	}

	_, _, err = Users.EnsureHashIndex(
		nil,
		[]string{"verifier"},
		&driver.EnsureHashIndexOptions{Unique: true, Sparse: true},
	)
	if err != nil {
		panic(err)
	}

	_, _, err = Writs.EnsureHashIndex(
		nil,
		[]string{"title", "tags", "slug"},
		&driver.EnsureHashIndexOptions{Unique: true},
	)
	if err != nil {
		panic(err)
	}

	ratelimits, err := DB.Collection(nil, "ratelimits")
	if err != nil {
		fmt.Println("Could not get ratelimiting collection from db:")
		panic(err)
	}
	RateLimits = ratelimits
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
			_, err = cursor.ReadDocument(ctx, &doc)
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
