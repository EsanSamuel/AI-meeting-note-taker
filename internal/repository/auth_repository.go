package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"example.com/internal/db/sqlc"
)

type Organization struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Domain    pgtype.Text `json:"domain,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type User struct {
	ID           uuid.UUID   `json:"id"`
	Email        string      `json:"email"`
	Name         string      `json:"name"`
	IsActive     bool        `json:"is_active"`
	PasswordHash pgtype.Text `json:"-"`
	CreatedAt    time.Time   `json:"created_at"`
}

type OrganizationMember struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Role           string    `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
}

type Session struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type Invitation struct {
	ID             uuid.UUID          `json:"id"`
	OrganizationID uuid.UUID          `json:"organization_id"`
	Email          string             `json:"email"`
	Name           string             `json:"name"`
	Role           string             `json:"role"`
	TokenHash      string             `json:"-"`
	ExpiresAt      time.Time          `json:"expires_at"`
	AcceptedAt     pgtype.Timestamptz `json:"accepted_at,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
}

type AuthenticatedUser struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	IsActive       bool      `json:"is_active"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Role           string    `json:"role"`
}

type CreateOrganizationParams struct {
	Name   string
	Domain pgtype.Text
}

type CreateUserParams struct {
	Email        string
	Name         string
	PasswordHash pgtype.Text
}

type CreateSessionParams struct {
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
}

type CreateInvitationParams struct {
	OrganizationID uuid.UUID
	Email          string
	Name           string
	Role           string
	TokenHash      string
	ExpiresAt      time.Time
}

type UpdateUserParams struct {
	UserID   uuid.UUID
	Name     string
	IsActive bool
}

type UpdateUserRoleParams struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
}

type AuthRepository interface {
	CreateOrganization(ctx context.Context, params CreateOrganizationParams) (Organization, error)
	CreateUser(ctx context.Context, params CreateUserParams) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	GetUser(ctx context.Context, id uuid.UUID) (User, error)
	UpdateUser(ctx context.Context, params UpdateUserParams) (User, error)
	AddOrganizationMember(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, role string) (OrganizationMember, error)
	GetUserOrganization(ctx context.Context, userID uuid.UUID) (AuthenticatedUser, error)
	ListOrganizationMembers(ctx context.Context, organizationID uuid.UUID) ([]User, error)
	UpdateUserRole(ctx context.Context, params UpdateUserRoleParams) error
	CreateSession(ctx context.Context, params CreateSessionParams) (Session, error)
	GetSessionUser(ctx context.Context, tokenHash string) (AuthenticatedUser, error)
	DeleteSession(ctx context.Context, tokenHash string) error
	DeleteExpiredSessions(ctx context.Context) error
	CreateInvitation(ctx context.Context, params CreateInvitationParams) (Invitation, error)
	GetInvitationByTokenHash(ctx context.Context, tokenHash string) (Invitation, error)
	AcceptInvitation(ctx context.Context, id uuid.UUID) error
}

type authRepository struct {
	queries *sqlc.Queries
}

func NewAuthRepository(pool *pgxpool.Pool) AuthRepository {
	return &authRepository{
		queries: sqlc.New(pool),
	}
}

func (r *authRepository) CreateOrganization(ctx context.Context, params CreateOrganizationParams) (Organization, error) {
	org, err := r.queries.CreateOrganization(ctx, sqlc.CreateOrganizationParams{
		Name:   params.Name,
		Domain: params.Domain,
	})
	if err != nil {
		return Organization{}, err
	}

	return Organization{
		ID:        uuid.UUID(org.ID.Bytes),
		Name:      org.Name,
		Domain:    org.Domain,
		CreatedAt: org.CreatedAt.Time,
	}, nil
}

func (r *authRepository) CreateUser(ctx context.Context, params CreateUserParams) (User, error) {
	user, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        params.Email,
		Name:         params.Name,
		PasswordHash: params.PasswordHash,
	})
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           uuid.UUID(user.ID.Bytes),
		Email:        user.Email,
		Name:         user.Name,
		IsActive:     user.IsActive,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
	}, nil
}

func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           uuid.UUID(user.ID.Bytes),
		Email:        user.Email,
		Name:         user.Name,
		IsActive:     user.IsActive,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
	}, nil
}

func (r *authRepository) GetUser(ctx context.Context, id uuid.UUID) (User, error) {
	user, err := r.queries.GetUser(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	})
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           uuid.UUID(user.ID.Bytes),
		Email:        user.Email,
		Name:         user.Name,
		IsActive:     user.IsActive,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
	}, nil
}

func (r *authRepository) UpdateUser(ctx context.Context, params UpdateUserParams) (User, error) {
	user, err := r.queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:       pgtype.UUID{Bytes: params.UserID, Valid: true},
		Name:     params.Name,
		IsActive: params.IsActive,
	})
	if err != nil {
		return User{}, err
	}

	return User{
		ID:           uuid.UUID(user.ID.Bytes),
		Email:        user.Email,
		Name:         user.Name,
		IsActive:     user.IsActive,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt.Time,
	}, nil
}

func (r *authRepository) AddOrganizationMember(ctx context.Context, organizationID uuid.UUID, userID uuid.UUID, role string) (OrganizationMember, error) {
	member, err := r.queries.AddOrganizationMember(ctx, sqlc.AddOrganizationMemberParams{
		OrganizationID: pgtype.UUID{
			Bytes: organizationID,
			Valid: true,
		},
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		Role: sqlc.UserRole(role),
	})
	if err != nil {
		return OrganizationMember{}, err
	}

	return OrganizationMember{
		OrganizationID: uuid.UUID(member.OrganizationID.Bytes),
		UserID:         uuid.UUID(member.UserID.Bytes),
		Role:           string(member.Role),
		CreatedAt:      member.CreatedAt.Time,
	}, nil
}

