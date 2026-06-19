export interface DocumentRecord {
	id: number;
	title: string;
	document_type: string;
	issued_at: string | null;
	original_filename: string;
	content_type: string;
	size_bytes: number;
	sha256_hex: string;
	notes: string | null;
	illness_id: number | null;
	illness_title: string | null;
	examination_id: number | null;
	examination_title: string | null;
	created_at: string;
}

export interface Illness {
	id: number;
	title: string;
	status: 'active' | 'monitoring' | 'resolved';
	diagnosed_on: string | null;
	resolved_on: string | null;
	clinician: string | null;
	notes: string | null;
	created_at: string;
	updated_at: string;
}

export interface Examination {
	id: number;
	title: string;
	exam_date: string;
	category: string;
	facility: string | null;
	result_status: 'unknown' | 'normal' | 'attention' | 'urgent';
	summary: string | null;
	notes: string | null;
	results: ExaminationResult[];
	created_at: string;
	updated_at: string;
}

export interface ExaminationResult {
	id: number;
	examination_id: number;
	definition_id: number | null;
	definition: ResultDefinition | null;
	test_key: string;
	name: string;
	value_text: string | null;
	value_numeric: number | null;
	value_prefix: '<' | '>' | '<=' | '>=' | null;
	unit: string | null;
	reference_min: number | null;
	reference_max: number | null;
	flag: string | null;
	display_order: number;
	created_at: string;
	updated_at: string;
}

export interface ResultDefinition {
	id: number;
	test_key: string;
	name: string;
	unit: string | null;
	reference_min: number | null;
	reference_max: number | null;
	category: string;
	created_at: string;
	updated_at: string;
}

export interface Overview {
	document_count: number;
	illness_count: number;
	examination_count: number;
	flagged_results: number;
	recent_documents: DocumentRecord[];
}
