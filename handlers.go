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

func UserLoginHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	decoder := json.NewDecoder(r.Body)
	var credentials Credentials

	_ = decoder.Decode(&credentials)

	var player Player

	var queryError = database.Model(Player{}).
		Where("email = ?", credentials.Email).
		First(&player).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusBadRequest)

		response := map[string]string{
			"error_message": "Couldn't find any users with this email",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

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
		UserID:       credentials.Email,
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
		Where("email = ?", tokenInfo.GetUserID()).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}