func (r *authRepository) GetUserOrganization(ctx context.Context, userID uuid.UUID) (AuthenticatedUser, error) {
	user, err := r.queries.GetUserWithOrganization(ctx, pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})
	if err != nil {
		return AuthenticatedUser{}, err
	}

	return AuthenticatedUser{
		UserID:         uuid.UUID(user.UserID.Bytes),
		Email:          user.Email,
		Name:           user.Name,
		IsActive:       user.IsActive,
		OrganizationID: uuid.UUID(user.OrganizationID.Bytes),
		Role:           string(user.Role),
	}, nil
}

func (r *authRepository) ListOrganizationMembers(ctx context.Context, organizationID uuid.UUID) ([]User, error) {
	rows, err := r.queries.ListOrganizationMembers(ctx, pgtype.UUID{
		Bytes: organizationID,
		Valid: true,
	})
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(rows))

	for _, row := range rows {
		users = append(users, User{
			ID:        uuid.UUID(row.ID.Bytes),
			Email:     row.Email,
			Name:      row.Name,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt.Time,
		})
	}

	return users, nil
}

func (r *authRepository) UpdateUserRole(ctx context.Context, params UpdateUserRoleParams) error {
	return r.queries.UpdateUserRole(ctx, sqlc.UpdateUserRoleParams{
		OrganizationID: pgtype.UUID{
			Bytes: params.OrganizationID,
			Valid: true,
		},
		UserID: pgtype.UUID{
			Bytes: params.UserID,
			Valid: true,
		},
		Role: sqlc.UserRole(params.Role),
	})
}

func (r *authRepository) CreateSession(ctx context.Context, params CreateSessionParams) (Session, error) {
	session, err := r.queries.CreateSession(ctx, sqlc.CreateSessionParams{
		UserID: pgtype.UUID{
			Bytes: params.UserID,
			Valid: true,
		},
		TokenHash: params.TokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  params.ExpiresAt,
			Valid: true,
		},
	})
	if err != nil {
		return Session{}, err
	}

	return Session{
		ID:        uuid.UUID(session.ID.Bytes),
		UserID:    uuid.UUID(session.UserID.Bytes),
		TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt.Time,
		CreatedAt: session.CreatedAt.Time,
	}, nil
}

func (r *authRepository) GetSessionUser(ctx context.Context, tokenHash string) (AuthenticatedUser, error) {
	user, err := r.queries.GetSessionUser(ctx, tokenHash)
	if err != nil {
		return AuthenticatedUser{}, err
	}

	return AuthenticatedUser{
		UserID:         uuid.UUID(user.UserID.Bytes),
		Email:          user.Email,
		Name:           user.Name,
		IsActive:       user.IsActive,
		OrganizationID: uuid.UUID(user.OrganizationID.Bytes),
		Role:           string(user.Role),
	}, nil
}

func (r *authRepository) DeleteSession(ctx context.Context, tokenHash string) error {
	return r.queries.DeleteSession(ctx, tokenHash)
}

func (r *authRepository) DeleteExpiredSessions(ctx context.Context) error {
	return r.queries.DeleteExpiredSessions(ctx)
}

func (r *authRepository) CreateInvitation(ctx context.Context, params CreateInvitationParams) (Invitation, error) {
	invitation, err := r.queries.CreateInvitation(ctx, sqlc.CreateInvitationParams{
		OrganizationID: pgtype.UUID{
			Bytes: params.OrganizationID,
			Valid: true,
		},
		Email:     params.Email,
		Name:      params.Name,
		Role:      sqlc.UserRole(params.Role),
		TokenHash: params.TokenHash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  params.ExpiresAt,
			Valid: true,
		},
	})
	if err != nil {
		return Invitation{}, err
	}

	return Invitation{
		ID:             uuid.UUID(invitation.ID.Bytes),
		OrganizationID: uuid.UUID(invitation.OrganizationID.Bytes),
		Email:          invitation.Email,
		Name:           invitation.Name,
		Role:           string(invitation.Role),
		TokenHash:      invitation.TokenHash,
		ExpiresAt:      invitation.ExpiresAt.Time,
		AcceptedAt:     invitation.AcceptedAt,
		CreatedAt:      invitation.CreatedAt.Time,
	}, nil
}

func (r *authRepository) GetInvitationByTokenHash(ctx context.Context, tokenHash string) (Invitation, error) {
	invitation, err := r.queries.GetInvitationByTokenHash(ctx, tokenHash)
	if err != nil {
		return Invitation{}, err
	}

	return Invitation{
		ID:             uuid.UUID(invitation.ID.Bytes),
		OrganizationID: uuid.UUID(invitation.OrganizationID.Bytes),
		Email:          invitation.Email,
		Name:           invitation.Name,
		Role:           string(invitation.Role),
		TokenHash:      invitation.TokenHash,
		ExpiresAt:      invitation.ExpiresAt.Time,
		AcceptedAt:     invitation.AcceptedAt,
		CreatedAt:      invitation.CreatedAt.Time,
	}, nil
}

func (r *authRepository) AcceptInvitation(ctx context.Context, id uuid.UUID) error {
	return r.queries.AcceptInvitation(ctx, pgtype.UUID{
		Bytes: id,
		Valid: true,
	})
}
