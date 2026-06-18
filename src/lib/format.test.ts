import { describe, expect, it } from 'vitest';
import { formatBytes, formatDate } from './format';

describe('formatDate', () => {
	it('formats empty dates as dash', () => {
		expect(formatDate(null)).toBe('-');
		expect(formatDate('')).toBe('-');
	});
});

describe('formatBytes', () => {
	it('formats bytes with binary units', () => {
		expect(formatBytes(512)).toBe('512 B');
		expect(formatBytes(2048)).toBe('2.0 KB');
	});
});
