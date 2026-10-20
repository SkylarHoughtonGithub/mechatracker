package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/mechatracker/api/internal/middleware"
	"github.com/mechatracker/api/internal/store"
)

func googleFromCtx(c *gin.Context) googleDTO {
	return googleDTO{
		Sub:           c.GetString(middleware.KeyGoogleSub),
		Email:         c.GetString(middleware.KeyGoogleEmail),
		EmailVerified: c.GetBool(middleware.KeyGoogleEmailVerified),
		Name:          c.GetString(middleware.KeyGoogleName),
		PictureURL:    c.GetString(middleware.KeyGooglePicture),
	}
}

func jwtSubject(sub string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{Subject: sub}
}

func userToDTO(u store.User) *userDTO {
	picture := ""
	if u.PictureUrl != nil {
		picture = *u.PictureUrl
	}
	return &userDTO{
		ID:            u.ID.String(),
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		Name:          u.Name,
		PictureURL:    picture,
		Role:          string(u.Role),
		OrgID:         u.OrgID.String(),
	}
}

func orgToDTO(o store.Organization) *orgDTO {
	return &orgDTO{
		ID:   o.ID.String(),
		Name: o.Name,
		Slug: o.Slug,
		Plan: string(o.Plan),
	}
}
