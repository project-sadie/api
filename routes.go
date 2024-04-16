package main

import (
	"github.com/gorilla/mux"
)

var router *mux.Router

func registerRoutes() {
	router = mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/auth/token", TokenRequestHandler).Methods("GET")
	router.HandleFunc("/auth/login", PlayerLoginHandler).Methods("POST")
	router.HandleFunc("/auth/create", PlayerCreateHandler).Methods("POST")

	router.HandleFunc("/send-password-reset-email", SendForgotPasswordEmailHandler).Methods("POST")
	router.HandleFunc("/reset-password-link/{token}", GetResetPasswordLink).Methods("GET")
	router.HandleFunc("/reset-password-link/{token}", ResetPasswordHandler).Methods("POST")

	router.HandleFunc("/ping", PingHandler)

	authRouter := router.PathPrefix("/").Subrouter()
	authRouter.Use(authorizeMiddleware)

	authRouter.HandleFunc("/auth/me", PlayerRequestHandler)
	authRouter.HandleFunc("/sso-token", PlayerSsoTokenHandler)
}
