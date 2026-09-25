import {describe, expect, it} from 'vitest'

import {normalizePomodoroSession, type PomodoroSessionResponse} from '@/client/queries/pomodoro'
import {
	elapsedSecondsOf,
	focusSecondsOf,
	focusSessionsUntilLongBreak,
	formatCountdown,
	isFinishedFocus,
	markRung,
	nextPhaseFor,
	plannedSecondsForMinutes,
	remainingSecondsAt,
} from './usePomodoro'

const START = new Date('2026-09-25T10:00:00Z')

function finished(overrides: Partial<PomodoroSessionResponse> = {}): PomodoroSessionResponse {
	return normalizePomodoroSession({
		id: 1,
		phase: 'focus',
		planned_seconds: 1500,
		started_at: START.toISOString(),
		ended_at: new Date(START.getTime() + 25 * 60 * 1000).toISOString(),
		status: 'finished',
		remaining_seconds: 0,
		...overrides,
	})
}

function running(overrides: Partial<PomodoroSessionResponse> = {}): PomodoroSessionResponse {
	return normalizePomodoroSession({
		id: 1,
		phase: 'focus',
		planned_seconds: 1500,
		started_at: START.toISOString(),
		status: 'running',
		remaining_seconds: 1500,
		...overrides,
	})
}

describe('formatCountdown', () => {
	it('pads minutes and seconds, adding hours only when there are some', () => {
		expect(formatCountdown(0)).toBe('00:00')
		expect(formatCountdown(59)).toBe('00:59')
		expect(formatCountdown(1500)).toBe('25:00')
		expect(formatCountdown(3661)).toBe('1:01:01')
	})

	it('treats a negative remainder as zero', () => {
		// A clock that jumped backwards must not print "-1:-5".
		expect(formatCountdown(-5)).toBe('00:00')
	})
})

describe('remainingSecondsAt', () => {
	const now = START.getTime()

	it('counts down to the deadline', () => {
		expect(remainingSecondsAt(now + 100_000, now, 0)).toBe(100)
		expect(remainingSecondsAt(now + 100_000, now + 40_000, 0)).toBe(60)
	})

	it('never goes below zero once the deadline passed', () => {
		expect(remainingSecondsAt(now, now + 60_000, 0)).toBe(0)
	})

	it('holds the reported remainder while paused', () => {
		// No deadline means paused: a pause must not consume the phase.
		expect(remainingSecondsAt(null, now + 600_000, 420)).toBe(420)
	})
})

describe('elapsedSecondsOf', () => {
	it('is the wall time of a finished session', () => {
		expect(elapsedSecondsOf(finished())).toBe(25 * 60)
	})

	it('subtracts the pauses', () => {
		expect(elapsedSecondsOf(finished({
			ended_at: new Date(START.getTime() + 30 * 60 * 1000).toISOString(),
			paused_seconds: 300,
		}))).toBe(25 * 60)
	})

	it('is zero for a session that has not ended', () => {
		expect(elapsedSecondsOf(running())).toBe(0)
	})

	it('is zero rather than NaN for unusable timestamps', () => {
		expect(elapsedSecondsOf(normalizePomodoroSession({
			started_at: 'not a date',
			ended_at: 'not a date either',
		}))).toBe(0)
	})
})

describe('focusSecondsOf', () => {
	it('adds up the finished focus phases, interrupted ones included', () => {
		expect(focusSecondsOf([
			finished({id: 1}),
			finished({id: 2, interrupted: true}),
		])).toBe(50 * 60)
	})

	it('leaves out breaks and unfinished phases', () => {
		expect(focusSecondsOf([
			finished({id: 1}),
			finished({id: 2, phase: 'short_break'}),
			running({id: 3}),
		])).toBe(25 * 60)
	})
})

describe('isFinishedFocus', () => {
	it('is true only for a focus phase that ended', () => {
		expect(isFinishedFocus(finished())).toBe(true)
		expect(isFinishedFocus(finished({phase: 'long_break'}))).toBe(false)
		expect(isFinishedFocus(running())).toBe(false)
	})
})

describe('nextPhaseFor', () => {
	it('follows a focus phase with a short break', () => {
		expect(nextPhaseFor('focus', 1, 4)).toBe('short_break')
		expect(nextPhaseFor('focus', 3, 4)).toBe('short_break')
	})

	it('makes every fourth break a long one', () => {
		expect(nextPhaseFor('focus', 4, 4)).toBe('long_break')
		expect(nextPhaseFor('focus', 8, 4)).toBe('long_break')
	})

	it('follows a break with focus', () => {
		expect(nextPhaseFor('short_break', 4, 4)).toBe('focus')
		expect(nextPhaseFor('long_break', 4, 4)).toBe('focus')
		expect(nextPhaseFor(null, 4, 4)).toBe('focus')
	})

	it('takes a short break before the first completed phase', () => {
		expect(nextPhaseFor('focus', 0, 4)).toBe('short_break')
	})

	it('survives a long-break-every of zero', () => {
		// A hand-edited setting must not divide by zero.
		expect(nextPhaseFor('focus', 3, 0)).toBe('long_break')
	})
})

describe('focusSessionsUntilLongBreak', () => {
	it('counts down towards the long break', () => {
		expect(focusSessionsUntilLongBreak([], 4)).toBe(4)
		expect(focusSessionsUntilLongBreak([finished({id: 1})], 4)).toBe(3)
		expect(focusSessionsUntilLongBreak([
			finished({id: 1}),
			finished({id: 2}),
			finished({id: 3}),
		], 4)).toBe(1)
	})

	it('restarts after a long break was taken', () => {
		expect(focusSessionsUntilLongBreak([
			finished({id: 1}),
			finished({id: 2}),
			finished({id: 3, phase: 'long_break'}),
			finished({id: 4}),
		], 4)).toBe(3)
	})

	it('does not count an interrupted phase', () => {
		expect(focusSessionsUntilLongBreak([
			finished({id: 1}),
			finished({id: 2, interrupted: true}),
		], 4)).toBe(3)
	})
})

describe('markRung', () => {
	it('rings once per session', () => {
		const rung = new Set<number>()
		expect(markRung(rung, 7)).toBe(true)
		// A refetch that still reports the expired session must stay silent.
		expect(markRung(rung, 7)).toBe(false)
		expect(markRung(rung, 8)).toBe(true)
	})
})

describe('plannedSecondsForMinutes', () => {
	it('converts minutes to seconds', () => {
		expect(plannedSecondsForMinutes(25)).toBe(1500)
	})

	it('clamps to the bounds the API enforces', () => {
		expect(plannedSecondsForMinutes(0)).toBe(60)
		expect(plannedSecondsForMinutes(-5)).toBe(60)
		expect(plannedSecondsForMinutes(1000)).toBe(4 * 60 * 60)
	})
})
