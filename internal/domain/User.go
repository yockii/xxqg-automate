package domain

import "xxqg-automate/internal/model"

type UserLoginReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type UserLoginResp struct {
	User  *model.User `json:"user,omitempty"`
	Token string      `json:"token,omitempty"`
}
