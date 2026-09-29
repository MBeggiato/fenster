import {describe, it, expect} from 'vitest'

import {resolveSwipeAxis, resolveSwipeCommit} from './useSwipeActions'

describe('resolveSwipeAxis', () => {
	it('stays undecided below the threshold', () => {
		expect(resolveSwipeAxis(5, 3, 10)).toBeNull()
		expect(resolveSwipeAxis(0, 0, 10)).toBeNull()
	})

	it('locks horizontal once dx dominates past the threshold', () => {
		expect(resolveSwipeAxis(20, 4, 10)).toBe('horizontal')
	})

	it('locks vertical once dy dominates past the threshold', () => {
		expect(resolveSwipeAxis(4, 20, 10)).toBe('vertical')
	})

	it('prefers horizontal on an exact tie', () => {
		expect(resolveSwipeAxis(15, 15, 10)).toBe('horizontal')
	})

	it('uses the default threshold when none is given', () => {
		expect(resolveSwipeAxis(9, 0)).toBeNull()
		expect(resolveSwipeAxis(11, 0)).toBe('horizontal')
	})
})

describe('resolveSwipeCommit', () => {
	const width = 300

	it('springs back below both the ratio and velocity thresholds', () => {
		expect(resolveSwipeCommit(50, width, 1000)).toBeNull() // 16% of width, slow
	})

	it('commits right once the distance ratio is reached', () => {
		expect(resolveSwipeCommit(91, width, 1000)).toBe('right') // >30% of width
	})

	it('commits left once the distance ratio is reached', () => {
		expect(resolveSwipeCommit(-91, width, 1000)).toBe('left')
	})

	it('commits on a fast flick even under the distance ratio', () => {
		expect(resolveSwipeCommit(60, width, 50)).toBe('right') // 20% of width but 1.2px/ms
	})

	it('respects a custom commit ratio', () => {
		expect(resolveSwipeCommit(60, width, 1000, {commitRatio: 0.1})).toBe('right')
		expect(resolveSwipeCommit(20, width, 1000, {commitRatio: 0.1})).toBeNull()
	})

	it('respects a custom velocity threshold', () => {
		expect(resolveSwipeCommit(60, width, 50, {velocityThreshold: 5})).toBeNull()
	})

	it('never commits with zero width or zero offset', () => {
		expect(resolveSwipeCommit(100, 0, 1000)).toBeNull()
		expect(resolveSwipeCommit(0, width, 1000)).toBeNull()
	})
})
