CREATE TABLE IF NOT EXISTS result_definitions (
	id serial PRIMARY KEY,
	test_key varchar(120) NOT NULL UNIQUE,
	name varchar(200) NOT NULL,
	unit varchar(80),
	reference_min double precision,
	reference_max double precision,
	category varchar(80) NOT NULL DEFAULT 'laboratory',
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT result_definitions_test_key_check CHECK (test_key ~ '^[a-z0-9_]+$')
);

CREATE INDEX IF NOT EXISTS ix_result_definitions_name ON result_definitions (name);

ALTER TABLE examination_results
	ADD COLUMN IF NOT EXISTS definition_id integer REFERENCES result_definitions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS ix_examination_results_definition_id
	ON examination_results (definition_id);

INSERT INTO result_definitions (test_key, name, unit, reference_min, reference_max)
SELECT DISTINCT ON (test_key)
	test_key,
	name,
	unit,
	reference_min,
	reference_max
FROM examination_results
WHERE test_key <> ''
ORDER BY test_key, updated_at DESC, id DESC
ON CONFLICT (test_key) DO NOTHING;

UPDATE examination_results er
SET definition_id = rd.id
FROM result_definitions rd
WHERE er.definition_id IS NULL
	AND er.test_key = rd.test_key;

---- create above / drop below ----

ALTER TABLE examination_results DROP COLUMN IF EXISTS definition_id;
DROP TABLE IF EXISTS result_definitions;
