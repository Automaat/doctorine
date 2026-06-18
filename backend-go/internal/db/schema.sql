CREATE TABLE IF NOT EXISTS users (
	id serial PRIMARY KEY,
	username varchar(80) NOT NULL UNIQUE,
	password_hash text NOT NULL,
	is_admin boolean NOT NULL DEFAULT false,
	display_name varchar(160),
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE TABLE IF NOT EXISTS illnesses (
	id serial PRIMARY KEY,
	title varchar(200) NOT NULL,
	status varchar(40) NOT NULL DEFAULT 'active',
	diagnosed_on date,
	resolved_on date,
	clinician varchar(200),
	notes text,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT illnesses_status_check CHECK (status IN ('active', 'monitoring', 'resolved'))
);

CREATE INDEX IF NOT EXISTS ix_illnesses_status_created_at ON illnesses (status, created_at DESC);

CREATE TABLE IF NOT EXISTS examinations (
	id serial PRIMARY KEY,
	title varchar(200) NOT NULL,
	exam_date date NOT NULL,
	category varchar(80) NOT NULL DEFAULT 'general',
	facility varchar(200),
	result_status varchar(40) NOT NULL DEFAULT 'unknown',
	summary text,
	notes text,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	updated_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc'),
	CONSTRAINT examinations_result_status_check CHECK (
		result_status IN ('unknown', 'normal', 'attention', 'urgent')
	)
);

CREATE INDEX IF NOT EXISTS ix_examinations_exam_date ON examinations (exam_date DESC);
CREATE INDEX IF NOT EXISTS ix_examinations_result_status ON examinations (result_status);

CREATE TABLE IF NOT EXISTS documents (
	id serial PRIMARY KEY,
	title varchar(240) NOT NULL,
	document_type varchar(80) NOT NULL DEFAULT 'medical',
	issued_at date,
	original_filename varchar(255) NOT NULL,
	storage_name varchar(255) NOT NULL UNIQUE,
	content_type varchar(120) NOT NULL,
	size_bytes bigint NOT NULL CHECK (size_bytes > 0),
	sha256_hex char(64) NOT NULL,
	notes text,
	illness_id integer REFERENCES illnesses(id) ON DELETE SET NULL,
	examination_id integer REFERENCES examinations(id) ON DELETE SET NULL,
	created_at timestamp without time zone NOT NULL DEFAULT (now() at time zone 'utc')
);

CREATE INDEX IF NOT EXISTS ix_documents_created_at ON documents (created_at DESC);
CREATE INDEX IF NOT EXISTS ix_documents_issued_at ON documents (issued_at DESC NULLS LAST);
CREATE INDEX IF NOT EXISTS ix_documents_illness_id ON documents (illness_id);
CREATE INDEX IF NOT EXISTS ix_documents_examination_id ON documents (examination_id);
