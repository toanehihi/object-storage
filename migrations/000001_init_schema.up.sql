CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name VARCHAR(255),
    avatar_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY,

    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    filename TEXT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,

    size BIGINT NOT NULL,
    content_type TEXT,

    status VARCHAR(30) NOT NULL,

    checksum TEXT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS file_chunks (
    id UUID PRIMARY KEY,

    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,

    chunk_index INT NOT NULL,

    etag TEXT,

    uploaded BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(file_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_file_chunks_file_id
ON file_chunks(file_id);

CREATE INDEX IF NOT EXISTS idx_files_owner_id
ON files(owner_id);

CREATE INDEX IF NOT EXISTS idx_files_status
ON files(status);

CREATE INDEX IF NOT EXISTS idx_files_checksum
ON files(checksum);

CREATE INDEX IF NOT EXISTS idx_users_email
ON users(email);