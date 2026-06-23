CREATE TABLE IF NOT EXISTS personal_access_tokens (
	id serial PRIMARY KEY,
	user_id integer NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	token_hash char(64) NOT NULL UNIQUE,
	name varchar(120) NOT NULL,
	scope varchar(40) NOT NULL DEFAULT 'full',
	expires_at timestamp without time zone,
	last_used_at timestamp without time zone,
	revoked_at timestamp without time zone,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT personal_access_tokens_scope_check CHECK (scope IN ('full', 'read'))
);

CREATE INDEX IF NOT EXISTS ix_personal_access_tokens_user_id
	ON personal_access_tokens (user_id);

---- create above / drop below ----

DROP TABLE IF EXISTS personal_access_tokens;
