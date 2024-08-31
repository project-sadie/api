package main

import (
	"github.com/joho/godotenv"
	"log"
	"os"
	"time"
)

import _ "github.com/go-sql-driver/mysql"

func main() {
	dotenvError := godotenv.Load()

	if dotenvError != nil {
		log.Fatal(dotenvError)
	}

	location, _ = time.LoadLocation(os.Getenv("TIMEZONE"))

	loadDatabase()
	setupOauth()
	setupMail()
	registerRoutes()
	serveHttp()
}
