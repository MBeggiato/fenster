import {Factory} from '../support/factory'

export class TaskEisenhowerClassificationFactory extends Factory {
	static table = 'task_eisenhower_classifications'

	static factory() {
		const now = new Date()

		return {
			id: '{increment}',
			task_id: 1,
			user_id: 1,
			urgent: false,
			important: false,
			created: now.toISOString(),
			updated: now.toISOString(),
		}
	}
}
