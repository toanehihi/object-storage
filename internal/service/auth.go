package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/internal/token"
)

var (
	ErrEmailTaken      = errors.New("email already registered")
	ErrInvalidPassword = errors.New("invalid password")
)

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"fullName"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type AuthResponse struct {
	User  *model.User `json:"user"`
	Token *token.Pair `json:"token"`
}

type AuthService struct {
	repo         repository.UserRepository
	tokenManager *token.Manager
}

func NewAuthService(repo repository.UserRepository, tokenManager *token.Manager) *AuthService {
	return &AuthService{
		repo:         repo,
		tokenManager: tokenManager,
	}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	_, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailTaken
	}
	if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &model.User{
		ID:           uuid.New(),
		Email:        req.Email,
		PasswordHash: string(hash),
		FullName:     req.FullName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	tokens, err := s.tokenManager.GeneratePair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Token: tokens}, nil
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrInvalidPassword
	}
	if err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidPassword
	}

	tokens, err := s.tokenManager.GeneratePair(user.ID)
	if err != nil {
		return nil, err
	}

	return &AuthResponse{User: user, Token: tokens}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req RefreshRequest) (*token.Pair, error) {
	claims, err := s.tokenManager.Validate(req.RefreshToken)
	if err != nil {
		return nil, token.ErrInvalidToken
	}

	if _, err := s.repo.GetByID(ctx, claims.UserID); err != nil {
		return nil, err
	}

	return s.tokenManager.GeneratePair(claims.UserID)
}

func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	return s.repo.GetByID(ctx, userID)
}
