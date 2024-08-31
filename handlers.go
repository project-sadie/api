package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/gorilla/mux"
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
		w.WriteHeader(http.StatusUnauthorized)

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
		w.WriteHeader(http.StatusUnauthorized)

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
	json.NewEncoder(w).Encode(DefaultApiResponse{Message: time.Now().String()})
}

func PlayerRequestHandler(w http.ResponseWriter, r *http.Request) {
	tokenInfo := r.Context().Value("tokenInfo").(oauth2.TokenInfo)

	var player Player

	var queryError = database.Model(Player{}).
		Preload("Data").
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
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your confirmation must match your password"})
		return
	}

	if len(password) < getEnvAsInt("VALIDATION_MIN_PASSWORD_LENGTH", 10) {
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

	var foundEmail Player
	var emailQueryError = database.Model(Player{}).
		Where("email = ?", email).
		First(&foundEmail).
		Error

	if !errors.Is(emailQueryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The email you've chosen has been taken"})
		return
	}

	if getEnvAsInt("MAX_ACCOUNTS_PER_IP", 5) != 0 {
		var count int

		countError := database.Model(PlayerWebsiteData{}).
			Where("initial_ip = ?", getUserIp(r)).
			Count(&count).
			Error

		if countError != nil {
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: countError.Error()})
			return
		}

		if count > getEnvAsInt("MAX_ACCOUNTS_PER_IP", 5) {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Too many accounts, try again soon!"})
			return
		}
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
		CreditBalance:   getEnvAsInt64("DEFAULT_PLAYER_CREDITS", 10000),
		PixelBalance:    getEnvAsInt64("DEFAULT_PLAYER_PIXELS", 10000),
		SeasonalBalance: getEnvAsInt64("DEFAULT_PLAYER_SEASONAL", 500),
		GotwPoints:      0,
		LastOnline: 	 time.Now(),
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
		LastLogin: time.Now(),
	}

	var websiteDataError = database.Create(&websiteData).Error

	if websiteDataError != nil {
		log.Fatalln(websiteDataError)
		return
	}

	if os.Getenv("SEND_WELCOME_EMAIL") == "true" {
		sendWelcomeEmail(player)
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
		Token:     randSeq(30),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute * 230),
	}

	var tokenError = database.Create(&token).Error

	if tokenError != nil {
		log.Fatalln(tokenError)
		return
	}

	json.NewEncoder(w).Encode(token)
}

func SendForgotPasswordEmailHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var player Player

	var bodyMap map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&bodyMap)

	email := bodyMap["email"].(string)

	var queryError = database.Model(Player{}).
		Where("email = ?", email).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
		return
	}

	var count int

	countError := database.Model(PlayerPasswordResetLink{}).
		Where("player_id = ?", player.ID).
		Where("expires_at > ?", time.Now()).
		Where("used_at IS NULL").
		Count(&count).
		Error

	if countError != nil {
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: countError.Error()})
		return
	}

	if count > getEnvAsInt("MAX_PASSWORD_RESETS_PER_HOUR", 5) {
		w.WriteHeader(429)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "You're doing too much, slow down!"})
		return
	}

	resetLink := PlayerPasswordResetLink{
		PlayerId:  player.ID,
		Token:     randSeq(30),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Minute * 10),
	}

	var resetLinkError = database.Create(&resetLink).Error

	if resetLinkError != nil {
		log.Fatalln(resetLinkError)
		return
	}

	sendResetPasswordEmail(player, resetLink.Token)
	json.NewEncoder(w).Encode(DefaultApiResponse{Message: "We've sent you an email"})
}

func GetResetPasswordLink(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var resetLink PlayerPasswordResetLink

	var queryError = database.Model(PlayerPasswordResetLink{}).
		Where("token = ?", params["token"]).
		Where("expires_at > ?", time.Now()).
		Where("used_at IS NULL").
		First(&resetLink).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resetLink)
}

func UseResetPasswordLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)

	var resetLink PlayerPasswordResetLink

	var queryError = database.Model(PlayerPasswordResetLink{}).
		Where("token = ?", params["token"]).
		Where("expires_at > ?", time.Now()).
		Where("used_at IS NULL").
		First(&resetLink).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var bodyMap map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&bodyMap)

	password := bodyMap["password"].(string)
	passwordConfirm := bodyMap["password_confirm"].(string)

	if password != passwordConfirm {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your confirmation must match your password"})
		return
	}

	if len(password) < getEnvAsInt("VALIDATION_MIN_PASSWORD_LENGTH", 10) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The password you've selected is too short"})
		return
	}

	hashedPassword, hashError := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Println(password)

	if hashError != nil {
		log.Fatalln(hashError)
		return
	}

	database.
		Model(&resetLink).
		Update("used_at", time.Now())

	var player Player

	var playerError = database.Model(Player{}).
		Where("id = ?", resetLink.PlayerId).
		First(&player).
		Error

	if errors.Is(playerError, gorm.ErrRecordNotFound) {
		log.Fatalln(playerError)
		return
	}

	database.
		Model(&player).
		Update("password", hashedPassword)

	json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your password has been updated"})
}

func RolesHandler(w http.ResponseWriter, r *http.Request) {
	var roles []Role

	var queryError = database.Model(&Role{}).
		Preload("Players.Data").
		Preload("Players.AvatarData").
		Where("id > 1").
		Find(&roles).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(roles)
}

func UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var bodyMap map[string]interface{}

	decoder := json.NewDecoder(r.Body)
	_ = decoder.Decode(&bodyMap)

	email := bodyMap["email"].(string)
	motto := bodyMap["motto"].(string)
	password := bodyMap["password"].(string)

	tokenInfo := r.Context().Value("tokenInfo").(oauth2.TokenInfo)

	var player Player

	var queryError = database.Model(Player{}).
		Preload("Data").
		Preload("AvatarData").
		Where("username = ?", tokenInfo.GetUserID()).
		First(&player).
		Error

	if queryError != nil {
		log.Fatalln(queryError)
	}

	if len(email) < 5 || len(email) > 30 || !isValidEmail(email) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Please provide a valid email address"})
		return
	}

	if len(motto) > 30 {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "This motto is too long"})
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(player.Password), []byte(password)) != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your current password is incorrect"})
		return
	}

	if newPassword, ok := bodyMap["new_password"].(string); ok {
		if len(newPassword) < getEnvAsInt("VALIDATION_MIN_PASSWORD_LENGTH", 10) {
			w.WriteHeader(403)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The password you've selected is too short"})
			return
		}

		hashedPassword, hashError := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)

		if hashError != nil {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Something went wrong"})
			log.Fatalln(queryError)
			return
		}

		database.
			Model(&player).
			Update("password", hashedPassword)

	}

	database.
		Model(&player).
		Update("email", email)

	database.
		Model(&player.AvatarData).
		Update("motto", motto)

	json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Your changes have been saved"})
}

func GetPlayerProfileHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var player Player

	var queryError = database.Model(Player{}).
		Preload("Data").
		Preload("AvatarData").
		Where("username = ?", params["username"]).
		First(&player).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The requested profile couldn't be found"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(player)
}
