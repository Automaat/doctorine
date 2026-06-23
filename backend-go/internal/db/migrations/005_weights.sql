CREATE TABLE IF NOT EXISTS weight_entries (
	id serial PRIMARY KEY,
	measured_on date NOT NULL UNIQUE,
	weight_kg double precision NOT NULL,
	notes text,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT weight_entries_weight_kg_check CHECK (weight_kg > 0 AND weight_kg < 1000)
);

---- create above / drop below ----

DROP TABLE IF EXISTS weight_entries;
