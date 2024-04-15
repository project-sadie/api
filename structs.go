package main

import "time"

type OauthClient struct {
	ID     int64  `json:"id"`
	Secret string `json:"secret"`
	Domain string `json:"domain"`
}

type DefaultApiResponse struct {
	Message string `json:"response_text"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Player struct {
	ID         int64            `json:"id" gorm:"primary_key"`
	Username   string           `json:"username"`
	Password   string           `json:"-"`
	CreatedAt  time.Time        `json:"created_at"`
	AvatarData PlayerAvatarData `json:"avatar_data"`
}

type PlayerAvatarData struct {
	PlayerId int64 `json:"player_id"`
}
