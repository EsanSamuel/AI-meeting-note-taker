-- 000004_auth.up.sql
CREATE TYPE user_role AS ENUM('owner', 'admin', 'member', 'viewer');

CREATE TABLE
    organizations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        name TEXT NOT NULL,
        domain TEXT,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE
    users (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        email TEXT NOT NULL UNIQUE,
        name TEXT NOT NULL,
        password_hash TEXT,
        is_active BOOLEAN NOT NULL DEFAULT TRUE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE TABLE
    organization_members (
        organization_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        role user_role NOT NULL DEFAULT 'member',
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        PRIMARY KEY (organization_id, user_id)
    );

CREATE TABLE
    sessions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
        token_hash TEXT NOT NULL UNIQUE,
        expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_sessions_user_id ON sessions (user_id);

CREATE INDEX idx_sessions_expires_at ON sessions (expires_at);

CREATE TABLE
    invitations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        organization_id UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
        email TEXT NOT NULL,
        name TEXT NOT NULL,
        role user_role NOT NULL DEFAULT 'member',
        token_hash TEXT NOT NULL UNIQUE,
        expires_at TIMESTAMPTZ NOT NULL,
        accepted_at TIMESTAMPTZ,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    );

CREATE INDEX idx_invitations_organization_id ON invitations (organization_id);

CREATE INDEX idx_invitations_email ON invitations (email);