package main

import (
	"github.com/gorilla/mux"
)

var router *mux.Router

func registerRoutes() {
	router = mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/auth/token", TokenRequestHandler).Methods("GET")
	router.HandleFunc("/auth/login", UserLoginHandler).Methods("POST")

	router.HandleFunc("/ping", PingHandler)

	authRouter := router.PathPrefix("/").Subrouter()
	authRouter.Use(authorizeMiddleware)

	authRouter.HandleFunc("/auth/me", PlayerRequestHandler)
}
