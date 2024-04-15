package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/bcrypt"
	"log"
	"net/http"
	"strconv"
	"time"
)

func TokenRequestHandler(w http.ResponseWriter, r *http.Request) {
	err := oauthServer.HandleTokenRequest(w, r)
	fmt.Println(err)
}

func PlayerLoginHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	var credentials Credentials

	_ = decoder.Decode(&credentials)

	var player Player

	var queryError = database.Model(Player{}).
		Where("username = ?", credentials.Username).
		First(&player).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusBadRequest)

		response := map[string]string{
			"error_message": "Couldn't find a record with this username",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	fmt.Println(credentials.Password)
	fmt.Println(player.Password)

	if bcrypt.CompareHashAndPassword([]byte(player.Password), []byte(credentials.Password)) != nil {
		w.WriteHeader(http.StatusBadRequest)

		response := map[string]string{
			"error_message": "Incorrect password, please try again",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	tokenInfo, tokenError := oauthServer.Manager.GenerateAccessToken(r.Context(), oauth2.PasswordCredentials, &oauth2.TokenGenerateRequest{
		ClientID:     strconv.FormatInt(serviceClient.ID, 10),
		ClientSecret: serviceClient.Secret,
		Request:      r,
		Scope:        "read",
		UserID:       credentials.Username,
	})

	if tokenError != nil {
		log.Fatalln(tokenError)
	}

	response := map[string]interface{}{
		"access_token": tokenInfo.GetAccess(),
		"token_type":   oauthServer.Config.TokenType,
		"expires_in":   int64(tokenInfo.GetAccessExpiresIn() / time.Second),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(DefaultApiResponse{Message: "reachable"})
}

func PlayerRequestHandler(w http.ResponseWriter, r *http.Request) {
	tokenInfo := r.Context().Value("tokenInfo").(oauth2.TokenInfo)

	var player Player

	var queryError = database.Model(Player{}).
		Preload("AvatarData").
		Where("email = ?", tokenInfo.GetUserID()).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func PlayerCreateHandler(w http.ResponseWriter, r *http.Request) {
	var bodyMap map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&bodyMap)

	username := bodyMap["username"].(string)
	password := bodyMap["password"].(string)

	player := Player{
		Username:  username,
		Password:  password,
		CreatedAt: time.Now(),
	}

	var dbError = database.Create(&player).Error

	if dbError != nil {
		log.Fatalln(dbError)
	}

	// TODO; create avatar data
	// TODO; create player data

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func PlayerSsoTokenHandler(w http.ResponseWriter, r *http.Request) {

}
