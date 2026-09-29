import {PRIORITIES} from '@/constants/priorities'

/**
 * Returns the CSS custom property name for a priority's color.
 * - UNSET/LOW → --info (blue)
 * - MEDIUM → --warning-text (orange)
 * - HIGH/URGENT/DO_NOW → --danger-text (red)
 */
export function getPriorityColorVar(priority: number): string {
	if (priority >= PRIORITIES.HIGH) {
		return 'var(--danger-text)'
	}
	if (priority === PRIORITIES.MEDIUM) {
		return 'var(--warning-text)'
	}
	return 'var(--info)'
}
