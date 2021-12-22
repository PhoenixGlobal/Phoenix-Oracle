package controllers

import (
	"errors"
	"fmt"
	"net/http"

	"PhoenixOracle/core/service/phoenix"
	"PhoenixOracle/db/models"

	"PhoenixOracle/web"
	"github.com/gin-gonic/contrib/sessions"
	"github.com/gin-gonic/gin"
	"go.uber.org/multierr"
)

type SessionsController struct {
	App phoenix.Application
}

func (sc *SessionsController) Create(c *gin.Context) {
	defer sc.App.WakeSessionReaper()

	session := sessions.Default(c)
	var sr models.SessionRequest
	if err := c.ShouldBindJSON(&sr); err != nil {
		web.JsonAPIError(c, http.StatusBadRequest, fmt.Errorf("error binding json %v", err))
		return
	}

	sid, err := sc.App.GetStore().CreateSession(sr)
	if err != nil {
		web.JsonAPIError(c, http.StatusUnauthorized, err)
		return
	}
	if err := saveSessionID(session, sid); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, multierr.Append(errors.New("unable to save session id"), err))
		return
	}

	web.JsonAPIResponse(c, Session{Authenticated: true}, "session")
}

func (sc *SessionsController) Destroy(c *gin.Context) {
	defer sc.App.WakeSessionReaper()

	session := sessions.Default(c)
	defer session.Clear()
	sessionID, ok := session.Get(web.SessionIDKey).(string)
	if !ok {
		web.JsonAPIResponse(c, Session{Authenticated: false}, "session")
		return
	}
	if err := sc.App.GetStore().DeleteUserSession(sessionID); err != nil {
		web.JsonAPIError(c, http.StatusInternalServerError, err)
		return
	}

	web.JsonAPIResponse(c, Session{Authenticated: false}, "session")
}

func saveSessionID(session sessions.Session, sessionID string) error {
	session.Set(web.SessionIDKey, sessionID)
	return session.Save()
}

type Session struct {
	Authenticated bool `json:"authenticated"`
}

func (s Session) GetID() string {
	return "sessionID"
}

func (Session) GetName() string {
	return "session"
}

func (*Session) SetID(string) error {
	return nil
}
