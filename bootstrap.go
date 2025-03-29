package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/go-oauth2/mysql/v4"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-oauth2/oauth2/v4/errors"
	"github.com/go-oauth2/oauth2/v4/manage"
	"github.com/go-oauth2/oauth2/v4/models"
	"github.com/go-oauth2/oauth2/v4/server"
	"github.com/go-oauth2/oauth2/v4/store"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
	"gopkg.in/gomail.v2"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var database *gorm.DB
var databaseError error
var oauthServer *server.Server
var serviceClient OauthClient
var eDialer *gomail.Dialer
var location *time.Location

func loadDatabase() {
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

func setupOauth() {
	manager := manage.NewDefaultManager()

	clientStore := store.NewClientStore()
	tokenStore := mysql.NewDefaultStore(
		mysql.NewConfig(
			fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
				os.Getenv("DB_USER"),
				os.Getenv("DB_PASS"),
				os.Getenv("DB_HOST"),
				os.Getenv("DB_PORT"),
				os.Getenv("DB_NAME"))),
	)

	var clients []OauthClient

	domain := os.Getenv("HTTP_HOST")

	clientError := database.
		Model(OauthClient{}).
		Find(&clients).Error

	if len(clients) < 1 {
		log.Fatalln("You must register at least one oauth client", domain)
		return
	}

	serviceClient = clients[0]

	if clientError != nil {
		log.Fatalln(clientError)
		return
	}

	for _, client := range clients {
		log.Println("Registered oauth client " + strconv.FormatInt(client.ID, 10))

		clientId := strconv.FormatInt(client.ID, 10)

		clientStore.Set(clientId, &models.Client{
			ID:     clientId,
			Secret: client.Secret,
			Domain: client.Domain,
		})
	}

	manager.MapClientStorage(clientStore)
	manager.MapTokenStorage(tokenStore)

	oauthServer = server.NewDefaultServer(manager)
	oauthServer.SetAllowGetAccessRequest(true)
	oauthServer.SetClientInfoHandler(server.ClientFormHandler)

	oauthServer.SetResponseErrorHandler(func(re *errors.Response) {
		log.Println("Response Error:", re.Error.Error())
	})

	oauthServer.SetPasswordAuthorizationHandler(func(ctx context.Context, clientID, username, password string) (userID string, err error) {

		return "", errors.New("no account exists with that email")
	})

	oauthServer.SetAllowedGrantType(oauth2.PasswordCredentials)
}

func setupMail() {
	eDialer = gomail.NewDialer(
		os.Getenv("MAIL_HOST"),
		getEnvAsInt("MAIL_PORT", 587),
		os.Getenv("MAIL_USERNAME"),
		os.Getenv("MAIL_PASSWORD"))

	eDialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}
}

func authorizeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" || !strings.HasPrefix(authorizationHeader, "Bearer ") {
			w.WriteHeader(401)
			return
		}

		tokenInfo, error := oauthServer.ValidationBearerToken(r)

		if error != nil {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: error.Error()})
			return
		}

		if tokenInfo == nil {
			w.WriteHeader(401)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: "UNKNOWN_ERROR_AUTH_3"})
			return
		}

		ctx := context.WithValue(r.Context(), "tokenInfo", tokenInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func serveHttp() {
	log.Fatal(http.ListenAndServe("0.0.0.0:"+os.Getenv("HTTP_PORT"), corsHandler(router)))
}

func corsHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		w.Header().Set("Access-Control-Allow-Credentials", "false")
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))

		if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.Header().Set("Vary", "Origin")
			h.ServeHTTP(w, r)
		}
	})
}
