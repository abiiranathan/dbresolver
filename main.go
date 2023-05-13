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
	mux := resolver.Middleware(http.DefaultServeMux)
	log.Fatalln(http.ListenAndServe(":8080", mux))
}
