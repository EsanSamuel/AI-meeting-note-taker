-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1
LIMIT
    1;

-- name: GetUserWithOrganization :one
SELECT
    u.id AS user_id,
    u.email,
    u.name,
    u.password_hash,
    u.is_active,
    om.organization_id,
    om.role
FROM
    users u
    JOIN organization_members om ON om.user_id = u.id
WHERE
    u.id = $1
LIMIT
    1;

-- name: CreateUser :one
INSERT INTO
    users (email, name, password_hash)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetUser :one
SELECT
    *
FROM
    users
WHERE
    id = $1
LIMIT
    1;

-- name: UpdateUser :one
UPDATE users
SET
    name = COALESCE($2, name),
    is_active = COALESCE($3, is_active)
WHERE
    id = $1
RETURNING
    *;

-- name: UpdateUserRole :exec
UPDATE organization_members
SET role = $3
WHERE
    organization_id = $1
    AND user_id = $2;

-- name: CreateOrganization :one
INSERT INTO
    organizations (name, domain)
VALUES
    ($1, $2)
RETURNING
    *;

-- name: AddOrganizationMember :one
INSERT INTO
    organization_members (organization_id, user_id, role)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: CreateSession :one
INSERT INTO
    sessions (user_id, token_hash, expires_at)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetSessionUser :one
SELECT
    u.id AS user_id,
    u.email,
    u.name,
    u.is_active,
    om.organization_id,
    om.role
FROM
    sessions s
    JOIN users u ON u.id = s.user_id
    JOIN organization_members om ON om.user_id = u.id
WHERE
    s.token_hash = $1
    AND s.expires_at > NOW()
    AND u.is_active = TRUE
LIMIT
    1;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE
    token_hash = $1;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions
WHERE
    expires_at <= NOW();

-- name: CreateInvitation :one
INSERT INTO
    invitations (
        organization_id,
        email,
        name,
        role,
        token_hash,
        expires_at
    )
VALUES
    ($1, $2, $3, $4, $5, NOW() + INTERVAL '24 hours')
RETURNING
    *;

-- name: GetInvitationByTokenHash :one
SELECT
    *
FROM
    invitations
WHERE
    token_hash = $1
    AND accepted_at IS NULL
    --AND expires_at > NOW()
LIMIT
    1;

-- name: AcceptInvitation :exec
UPDATE invitations
SET
    accepted_at = NOW()
WHERE
    id = $1;

-- name: ListOrganizationMembers :many
SELECT
    u.id,
    u.email,
    u.name,
    u.is_active,
    om.role,
    om.created_at
FROM
    organization_members om
    JOIN users u ON u.id = om.user_id
WHERE
    om.organization_id = $1
ORDER BY
    om.created_at DESC;