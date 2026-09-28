import {describe, it, expect} from 'vitest'

import {dueDateState} from './dueDateState'

const NOW = new Date('2024-06-15T12:00:00')

describe('dueDateState', () => {
	it('returns null when there is no due date', () => {
		expect(dueDateState(null, false, NOW)).toBeNull()
		expect(dueDateState(undefined, false, NOW)).toBeNull()
		expect(dueDateState('0001-01-01T00:00:00Z', false, NOW)).toBeNull()
	})

	it('returns null when the task is already done', () => {
		expect(dueDateState('2024-06-01T00:00:00', true, NOW)).toBeNull()
	})

	it('returns overdue for a due date in the past', () => {
		expect(dueDateState('2024-06-15T11:00:00', false, NOW)).toBe('overdue')
	})

	it('returns today for a due date later the same day', () => {
		expect(dueDateState('2024-06-15T18:00:00', false, NOW)).toBe('today')
	})

	it('returns soon for a due date within the next 3 days', () => {
		expect(dueDateState('2024-06-16T09:00:00', false, NOW)).toBe('soon')
		expect(dueDateState('2024-06-18T09:00:00', false, NOW)).toBe('soon')
	})

	it('returns later for a due date beyond 3 days', () => {
		expect(dueDateState('2024-06-19T09:00:00', false, NOW)).toBe('later')
	})
})
