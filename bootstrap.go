package sadie_api

import (
	"fmt"
	"github.com/jinzhu/gorm"
	"log"
	"os"
)

var database *gorm.DB
var databaseError error

func testDatabaseConnection() {
	var connectionString = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASS"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"))

	database, databaseError = gorm.Open("mysql", connectionString)

	if databaseError != nil {
		log.Fatalln(databaseError)
	} else {
		log.Println("We connected to the database.")
		database.LogMode(true)
	}
}
