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
	Email      string           `json:"email"`
	Password   string           `json:"-"`
	CreatedAt  time.Time        `json:"created_at"`
	AvatarData PlayerAvatarData `json:"avatar_data"`
}

type PlayerData struct {
	ID              int64 `json:"id" gorm:"primary_key"`
	PlayerId        int64 `json:"player_id"`
	HomeRoomId      int32 `json:"home_room_id"`
	CreditBalance   int64 `json:"credit_balance"`
	PixelBalance    int64 `json:"pixel_balance"`
	SeasonalBalance int64 `json:"seasonal_balance"`
}

type PlayerAvatarData struct {
	ID           int64  `json:"id" gorm:"primary_key"`
	PlayerId     int64  `json:"player_id"`
	FigureCode   string `json:"figure_code"`
	Motto        string `json:"motto"`
	Gender       string `json:"gender"`
	ChatBubbleId int32  `json:"chat_bubble_id"`
}

type PlayerGameSettings struct {
	ID       int64 `json:"id" gorm:"primary_key"`
	PlayerId int64 `json:"player_id"`
}

type PlayerNavigatorSettings struct {
	ID       int64 `json:"id" gorm:"primary_key"`
	PlayerId int64 `json:"player_id"`
}

type PlayerWebsiteData struct {
	ID        int64  `json:"id" gorm:"primary_key"`
	PlayerId  int64  `json:"player_id"`
	InitialIp string `json:"initial_ip"`
	LastIp    string `json:"last_ip"`
}

type PlayerSsoToken struct {
	ID        int64     `json:"id" gorm:"primary_key"`
	PlayerId  int64     `json:"player_id"`
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type PlayerPasswordResetLink struct {
	ID        int64      `json:"id" gorm:"primary_key"`
	PlayerId  int64      `json:"player_id"`
	Token     string     `json:"token"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at" gorm:"type:TIMESTAMP;null;default:null"`
}
