import {Factory} from '../support/factory'

export class PomodoroSessionFactory extends Factory {
	static table = 'pomodoro_sessions'

	static factory() {
		const now = new Date()

		return {
			id: '{increment}',
			user_id: 1,
			task_id: 0,
			phase: 'focus',
			planned_seconds: 1500,
			// Finished by default, so a seeded row counts towards the statistics
			// instead of looking like a timer the user left running.
			started_at: new Date(now.getTime() - 25 * 60 * 1000).toISOString(),
			ended_at: now.toISOString(),
			paused_at: null,
			paused_seconds: 0,
			interrupted: false,
			log_time_entry: false,
			created: now.toISOString(),
			updated: now.toISOString(),
		}
	}
}
