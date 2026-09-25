package services

import (
	"context"
	"errors"
	"strings"
	"time"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"example.com/internal/auth"
	"example.com/internal/repository"
)

type AuthService interface {
	Setup(ctx context.Context, organizationName, domain, name, email, password string) (repository.User, error)
	Login(ctx context.Context, email, password string) (repository.AuthenticatedUser, string, error)
	Logout(ctx context.Context, token string) error
	GetSessionUser(ctx context.Context, token string) (repository.AuthenticatedUser, error)
	CreateInvitation(ctx context.Context, organizationID uuid.UUID, email, name, role string) (string, error)
	AcceptInvitation(ctx context.Context, token, name, password string) (repository.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (repository.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, name string, isActive bool) (repository.User, error)
	UpdateUserRole(ctx context.Context, organizationID, userID uuid.UUID, role string) error
	ListOrganizationMembers(ctx context.Context, organizationID uuid.UUID) ([]repository.User, error)
}

type authService struct {
	repo repository.AuthRepository
}

func NewAuthService(repo repository.AuthRepository) AuthService {
	return &authService{
		repo: repo,
	}
}

func (s *authService) Setup(ctx context.Context, organizationName, domain, name, email, password string) (repository.User, error) {
	if strings.TrimSpace(organizationName) == "" {
		return repository.User{}, errors.New("organization name is required")
	}

	if strings.TrimSpace(name) == "" {
		return repository.User{}, errors.New("name is required")
	}

	email = strings.ToLower(strings.TrimSpace(email))

	if email == "" {
		return repository.User{}, errors.New("email is required")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return repository.User{}, err
	}

	organization, err := s.repo.CreateOrganization(ctx, repository.CreateOrganizationParams{
		Name: organizationName,
		Domain: pgtype.Text{
			String: domain,
			Valid:  domain != "",
		},
	})
	if err != nil {
		return repository.User{}, err
	}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		Email: email,
		Name:  name,
		PasswordHash: pgtype.Text{
			String: passwordHash,
			Valid:  true,
		},
	})
	if err != nil {
		return repository.User{}, err
	}

	_, err = s.repo.AddOrganizationMember(ctx, organization.ID, user.ID, "owner")
	if err != nil {
		return repository.User{}, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (repository.AuthenticatedUser, string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return repository.AuthenticatedUser{}, "", errors.New("invalid email or password")
	}

	if !user.IsActive || !user.PasswordHash.Valid {
		return repository.AuthenticatedUser{}, "", errors.New("invalid email or password")
	}

	if !auth.VerifyPassword(password, user.PasswordHash.String) {
		return repository.AuthenticatedUser{}, "", errors.New("invalid email or password")
	}

	authenticatedUser, err := s.repo.GetUserOrganization(ctx, user.ID)
	if err != nil {
		return repository.AuthenticatedUser{}, "", err
	}

	token, err := auth.GenerateToken()
	if err != nil {
		return repository.AuthenticatedUser{}, "", err
	}

	_, err = s.repo.CreateSession(ctx, repository.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: auth.HashToken(token),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if err != nil {
		return repository.AuthenticatedUser{}, "", err
	}

	return authenticatedUser, token, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	return s.repo.DeleteSession(ctx, auth.HashToken(token))
}

func (s *authService) GetSessionUser(ctx context.Context, token string) (repository.AuthenticatedUser, error) {
	if token == "" {
		return repository.AuthenticatedUser{}, errors.New("session token is required")
	}

	return s.repo.GetSessionUser(ctx, auth.HashToken(token))
}

func (s *authService) CreateInvitation(ctx context.Context, organizationID uuid.UUID, email, name, role string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	switch role {
	case "admin", "member", "viewer":
	default:
		return "", errors.New("invalid role")
	}

	token, err := auth.GenerateToken()
	if err != nil {
		return "", err
	}

	_, err = s.repo.CreateInvitation(ctx, repository.CreateInvitationParams{
		OrganizationID: organizationID,
		Email:          email,
		Name:           name,
		Role:           role,
		TokenHash:      auth.HashToken(token),
	})
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *authService) AcceptInvitation(ctx context.Context, token, name, password string) (repository.User, error) {
	if token == "" {
		return repository.User{}, errors.New("invitation token is required")
	}

	invitation, err := s.repo.GetInvitationByTokenHash(ctx, auth.HashToken(token))
	if err != nil {
		fmt.Println(err)
		return repository.User{}, errors.New("invalid or expired invitation")
	}

	if invitation.AcceptedAt.Valid {
		return repository.User{}, errors.New("invitation has already been accepted")
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		return repository.User{}, err
	}

	user, err := s.repo.CreateUser(ctx, repository.CreateUserParams{
		Email: invitation.Email,
		Name:  name,
		PasswordHash: pgtype.Text{
			String: passwordHash,
			Valid:  true,
		},
	})
	if err != nil {
		return repository.User{}, err
	}

	_, err = s.repo.AddOrganizationMember(
		ctx,
		invitation.OrganizationID,
		user.ID,
		invitation.Role,
	)
	if err != nil {
		return repository.User{}, err
	}

	if err := s.repo.AcceptInvitation(ctx, invitation.ID); err != nil {
		return repository.User{}, err
	}

	return user, nil
}

func (s *authService) GetUser(ctx context.Context, id uuid.UUID) (repository.User, error) {
	return s.repo.GetUser(ctx, id)
}

func (s *authService) UpdateUser(ctx context.Context, id uuid.UUID, name string, isActive bool) (repository.User, error) {

	return s.repo.UpdateUser(ctx, repository.UpdateUserParams{
		UserID:   id,
		Name:     name,
		IsActive: isActive,
	})
}

func (s *authService) UpdateUserRole(ctx context.Context, organizationID, userID uuid.UUID, role string) error {
	switch role {
	case "owner", "admin", "member", "viewer":
	default:
		return errors.New("invalid role")
	}

	return s.repo.UpdateUserRole(ctx, repository.UpdateUserRoleParams{
		OrganizationID: organizationID,
		UserID:         userID,
		Role:           role,
	})
}

func (s *authService) ListOrganizationMembers(ctx context.Context, organizationID uuid.UUID) ([]repository.User, error) {
	return s.repo.ListOrganizationMembers(ctx, organizationID)
}
