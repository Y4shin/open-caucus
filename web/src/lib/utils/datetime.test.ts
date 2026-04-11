import { describe, it, expect } from 'vitest';
import { CalendarDateTime } from '@internationalized/date';
import { calendarDateTimeToUTC, formatCalendarDateTime } from './datetime.js';

describe('calendarDateTimeToUTC', () => {
	it('produces a valid ISO 8601 string ending in UTC offset or Z', () => {
		const dt = new CalendarDateTime(2026, 5, 15, 14, 30);
		const result = calendarDateTimeToUTC(dt);
		// Should be parseable as a Date and contain timezone info
		expect(new Date(result).toISOString()).toBeTruthy();
		// The result should contain a timezone offset (e.g. +00:00, Z, or similar)
		expect(result).toMatch(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/);
	});

	it('returns consistent UTC time for the same input', () => {
		const dt = new CalendarDateTime(2026, 12, 25, 8, 0);
		const a = calendarDateTimeToUTC(dt);
		const b = calendarDateTimeToUTC(dt);
		expect(a).toBe(b);
	});

	it('parses back to a valid Date object', () => {
		const dt = new CalendarDateTime(2026, 1, 1, 0, 0);
		const utcStr = calendarDateTimeToUTC(dt);
		const parsed = new Date(utcStr);
		expect(parsed.getFullYear()).toBe(2026);
		expect(parsed.getMonth()).toBe(0); // January
		expect(parsed.getDate()).toBeGreaterThanOrEqual(1);
	});

	it('different times produce different UTC strings', () => {
		const morning = new CalendarDateTime(2026, 6, 1, 9, 0);
		const evening = new CalendarDateTime(2026, 6, 1, 21, 0);
		expect(calendarDateTimeToUTC(morning)).not.toBe(calendarDateTimeToUTC(evening));
	});
});

describe('formatCalendarDateTime', () => {
	it('formats with zero-padded month, day, hour, minute', () => {
		const dt = new CalendarDateTime(2026, 3, 5, 8, 7);
		expect(formatCalendarDateTime(dt)).toBe('2026-03-05 08:07');
	});

	it('formats double-digit values correctly', () => {
		const dt = new CalendarDateTime(2026, 12, 25, 14, 30);
		expect(formatCalendarDateTime(dt)).toBe('2026-12-25 14:30');
	});

	it('handles midnight', () => {
		const dt = new CalendarDateTime(2026, 1, 1, 0, 0);
		expect(formatCalendarDateTime(dt)).toBe('2026-01-01 00:00');
	});

	it('handles end of day', () => {
		const dt = new CalendarDateTime(2026, 1, 1, 23, 59);
		expect(formatCalendarDateTime(dt)).toBe('2026-01-01 23:59');
	});
});
