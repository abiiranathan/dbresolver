# dbresolver

<div style="display:flex; gap:8px">

[![GoDoc](https://pkg.go.dev/badge/github.com/abiiranathan/dbresolver)](https://pkg.go.dev/github.com/abiiranathan/dbresolver)

[![Go Report Card](https://goreportcard.com/badge/github.com/abiiranathan/repo-name)](https://goreportcard.com/report/github.com/abiiranathan/dbresolver)

</div>

dbresolver provides functionality for resolving and managing multiple database connections based on API keys from different clients.

If your have a small number of customers with a desire to use the same API but with different databases, this might be for you.

### Supported databases

- postgres
- mysql
- sqlite3

Usage:

```go

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/abiiranathan/dbresolver/dbresolver"
)

type Todo struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Completed bool   `json:"completed"`
}

func main() {
	// Customize name of header or query containing unique key.
	dbresolver.SetHeaderName("apikey")

	// Parse the connection information from yaml.
	// Can as well use dbresolver.ConfigFromYAMLString.
	databaseConfig, err := dbresolver.ConfigFromYAMLFile("dbresolver.yaml")
	if err != nil {
		panic(err)
	}

	// Create an instance of the resolver.
	resolver, err := dbresolver.New(databaseConfig)
	if err != nil {
		panic(err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get the database name from the request
		db := resolver.DB(r)

		// Print database name
		// fmt.Println(resolver.DBName(r))

		todos := []Todo{}
		db.Find(&todos)
		b, _ := json.Marshal(todos)
		w.Write(b)
	})

	resolver.AutoMigrate([]any{&Todo{}}, func(err error) bool {
		// Errors if a table already exists can be skipped.
		fmt.Println(err)
		return true
	})

	// Hook your main handler with our resolver middleware.
	mux := resolver.Middleware()(http.DefaultServeMux)
	log.Fatalln(http.ListenAndServe(":8080", mux))
}
```

### Structure of dbresolver.yaml

```yaml
apikey-1:
  database: ristal.sqlite3
  driver: sqlite
apikey-2:
  database: imara.sqlite3
  driver: sqlite
apikey-3:
  database: dbname=dbname user=user host=localhost password=STRONG_PASSWORD sslmode=disabled
```

**When you perform the request, you specify the API key in the request headers or as a query parameter.**

```console
curl http://localhost:8080?apikey=apikey-3 | jq
```

```js
fetch("http://localhost:8080", {
  headers: {
    apikey: "Secret API Key",
  },
});
```
