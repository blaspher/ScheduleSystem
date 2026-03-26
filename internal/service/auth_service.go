package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"schedule-system/internal/dao"
	"schedule-system/internal/model"
	jwtpkg "schedule-system/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var ErrInvalidCredentials = errors.New("invalid username or password")
var ErrUsernameAlreadyExists = errors.New("username already exists")
var ErrInvalidUsername = errors.New("username must not be empty")
var ErrUserNotFound = errors.New("user not found")

type AuthService struct {
	userDAO dao.UserDAO
	jwtMgr  *jwtpkg.Manager
}

type RegisterInput struct {
	Username string
	Password string
}

type LoginInput struct {
	Username string
	Password string
}

type UserBasic struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuthResult struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	User        UserBasic `json:"user"`
}

func NewAuthService(userDAO dao.UserDAO, jwtMgr *jwtpkg.Manager) *AuthService {
	return &AuthService{userDAO: userDAO, jwtMgr: jwtMgr}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" {
		return nil, ErrInvalidUsername
	}

	user, err := s.userDAO.GetByUsername(ctx, username)
	if err == nil && user != nil {
		return nil, ErrUsernameAlreadyExists
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &model.User{
		Username: username,
		Password: string(hashedPassword),
	}
	if err := s.userDAO.Create(ctx, newUser); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrUsernameAlreadyExists
		}
		return nil, err
	}

	token, err := s.jwtMgr.GenerateToken(newUser.ID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return buildAuthResult(token, newUser), nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.userDAO.GetByUsername(ctx, input.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.jwtMgr.GenerateToken(user.ID, 24*time.Hour)
	if err != nil {
		return nil, err
	}
	return buildAuthResult(token, user), nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID uint) (*UserBasic, error) {
	user, err := s.userDAO.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	result := buildUserBasic(user)
	return &result, nil
}

func buildAuthResult(token string, user *model.User) *AuthResult {
	return &AuthResult{
		AccessToken: token,
		TokenType:   "Bearer",
		User:        buildUserBasic(user),
	}
}

func buildUserBasic(user *model.User) UserBasic {
	return UserBasic{
		ID:        user.ID,
		Username:  user.Username,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
