package main

import (
	"github.com/joho/godotenv"
	"log"
)

import _ "github.com/go-sql-driver/mysql"

func main() {
	dotenvError := godotenv.Load()

	if dotenvError != nil {
		log.Fatal(dotenvError)
	}

	loadDatabase()
	setupOauth()
	registerRoutes()
	serveHttp()
}
