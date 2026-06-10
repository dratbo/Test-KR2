CREATE TABLE IF NOT EXISTS favorite_users (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    favorite_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, favorite_user_id),
    CHECK (user_id <> favorite_user_id)
);
