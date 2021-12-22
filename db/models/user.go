package models

import (
	"crypto/subtle"
	"fmt"
	"regexp"
	"time"

	"PhoenixOracle/lib/auth"
	"PhoenixOracle/util"

	"github.com/pkg/errors"
)

type User struct {
	Email             string `gorm:"primary_key"`
	HashedPassword    string
	CreatedAt         time.Time `gorm:"index"`
	TokenKey          string
	TokenSalt         string
	TokenHashedSecret string
	UpdatedAt         time.Time
}

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

const (
	MaxBcryptPasswordLength = 50
)

func NewUser(email, plainPwd string) (User, error) {
	if len(email) == 0 {
		return User{}, errors.New("Must enter an email")
	}

	if !emailRegexp.MatchString(email) {
		return User{}, errors.New("Invalid email format")
	}

	if len(plainPwd) < 8 || len(plainPwd) > MaxBcryptPasswordLength {
		return User{}, fmt.Errorf("must enter a password with 8 - %v characters", MaxBcryptPasswordLength)
	}

	pwd, err := utils.HashPassword(plainPwd)
	if err != nil {
		return User{}, err
	}

	return User{
		Email:          email,
		HashedPassword: pwd,
	}, nil
}

type SessionRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Session struct {
	ID        string    `json:"id" gorm:"primary_key"`
	LastUsed  time.Time `json:"lastUsed" gorm:"index"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
}

func NewSession() Session {
	return Session{
		ID:       utils.NewBytes32ID(),
		LastUsed: time.Now(),
	}
}

type ChangeAuthTokenRequest struct {
	Password string `json:"password"`
}

func (u *User) GenerateAuthToken() (*auth.Token, error) {
	token := auth.NewToken()
	return token, u.SetAuthToken(token)
}

func (u *User) DeleteAuthToken() {
	u.TokenKey = ""
	u.TokenSalt = ""
	u.TokenHashedSecret = ""
}

func (u *User) SetAuthToken(token *auth.Token) error {
	salt := utils.NewSecret(utils.DefaultSecretSize)
	hashedSecret, err := auth.HashedSecret(token, salt)
	if err != nil {
		return errors.Wrap(err, "user")
	}
	u.TokenSalt = salt
	u.TokenKey = token.AccessKey
	u.TokenHashedSecret = hashedSecret
	return nil
}

func AuthenticateUserByToken(token *auth.Token, user *User) (bool, error) {
	hashedSecret, err := auth.HashedSecret(token, user.TokenSalt)
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(hashedSecret), []byte(user.TokenHashedSecret)) == 1, nil
}
