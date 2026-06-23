CREATE TABLE IF NOT EXISTS test_orders (
	id serial PRIMARY KEY,
	source varchar(40) NOT NULL DEFAULT 'coach',
	test_keys text[] NOT NULL,
	reason text,
	status varchar(20) NOT NULL DEFAULT 'requested',
	requested_on date NOT NULL DEFAULT ((now() at time zone 'utc')::date),
	due_on date,
	examination_id integer REFERENCES examinations(id) ON DELETE SET NULL,
	notes text,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT test_orders_status_check CHECK (status IN ('requested', 'completed', 'canceled')),
	CONSTRAINT test_orders_test_keys_not_empty CHECK (cardinality(test_keys) > 0)
);

CREATE INDEX IF NOT EXISTS ix_test_orders_status ON test_orders (status, requested_on DESC);

---- create above / drop below ----

DROP TABLE IF EXISTS test_orders;
