import { type CalendarDateTime, toZoned, getLocalTimeZone } from '@internationalized/date';

/**
 * Convert a CalendarDateTime (interpreted as the user's local timezone)
 * to a UTC ISO 8601 string for the API.
 */
export function calendarDateTimeToUTC(dt: CalendarDateTime): string {
	return toZoned(dt, getLocalTimeZone()).toAbsoluteString();
}

/**
 * Format a CalendarDateTime for display (local time, no timezone conversion).
 */
export function formatCalendarDateTime(dt: CalendarDateTime): string {
	const pad = (n: number) => String(n).padStart(2, '0');
	return `${dt.year}-${pad(dt.month)}-${pad(dt.day)} ${pad(dt.hour)}:${pad(dt.minute)}`;
}
