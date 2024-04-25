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

	router.HandleFunc("/reset-password/send-email", SendForgotPasswordEmailHandler).Methods("POST")

	router.HandleFunc("/reset-password/{token}", GetResetPasswordLink).Methods("GET")
	router.HandleFunc("/reset-password/{token}", UseResetPasswordLink).Methods("POST")

	router.HandleFunc("/ping", PingHandler).Methods("GET")

	authRouter := router.PathPrefix("/").Subrouter()
	authRouter.Use(authorizeMiddleware)

	authRouter.HandleFunc("/auth/me", PlayerRequestHandler).Methods("GET")

	authRouter.HandleFunc("/settings", UpdateSettingsHandler).Methods("POST")

	authRouter.HandleFunc("/sso-token", PlayerSsoTokenHandler).Methods("GET")
	authRouter.HandleFunc("/roles", RolesHandler).Methods("GET")
}
