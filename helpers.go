package main

import (
	"fmt"
	"gopkg.in/gomail.v2"
	"math/rand"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"time"
)

func getEnv(key string, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}

	return defaultVal
}

func getEnvAsInt(name string, defaultVal int) int {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return int(value)
	}

	return defaultVal
}

func getEnvAsInt32(name string, defaultVal int32) int32 {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return int32(value)
	}

	return defaultVal
}

func getEnvAsInt64(name string, defaultVal int64) int64 {
	valueStr := getEnv(name, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return int64(value)
	}

	return defaultVal
}

func isValidEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func getUserIp(r *http.Request) string {
	IPAddress := r.Header.Get("X-Real-Ip")

	if IPAddress == "" {
		IPAddress = r.Header.Get("X-Forwarded-For")
	}

	if IPAddress == "" {
		IPAddress = r.RemoteAddr
	}
	return IPAddress
}

func seedRandom() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func sendWelcomeEmail(player Player) {
	siteName := os.Getenv("SITE_NAME")
	subject := fmt.Sprintf("Welcome to %s %s!", siteName, player.Username)
	body := "We're glad you're here."

	m := gomail.NewMessage()

	m.SetHeader("From", os.Getenv("MAIL_FROM_ADDRESS"))
	m.SetHeader("To", player.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := eDialer.DialAndSend(m); err != nil {
		fmt.Println(err)
		panic(err)
	}
}

func sendResetPasswordEmail(player Player, resetId string) {
	siteUrl := os.Getenv("SITE_URL")
	siteName := os.Getenv("SITE_NAME")
	subject := fmt.Sprintf("%s password reset", siteName, player.Username)
	resetLink := fmt.Sprintf("%s/password-reset/%s", siteUrl, resetId)
	body := fmt.Sprintf("You can use the following link to reset your password.<br><a href=\"%s\">%s</a><br><br>This link will expire in 10 minutes.", resetLink, resetLink)

	m := gomail.NewMessage()

	m.SetHeader("From", os.Getenv("MAIL_FROM_ADDRESS"))
	m.SetHeader("To", player.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	if err := eDialer.DialAndSend(m); err != nil {
		fmt.Println(err)
		panic(err)
	}
}
