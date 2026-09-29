import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {createDefaultViews} from '../project/prepareProjects'
import {PRIORITIES} from '../../../src/constants/priorities'

const MOBILE_VIEWPORT = {width: 390, height: 844}

test.describe('Mobile filters sheet', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test('Applies a filter from the sheet and narrows the visible task list', async ({authenticatedPage: page}) => {
		const [project] = await ProjectFactory.create(1)
		const [listView] = await createDefaultViews(project.id)
		await TaskFactory.create(1, {
			id: 1,
			project_id: project.id,
			title: 'Urgent task',
			priority: PRIORITIES.URGENT,
		})
		await TaskFactory.create(1, {
			id: 2,
			project_id: project.id,
			title: 'Low priority task',
			priority: PRIORITIES.LOW,
		}, false)

		await page.goto(`/projects/${project.id}/${listView.id}`)

		const urgentRow = page.locator('.single-task').filter({hasText: 'Urgent task'})
		const lowRow = page.locator('.single-task').filter({hasText: 'Low priority task'})
		await expect(urgentRow).toBeVisible()
		await expect(lowRow).toBeVisible()

		await page.locator('.filter-container .base-button').filter({hasText: 'Filters'}).click()

		const panel = page.locator('.bottom-sheet__panel')
		await expect(panel).toBeVisible()
		await expect(panel.locator('.bottom-sheet__title')).toContainText('Filters')

		const filterInput = panel.locator('.filter-input .ProseMirror')
		await filterInput.click()
		await filterInput.pressSequentially('priority >= 3')

		const filteredTasksRequest = page.waitForResponse(response =>
			response.url().includes(`/projects/${project.id}/views/${listView.id}/tasks`) &&
			response.request().method() === 'GET' &&
			decodeURIComponent(response.url()).includes('priority'),
		)
		await panel.locator('.button').filter({hasText: 'Show results'}).click()
		const filteredResponse = await filteredTasksRequest
		expect(filteredResponse.ok()).toBe(true)

		await expect(page.locator('.bottom-sheet')).toHaveCount(0)
		await expect(urgentRow).toBeVisible()
		await expect(lowRow).not.toBeVisible()
	})
})
