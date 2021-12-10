package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/models"
	"PhoenixOracle/util"
	"PhoenixOracle/web/presenters"

	"PhoenixOracle/web"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	App phoenix.Application
}

type UpdatePasswordRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (c *UserController) UpdatePassword(ctx *gin.Context) {
	var request UpdatePasswordRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(ctx, http.StatusUnprocessableEntity, err)
		return
	}

	user, err := c.App.GetStore().FindUser()
	if err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to obtain current user record: %+v", err))
		return
	}
	if !utils.CheckPasswordHash(request.OldPassword, user.HashedPassword) {
		web.JsonAPIError(ctx, http.StatusConflict, errors.New("old password does not match"))
		return
	}
	if err := c.updateUserPassword(ctx, &user, request.NewPassword); err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(ctx, presenters.NewUserResource(user), "user")
}

func (c *UserController) NewAPIToken(ctx *gin.Context) {
	var request models.ChangeAuthTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(ctx, http.StatusUnprocessableEntity, err)
		return
	}

	user, err := c.App.GetStore().FindUser()
	if err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to obtain current user record: %+v", err))
		return
	}
	if !utils.CheckPasswordHash(request.Password, user.HashedPassword) {
		web.JsonAPIError(ctx, http.StatusUnauthorized, errors.New("incorrect password"))
		return
	}
	newToken, err := user.GenerateAuthToken()
	if err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := c.App.GetStore().SaveUser(&user); err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponseWithStatus(ctx, newToken, "auth_token", http.StatusCreated)
}

func (c *UserController) DeleteAPIToken(ctx *gin.Context) {
	var request models.ChangeAuthTokenRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		web.JsonAPIError(ctx, http.StatusUnprocessableEntity, err)
		return
	}

	user, err := c.App.GetStore().FindUser()
	if err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to obtain current user record: %+v", err))
		return
	}
	if !utils.CheckPasswordHash(request.Password, user.HashedPassword) {
		web.JsonAPIError(ctx, http.StatusUnauthorized, errors.New("incorrect password"))
		return
	}
	if user.DeleteAuthToken(); false {
		web.JsonAPIError(ctx, http.StatusInternalServerError, err)
		return
	}
	if err := c.App.GetStore().SaveUser(&user); err != nil {
		web.JsonAPIError(ctx, http.StatusInternalServerError, err)
		return
	}
	{
		web.JsonAPIResponseWithStatus(ctx, nil, "auth_token", http.StatusNoContent)
	}
}

func (c *UserController) getCurrentSessionID(ctx *gin.Context) (string, error) {
	session := sessions.Default(ctx)
	sessionID, ok := session.Get(web.SessionIDKey).(string)
	if !ok {
		return "", errors.New("unable to get current session ID")
	}
	return sessionID, nil
}

func (c *UserController) saveNewPassword(user *models.User, newPassword string) error {
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.HashedPassword = hashedPassword
	return c.App.GetStore().SaveUser(user)
}

func (c *UserController) updateUserPassword(ctx *gin.Context, user *models.User, newPassword string) error {
	sessionID, err := c.getCurrentSessionID(ctx)
	if err != nil {
		return err
	}
	if err := c.App.GetStore().ClearNonCurrentSessions(sessionID); err != nil {
		return fmt.Errorf("failed to clear non current user sessions: %+v", err)
	}
	if err := c.saveNewPassword(user, newPassword); err != nil {
		return fmt.Errorf("failed to update current user password: %+v", err)
	}
	return nil
}
