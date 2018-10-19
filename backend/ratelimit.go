package backend

import (
	"context"
	"fmt"
	"time"

	"github.com/arangodb/go-driver"
)

type ratelimit struct {
	Key   string `json:"_key,omitempty"`
	Start int64  `json:"start"`
	Count int64  `json:"count"`
}

func ratelimitEmail(email string, maxcount int64, duration time.Duration) bool {
	var limit ratelimit
	err := QueryOne(
		`FOR l IN ratelimits
		 FILTER l._key == @key
		 UPDATE l WITH {count: l.count + 1} IN ratelimits OPTIONS {waitForSync: true}
		 RETURN NEW`,
		obj{"key": email},
		&limit,
	)
	if driver.IsNotFound(err) || driver.IsNoMoreDocuments(err) {
		_, err := DB.Query(
			driver.WithWaitForSync(context.Background()),
			`INSERT {_key: @key, start: @start, count: 1} IN ratelimits`,
			obj{"start": time.Now().Unix(), "key": email},
		)
		if err != nil && DevMode {
			fmt.Println("email ratelimits error: new limit ", err)
		}
		return err == nil
	} else if err != nil {
		if DevMode {
			fmt.Println("email ratelimits error: something happened ", err)
		}
		return false
	}

	if limit.Start+int64(duration) < time.Now().Unix() {
		// _, err := RateLimits.RemoveDocument(driver.WithWaitForSync(context.Background()), email)
		_, err := DB.Query(
			driver.WithWaitForSync(context.Background()),
			`UPDATE @key WITH {count: 0, start: @start} IN ratelimits`,
			obj{"start": time.Now().Unix(), "key": email},
		)
		if DevMode && err != nil {
			fmt.Println("email ratelimits error: trouble resetting ", err)
		}
		return err == nil
	}

	if limit.Count > maxcount {
		_, err := DB.Query(
			driver.WithWaitForSync(context.Background()),
			`FOR l IN ratelimits FILTER l._key == @key
		   UPDATE l WITH {count: l.count + 1, start: @start} IN ratelimits`,
			obj{"start": time.Now().Add(5 * time.Minute).Unix(), "key": email},
		)
		if DevMode && err != nil {
			fmt.Println("email ratelimits error: trouble removing entry ", err)
		}
		return false
	}

	return true
}
