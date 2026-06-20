CREATE TABLE IF NOT EXISTS supplements (
	id serial PRIMARY KEY,
	name varchar(200) NOT NULL,
	value_text varchar(120) NOT NULL,
	frequency varchar(120) NOT NULL,
	notes text,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE INDEX IF NOT EXISTS ix_supplements_name ON supplements (name);
CREATE INDEX IF NOT EXISTS ix_supplements_created_at ON supplements (created_at DESC);

---- create above / drop below ----

DROP TABLE IF EXISTS supplements;
