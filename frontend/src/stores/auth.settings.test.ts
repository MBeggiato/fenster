import {describe, it, expect, beforeEach} from 'vitest'
import {setActivePinia, createPinia} from 'pinia'

import {useAuthStore} from './auth'
import type {IUserSettings} from '@/modelTypes/IUserSettings'
import {plannedSecondsForMinutes, formatCountdown} from '@/composables/usePomodoro'

describe('auth store loadSettings', () => {
	beforeEach(() => setActivePinia(createPinia()))

	it('fills in pomodoro defaults for users who never saved them (idle timer must not be NaN)', () => {
		const auth = useAuthStore()
		auth.setUserSettings({frontendSettings: {colorSchema: 'dark'}} as IUserSettings)

		const {pomodoroFocusMinutes, colorSchema} = auth.settings.frontendSettings
		expect(colorSchema).toBe('dark')
		expect(pomodoroFocusMinutes).toBe(25)
		expect(formatCountdown(plannedSecondsForMinutes(pomodoroFocusMinutes))).toBe('25:00')
	})

	it('keeps values the user saved', () => {
		const auth = useAuthStore()
		auth.setUserSettings({frontendSettings: {pomodoroFocusMinutes: 50}} as IUserSettings)

		expect(auth.settings.frontendSettings.pomodoroFocusMinutes).toBe(50)
	})
})
