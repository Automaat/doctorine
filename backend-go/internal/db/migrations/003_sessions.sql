CREATE TABLE IF NOT EXISTS sessions (
	id serial PRIMARY KEY,
	user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash char(64) NOT NULL UNIQUE,
	expires_at timestamp without time zone NOT NULL,
	revoked_at timestamp without time zone,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE INDEX IF NOT EXISTS ix_sessions_user_id ON sessions (user_id);
CREATE INDEX IF NOT EXISTS ix_sessions_expires_at ON sessions (expires_at);

---- create above / drop below ----

DROP TABLE IF EXISTS sessions;
