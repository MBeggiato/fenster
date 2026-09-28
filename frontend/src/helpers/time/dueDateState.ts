import {MILLISECONDS_A_DAY} from '@/constants/date'

export type DueDateState = 'overdue' | 'today' | 'soon' | 'later' | null

const SOON_DAYS = 3

/**
 * Classifies a task's due date relative to `now` for reward/hierarchy styling.
 * Mirrors the "is this overdue" check already used across task rows: a due date of
 * null/unset/zero-value or a done task never gets a tier.
 */
export function dueDateState(
	dueDate: string | Date | null | undefined,
	done = false,
	now: Date = new Date(),
): DueDateState {
	if (done || !dueDate) {
		return null
	}

	const due = new Date(dueDate)
	const dueTime = due.getTime()
	if (Number.isNaN(dueTime) || dueTime <= 0) {
		return null
	}

	if (dueTime <= now.getTime()) {
		return 'overdue'
	}

	const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate())
	const startOfDue = new Date(due.getFullYear(), due.getMonth(), due.getDate())
	const dayDiff = Math.round((startOfDue.getTime() - startOfToday.getTime()) / MILLISECONDS_A_DAY)

	if (dayDiff <= 0) {
		return 'today'
	}

	return dayDiff <= SOON_DAYS ? 'soon' : 'later'
}
