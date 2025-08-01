package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-oauth2/oauth2/v4"
	"github.com/go-playground/validator/v10"
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
	var credentials Credentials

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

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

	json.NewEncoder(w).Encode(response)
}

func PingHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(DefaultApiResponse{Message: time.Now().In(location).String()})
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

	json.NewEncoder(w).Encode(player)
}

func PlayerCreateHandler(w http.ResponseWriter, r *http.Request) {
	var req PlayerCreateRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Invalid JSON body"})
		return
	}

	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Validation failed"})
		return
	}

	var foundPlayer Player
	if err := database.Model(Player{}).Where("username = ?", req.Username).First(&foundPlayer).Error; err == nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The username you've chosen has been taken"})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: err.Error()})
		return
	}

	var foundEmail Player
	if err := database.Model(Player{}).Where("email = ?", req.Email).First(&foundEmail).Error; err == nil {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "The email you've chosen has been taken"})
		return
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: err.Error()})
		return
	}

	if getEnvAsInt("MAX_ACCOUNTS_PER_IP", 5) != 0 {
		var count int
		if err := database.Model(PlayerWebsiteData{}).
			Where("initial_ip = ?", getUserIp(r)).
			Count(&count).Error; err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: err.Error()})
			return
		}
		if count >= getEnvAsInt("MAX_ACCOUNTS_PER_IP", 5) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Too many accounts, try again soon!"})
			return
		}
	}

	if !isValidEmail(req.Email) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(DefaultApiResponse{Message: "Please provide a real email address"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln(err)
		return
	}

	player := Player{
		Username:  req.Username,
		Email:     req.Email,
		Password:  string(hashedPassword),
		CreatedAt: time.Now().In(location),
	}

	if err := database.Create(&player).Error; err != nil {
		log.Fatalln(err)
		return
	}

	playerData := PlayerData{
		PlayerId:        player.ID,
		CreditBalance:   getEnvAsInt64("DEFAULT_PLAYER_CREDITS", 10000),
		PixelBalance:    getEnvAsInt64("DEFAULT_PLAYER_PIXELS", 10000),
		SeasonalBalance: getEnvAsInt64("DEFAULT_PLAYER_SEASONAL", 500),
		GotwPoints:      0,
		LastOnline:      time.Now().In(location),
	}

	if err := database.Create(&playerData).Error; err != nil {
		log.Fatalln("Failed to create player data: ", err)
		return
	}

	avatarData := PlayerAvatarData{
		PlayerId:     player.ID,
		FigureCode:   os.Getenv("DEFAULT_PLAYER_OUTFIT"),
		Motto:        os.Getenv("DEFAULT_PLAYER_MOTTO"),
		Gender:       "M",
		ChatBubbleId: 1,
	}

	if err := database.Create(&avatarData).Error; err != nil {
		log.Fatalln(err)
		return
	}

	gameSettings := PlayerGameSettings{PlayerId: player.ID}
	if err := database.Create(&gameSettings).Error; err != nil {
		log.Fatalln(err)
		return
	}

	navigatorSettings := PlayerNavigatorSettings{PlayerId: player.ID}
	if err := database.Create(&navigatorSettings).Error; err != nil {
		log.Fatalln(err)
		return
	}

	websiteData := PlayerWebsiteData{
		PlayerId:  player.ID,
		InitialIp: getUserIp(r),
		LastIp:    getUserIp(r),
		LastLogin: time.Now().In(location),
	}

	if err := database.Create(&websiteData).Error; err != nil {
		log.Fatalln(err)
		return
	}

	if os.Getenv("SEND_WELCOME_EMAIL") == "true" {
		sendWelcomeEmail(player)
	}

	json.NewEncoder(w).Encode(player)
}

func PlayerSsoTokenHandler(w http.ResponseWriter, r *http.Request) {
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
		CreatedAt: time.Now().In(location),
		ExpiresAt: time.Now().In(location).Add(time.Minute * 30),
	}

	var tokenError = database.Create(&token).Error

	if tokenError != nil {
		log.Fatalln(tokenError)
		return
	}

	json.NewEncoder(w).Encode(token)
}

func SendForgotPasswordEmailHandler(w http.ResponseWriter, r *http.Request) {
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
		Where("expires_at > ?", time.Now().In(location)).
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
		CreatedAt: time.Now().In(location),
		ExpiresAt: time.Now().In(location).Add(time.Minute * 10),
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
		Where("expires_at > ?", time.Now().In(location)).
		Where("used_at IS NULL").
		First(&resetLink).
		Error

	if errors.Is(queryError, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(resetLink)
}

func UseResetPasswordLink(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	var resetLink PlayerPasswordResetLink

	var queryError = database.Model(PlayerPasswordResetLink{}).
		Where("token = ?", params["token"]).
		Where("expires_at > ?", time.Now().In(location)).
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
		Update("used_at", time.Now().In(location))

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

	json.NewEncoder(w).Encode(roles)
}

func UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
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

	json.NewEncoder(w).Encode(player)
}
