CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY,
    "to" TEXT NOT NULL,
    content TEXT NOT NULL CHECK (char_length(content) <= 140),
    status TEXT NOT NULL CHECK (status IN ('unsent','sent')) DEFAULT 'unsent',
    attempt_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at TIMESTAMPTZ NULL,
    last_error TEXT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_status_created ON messages (status, created_at);
