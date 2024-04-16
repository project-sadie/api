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
	"os"
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(DefaultApiResponse{Message: "reachable"})
}

func PlayerRequestHandler(w http.ResponseWriter, r *http.Request) {
	tokenInfo := r.Context().Value("tokenInfo").(oauth2.TokenInfo)

	var player Player

	var queryError = database.Model(Player{}).
		Preload("AvatarData").
		Where("username = ?", tokenInfo.GetUserID()).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func PlayerCreateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var bodyMap map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&bodyMap)

	username := bodyMap["username"].(string)
	email := bodyMap["email"].(string)
	password := bodyMap["password"].(string)
	passwordConfirm := bodyMap["password_confirm"].(string)

	if len(username) < 3 {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The username you've selected is too short"})
		return
	}

	if len(username) > 20 {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The username you've selected is too long"})
		return
	}

	if password != passwordConfirm {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your password confirmation must match your password"})
		return
	}

	if len(password) < 10 {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The password you've selected is too short"})
		return
	}

	var foundPlayer Player
	var queryError = database.Model(Player{}).
		Where("username = ?", username).
		First(&foundPlayer).
		Error

	if !errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The username you've chosen has been taken"})
		return
	}

	if isValidEmail(email) == false {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Please provide a real email address"})
		return
	}

	hashedPassword, hashError := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if hashError != nil {
		log.Fatalln(hashError)
		return
	}

	player := Player{
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now(),
	}

	var dbError = database.Create(&player).Error

	if dbError != nil {
		log.Fatalln(dbError)
		return
	}

	playerData := PlayerData{
		PlayerId:        player.ID,
		HomeRoomId:      getEnvAsInt32("DEFAULT_PLAYER_HOME_ROOM", 0),
		CreditBalance:   getEnvAsInt64("DEFAULT_PLAYER_CREDITS", 10000),
		PixelBalance:    getEnvAsInt64("DEFAULT_PLAYER_PIXELS", 10000),
		SeasonalBalance: getEnvAsInt64("DEFAULT_PLAYER_SEASONAL", 500),
	}

	var dataError = database.Create(&playerData).Error

	if dataError != nil {
		log.Fatalln("Failed to create player data: ", dataError)
		return
	}

	avatarData := PlayerAvatarData{
		PlayerId:     player.ID,
		FigureCode:   os.Getenv("DEFAULT_PLAYER_OUTFIT"),
		Motto:        os.Getenv("DEFAULT_PLAYER_MOTTO"),
		Gender:       "M",
		ChatBubbleId: 1,
	}

	var avatarDataError = database.Create(&avatarData).Error

	if avatarDataError != nil {
		log.Fatalln(avatarDataError)
		return
	}

	gameSettings := PlayerGameSettings{
		PlayerId: player.ID,
	}

	var gameSettingsError = database.Create(&gameSettings).Error

	if gameSettingsError != nil {
		log.Fatalln(gameSettingsError)
		return
	}

	navigatorSettings := PlayerNavigatorSettings{
		PlayerId: player.ID,
	}

	var navigatorSettingsError = database.Create(&navigatorSettings).Error

	if navigatorSettingsError != nil {
		log.Fatalln(navigatorSettingsError)
		return
	}

	websiteData := PlayerWebsiteData{
		PlayerId:  player.ID,
		InitialIp: getUserIp(r),
		LastIp:    getUserIp(r),
	}

	var websiteDataError = database.Create(&websiteData).Error

	if websiteDataError != nil {
		log.Fatalln(websiteDataError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}

func PlayerSsoTokenHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tokenInfo := r.Context().Value("tokenInfo").(oauth2.TokenInfo)

	var player Player

	var queryError = database.Model(Player{}).
		Where("username = ?", tokenInfo.GetUserID()).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
		return
	}

	seedRandom()

	token := PlayerSsoToken{
		PlayerId:  player.ID,
		Token:     randSeq(20),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute * 30),
	}

	var tokenError = database.Create(&token).Error

	if tokenError != nil {
		log.Fatalln(tokenError)
		return
	}

	json.NewEncoder(w).Encode(token)
}
