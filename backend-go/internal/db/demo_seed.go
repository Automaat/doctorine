package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedDemoData inserts clearly fake development data. Every statement is
// idempotent so restarting the dev stack does not duplicate rows.
func SeedDemoData(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin demo seed transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	statements := []string{
		seedSupplementsSQL,
		seedIllnessesSQL,
		seedResultDefinitionsSQL,
		seedExaminationsSQL,
		seedExaminationResultsSQL,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("seed demo data: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit demo seed transaction: %w", err)
	}
	return nil
}

const seedSupplementsSQL = `
INSERT INTO supplements (name, value_text, frequency, notes)
SELECT name, value_text, frequency, notes
FROM (
	VALUES
		('Omega 3', '1000mg', 'Daily', 'Demo data: taken with breakfast'),
		('Vitamin D3', '2000 IU', 'Daily', 'Demo data: morning routine'),
		('Magnesium glycinate', '200mg', 'Nightly', 'Demo data: before sleep'),
		('Vitamin B12', '1000 mcg', 'Twice weekly', 'Demo data: vegetarian diet support'),
		('Creatine monohydrate', '5g', 'Daily', 'Demo data: mixed with water')
) AS seed(name, value_text, frequency, notes)
WHERE NOT EXISTS (
	SELECT 1 FROM supplements s
	WHERE s.name = seed.name
		AND s.value_text = seed.value_text
		AND s.frequency = seed.frequency
);
`

const seedIllnessesSQL = `
INSERT INTO illnesses (title, status, diagnosed_on, resolved_on, clinician, notes)
SELECT title, status, diagnosed_on::date, resolved_on::date, clinician, notes
FROM (
	VALUES
		('Demo seasonal allergies', 'monitoring', '2024-04-18', NULL, 'Demo Clinic', 'Synthetic record for development.'),
		('Demo thyroid monitoring', 'monitoring', '2023-10-12', NULL, 'Demo Endocrinology', 'Synthetic record for testing reminders.'),
		('Demo vitamin D insufficiency', 'resolved', '2025-01-20', '2025-08-15', 'Demo Primary Care', 'Synthetic resolved condition.')
) AS seed(title, status, diagnosed_on, resolved_on, clinician, notes)
WHERE NOT EXISTS (
	SELECT 1 FROM illnesses i
	WHERE i.title = seed.title
);
`

const seedResultDefinitionsSQL = `
INSERT INTO result_definitions (test_key, name, unit, reference_min, reference_max, category)
VALUES
	('tsh', 'TSH', 'uIU/mL', 0.4, 4.0, 'laboratory'),
	('glukoza', 'Glucose', 'mg/dL', 70, 99, 'laboratory'),
	('witamina_d_25_oh', 'Vitamin D 25-OH', 'ng/mL', 30, 100, 'laboratory'),
	('ast', 'AST', 'U/L', 11, 34, 'laboratory'),
	('alt', 'ALT', 'U/L', 10, 49, 'laboratory'),
	('kreatynina', 'Creatinine', 'mg/dL', 0.7, 1.2, 'laboratory'),
	('hemoglobina', 'Hemoglobin', 'g/dL', 13.2, 17.1, 'laboratory'),
	('witamina_b12', 'Vitamin B12', 'pg/mL', 211, 911, 'laboratory'),
	('ferrytyna', 'Ferritin', 'ng/mL', 30, 400, 'laboratory'),
	('crp', 'CRP', 'mg/L', NULL, 5, 'laboratory')
ON CONFLICT (test_key) DO UPDATE SET
	name = EXCLUDED.name,
	unit = EXCLUDED.unit,
	reference_min = EXCLUDED.reference_min,
	reference_max = EXCLUDED.reference_max,
	category = EXCLUDED.category,
	updated_at = now() at time zone 'utc';
`

