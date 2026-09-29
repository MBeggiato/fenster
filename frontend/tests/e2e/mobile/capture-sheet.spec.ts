import dayjs from 'dayjs'

import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {createDefaultViews} from '../project/prepareProjects'

const MOBILE_VIEWPORT = {width: 390, height: 844}

test.describe('Mobile capture sheet', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test('Creates a task via quick add magic from the capture sheet', async ({authenticatedPage: page}) => {
		const [project] = await ProjectFactory.create(1)
		await createDefaultViews(project.id)

		const tomorrow = dayjs().add(1, 'day').format('YYYY-MM-DD')

		// Land on the project's list view first so the capture sheet's quick add
		// (which falls back to the current route's projectId) has somewhere to file the task.
		await page.goto(`/projects/${project.id}/1`)

		await page.locator('.mobile-tab-bar__capture').click()
		const panel = page.locator('.bottom-sheet__panel')
		await expect(panel).toBeVisible()

		const input = panel.locator('.add-task-textarea')
		await expect(input).toBeVisible()
		await input.fill('Buy stamps tomorrow')

		const createTaskPromise = page.waitForResponse(response =>
			response.url().includes('/api/v2/projects/') &&
			response.url().includes('/tasks/bulk') &&
			response.request().method() === 'POST',
		)
		await input.press('Enter')
		const createResponse = await createTaskPromise
		expect(createResponse.ok()).toBe(true)
		const {tasks} = await createResponse.json()
		expect(tasks).toHaveLength(1)

		// Quick add magic strips the "tomorrow" token from the title and sets the due date.
		expect(tasks[0].title).toBe('Buy stamps')
		expect(dayjs(tasks[0].due_date).format('YYYY-MM-DD')).toBe(tomorrow)

		await expect(page.locator('.global-notification')).toContainText('The task was successfully created.')
		// The sheet closes itself once the task is created.
		await expect(page.locator('.bottom-sheet')).toHaveCount(0)
	})
})
