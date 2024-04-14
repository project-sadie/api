package main

type OauthClient struct {
	ID     int64  `json:"id"`
	Secret string `json:"secret"`
	Domain string `json:"domain"`
}

type DefaultApiResponse struct {
	Message string `json:"response_text"`
}

type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Player struct {
	ID       int64  `json:"id" gorm:"primary_key"`
	Username string `json:"username"`
	Password string `json:"-"`
}