const seedExaminationsSQL = `
INSERT INTO examinations (title, exam_date, category, facility, result_status, summary, notes)
SELECT title, exam_date::date, category, facility, result_status, summary, notes
FROM (
	VALUES
		(
			'Demo annual bloodwork',
			'2026-03-15',
			'laboratory',
			'Demo Diagnostics',
			'normal',
			'Synthetic annual lab panel with mostly normal values.',
			'Generated demo data. Not a real medical record.'
		),
		(
			'Demo nutrition panel',
			'2026-05-08',
			'laboratory',
			'Demo Diagnostics',
			'attention',
			'Synthetic nutrition panel with low ferritin.',
			'Generated demo data. Not a real medical record.'
		),
		(
			'Demo liver follow-up',
			'2025-11-20',
			'laboratory',
			'Demo Diagnostics',
			'attention',
			'Synthetic follow-up panel with elevated AST.',
			'Generated demo data. Not a real medical record.'
		)
) AS seed(title, exam_date, category, facility, result_status, summary, notes)
WHERE NOT EXISTS (
	SELECT 1 FROM examinations e
	WHERE e.title = seed.title
		AND e.exam_date = seed.exam_date::date
);
`

const seedExaminationResultsSQL = `
WITH seed(exam_title, exam_date, test_key, value_numeric, value_text, display_order) AS (
	VALUES
		('Demo annual bloodwork', '2026-03-15', 'tsh', 2.1, '2.1', 1),
		('Demo annual bloodwork', '2026-03-15', 'glukoza', 91, '91', 2),
		('Demo annual bloodwork', '2026-03-15', 'witamina_d_25_oh', 34, '34', 3),
		('Demo annual bloodwork', '2026-03-15', 'ast', 25, '25', 4),
		('Demo annual bloodwork', '2026-03-15', 'kreatynina', 0.92, '0.92', 5),
		('Demo annual bloodwork', '2026-03-15', 'hemoglobina', 14.8, '14.8', 6),
		('Demo nutrition panel', '2026-05-08', 'witamina_d_25_oh', 29, '29', 1),
		('Demo nutrition panel', '2026-05-08', 'witamina_b12', 386, '386', 2),
		('Demo nutrition panel', '2026-05-08', 'ferrytyna', 22, '22', 3),
		('Demo nutrition panel', '2026-05-08', 'hemoglobina', 13.7, '13.7', 4),
		('Demo liver follow-up', '2025-11-20', 'ast', 44, '44', 1),
		('Demo liver follow-up', '2025-11-20', 'alt', 38, '38', 2),
		('Demo liver follow-up', '2025-11-20', 'crp', 2.1, '2.1', 3)
)
INSERT INTO examination_results (
	examination_id,
	definition_id,
	test_key,
	name,
	value_text,
	value_numeric,
	unit,
	reference_min,
	reference_max,
	flag,
	display_order
)
SELECT
	e.id,
	rd.id,
	rd.test_key,
	rd.name,
	seed.value_text,
	seed.value_numeric,
	rd.unit,
	rd.reference_min,
	rd.reference_max,
	CASE
		WHEN rd.reference_min IS NOT NULL AND seed.value_numeric < rd.reference_min THEN 'L'
		WHEN rd.reference_max IS NOT NULL AND seed.value_numeric > rd.reference_max THEN 'H'
		ELSE NULL
	END,
	seed.display_order
FROM seed
JOIN examinations e ON e.title = seed.exam_title AND e.exam_date = seed.exam_date::date
JOIN result_definitions rd ON rd.test_key = seed.test_key
ON CONFLICT (examination_id, test_key) DO UPDATE SET
	definition_id = EXCLUDED.definition_id,
	name = EXCLUDED.name,
	value_text = EXCLUDED.value_text,
	value_numeric = EXCLUDED.value_numeric,
	unit = EXCLUDED.unit,
	reference_min = EXCLUDED.reference_min,
	reference_max = EXCLUDED.reference_max,
	flag = EXCLUDED.flag,
	display_order = EXCLUDED.display_order,
	updated_at = now() at time zone 'utc';
`
