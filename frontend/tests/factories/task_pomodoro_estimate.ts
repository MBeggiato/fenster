import {Factory} from '../support/factory'

export class TaskPomodoroEstimateFactory extends Factory {
	static table = 'task_pomodoro_estimates'

	static factory() {
		const now = new Date()

		return {
			id: '{increment}',
			task_id: 1,
			user_id: 1,
			estimate: 4,
			created: now.toISOString(),
			updated: now.toISOString(),
		}
	}
}
