import type {APIRequestContext, Page} from '@playwright/test'
import {test, expect} from '../../support/fixtures'
import {login} from '../../support/authenticateUser'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {LabelFactory} from '../../factories/labels'
import {LabelTaskFactory} from '../../factories/label_task'
import {UserFactory} from '../../factories/user'
import {UserProjectFactory} from '../../factories/users_project'
import {TaskEisenhowerClassificationFactory} from '../../factories/task_eisenhower_classification'
import {createDefaultViews} from '../project/prepareProjects'

async function seedMatrix() {
	const [project] = await ProjectFactory.create(1, {id: 1, title: 'Matrix project'})
	await createDefaultViews(project.id)
	const titles = ['Do task', 'Schedule task', 'Delegate task', 'Eliminate task', 'Unclassified task', 'Done task']
	const tasks = await TaskFactory.create(titles.length, {
		title: (i: number) => titles[i - 1],
		project_id: project.id,
		done: (i: number) => i === 6,
	})
	const flags = [
		{task_id: 1, urgent: true, important: true},
		{task_id: 2, urgent: false, important: true},
		{task_id: 3, urgent: true, important: false},
		{task_id: 4, urgent: false, important: false},
		{task_id: 6, urgent: true, important: true},
	]
	await TaskEisenhowerClassificationFactory.create(flags.length, {
		task_id: (i: number) => flags[i - 1].task_id,
		urgent: (i: number) => flags[i - 1].urgent,
		important: (i: number) => flags[i - 1].important,
	})
	return {project, tasks}
}

function area(page: Page, quadrant: string) {
	return page.locator(`section[data-quadrant="${quadrant}"]`)
}

async function storedClassification(apiContext: APIRequestContext, token: string, taskId: number) {
	const response = await apiContext.get(`/api/v2/tasks/${taskId}/eisenhower`, {
		headers: {Authorization: `Bearer ${token}`},
	})
	expect(response.ok()).toBe(true)
	return response.json()
}

async function openMatrix(page: Page, query = '') {
	const loaded = page.waitForResponse(r => r.url().includes('/api/v2/eisenhower/tasks') && r.ok())
	await page.goto(`/tasks/by/eisenhower${query}`)
	await loaded
}

test.describe('Eisenhower matrix', () => {
	test('places open tasks in their quadrant and hides done ones', async ({authenticatedPage: page}) => {
		await seedMatrix()
		await openMatrix(page)

		await expect(area(page, 'do')).toContainText('Do task')
		await expect(area(page, 'schedule')).toContainText('Schedule task')
		await expect(area(page, 'delegate')).toContainText('Delegate task')
		await expect(area(page, 'eliminate')).toContainText('Eliminate task')
		await expect(area(page, 'unclassified')).toContainText('Unclassified task')
		await expect(page.locator('.eisenhower-matrix')).not.toContainText('Done task')

		await page.getByText('Show done tasks').click()
		await expect(area(page, 'do')).toContainText('Done task')
	})

	test('is reachable from the main navigation', async ({authenticatedPage: page}) => {
		await seedMatrix()
		await page.goto('/')
		await page.locator('.menu-list').getByRole('link', {name: 'Eisenhower matrix'}).click()
		await expect(page).toHaveURL(/\/tasks\/by\/eisenhower/)
		await expect(area(page, 'do')).toContainText('Do task')
	})

	test('stores a drag between quadrants and a drag back to unclassified', async ({authenticatedPage: page, apiContext, userToken}) => {
		await seedMatrix()
		await openMatrix(page)

		const saved = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/5/eisenhower') && r.request().method() === 'PUT')
		await area(page, 'unclassified').locator('.single-task', {hasText: 'Unclassified task'})
			.dragTo(area(page, 'do').locator('ul.quadrant-tasks'))
		expect((await saved).ok()).toBe(true)
		await expect(area(page, 'do')).toContainText('Unclassified task')
		await expect(area(page, 'unclassified')).not.toContainText('Unclassified task')

		await page.reload()
		await expect(area(page, 'do')).toContainText('Unclassified task')
		expect(await storedClassification(apiContext, userToken, 5)).toMatchObject({classified: true, urgent: true, important: true})

		const reset = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/1/eisenhower') && r.request().method() === 'DELETE')
		await area(page, 'do').locator('.single-task', {hasText: 'Do task'})
			.dragTo(area(page, 'unclassified').locator('ul.quadrant-tasks'))
		expect((await reset).ok()).toBe(true)

		await page.reload()
		await expect(area(page, 'unclassified')).toContainText('Do task')
		expect(await storedClassification(apiContext, userToken, 1)).toMatchObject({classified: false})
	})

	test('creates a task directly in a quadrant', async ({authenticatedPage: page, apiContext, userToken}) => {
		const {project} = await seedMatrix()
		await openMatrix(page, `?project=${project.id}`)

		const classified = page.waitForResponse(r => /\/api\/v2\/tasks\/\d+\/eisenhower$/.test(r.url()) && r.request().method() === 'PUT')
		const input = area(page, 'schedule').getByPlaceholder('Add a task…')
		await input.fill('Planned from the matrix')
		await input.press('Enter')
		const response = await classified
		expect(response.ok()).toBe(true)
		await expect(area(page, 'schedule')).toContainText('Planned from the matrix')

		await page.reload()
		await expect(area(page, 'schedule')).toContainText('Planned from the matrix')
		const taskId = Number(new URL(response.url()).pathname.match(/tasks\/(\d+)\/eisenhower$/)?.[1])
		expect(await storedClassification(apiContext, userToken, taskId)).toMatchObject({urgent: false, important: true})
	})

	test('classifies a task from its detail view', async ({authenticatedPage: page, apiContext, userToken}) => {
		await seedMatrix()
		await page.goto('/tasks/5')

		const toggles = page.locator('.eisenhower-toggles')
		await expect(toggles).toContainText('Not in your matrix yet')
		const urgent = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/5/eisenhower') && r.request().method() === 'PUT')
		await toggles.getByRole('button', {name: 'Not urgent'}).click()
		expect((await urgent).ok()).toBe(true)
		const important = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/5/eisenhower') && r.request().method() === 'PUT')
		await toggles.getByRole('button', {name: 'Not important'}).click()
		expect((await important).ok()).toBe(true)
		await expect(toggles).toContainText('Do first')

		await page.reload()
		await expect(page.locator('.eisenhower-toggles')).toContainText('Do first')
		expect(await storedClassification(apiContext, userToken, 5)).toMatchObject({urgent: true, important: true})

		await openMatrix(page)
		await expect(area(page, 'do')).toContainText('Unclassified task')

		await page.goto('/tasks/5')
		const reset = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/5/eisenhower') && r.request().method() === 'DELETE')
		await page.locator('.eisenhower-toggles').getByRole('button', {name: 'Reset'}).click()
		expect((await reset).ok()).toBe(true)
		await page.reload()
		await expect(page.locator('.eisenhower-toggles')).toContainText('Not in your matrix yet')
	})

	test('filters by project, label and search', async ({authenticatedPage: page}) => {
		await seedMatrix()
		const [other] = await ProjectFactory.create(1, {id: 2, title: 'Other project'}, false)
		await TaskFactory.create(1, {id: 7, title: 'Other project task', project_id: other.id}, false)
		await LabelFactory.create(1, {id: 1, title: 'Focus'})
		await LabelTaskFactory.create(1, {task_id: 2, label_id: 1})

		await openMatrix(page, '?project=2')
		await expect(area(page, 'unclassified')).toContainText('Other project task')
		await expect(area(page, 'unclassified')).not.toContainText('Unclassified task')
		await expect(area(page, 'do')).not.toContainText('Do task')

		await openMatrix(page, '?labels=1')
		await expect(area(page, 'schedule')).toContainText('Schedule task')
		await expect(area(page, 'do')).not.toContainText('Do task')
		await expect(area(page, 'unclassified')).not.toContainText('Unclassified task')

		await openMatrix(page)
		const searched = page.waitForResponse(r => r.url().includes('/api/v2/eisenhower/tasks') && r.url().includes('q=Delegate'))
		await page.locator('#eisenhower-search').fill('Delegate')
		await searched
		await expect(area(page, 'delegate')).toContainText('Delegate task')
		await expect(area(page, 'do')).not.toContainText('Do task')
		await expect(page).toHaveURL(/q=Delegate/)
	})

	test('drops a task once it is done', async ({authenticatedPage: page, apiContext, userToken}) => {
		await seedMatrix()
		await openMatrix(page)

		const done = page.waitForResponse(r => r.url().endsWith('/api/v2/tasks/1') && r.request().method() === 'PATCH')
		await area(page, 'do').locator('.single-task', {hasText: 'Do task'}).locator('.fancy-checkbox').click()
		expect((await done).ok()).toBe(true)
		await expect(area(page, 'do')).not.toContainText('Do task')

		await page.reload()
		await expect(area(page, 'do')).not.toContainText('Do task')
		// The classification survives completion.
		expect(await storedClassification(apiContext, userToken, 1)).toMatchObject({classified: true, urgent: true, important: true})
	})

	test('keeps the classification private to each user', async ({authenticatedPage: page, apiContext, userToken, currentUser}) => {
		await seedMatrix()
		const [member] = await UserFactory.create(1, {id: 2}, false)
		await UserProjectFactory.create(1, {project_id: 1, user_id: member.id, permission: 0})
		expect(currentUser.id).toBe(1)

		const {token: memberToken} = await login(null, apiContext, member)
		expect(await storedClassification(apiContext, memberToken, 1)).toMatchObject({classified: false})

		const response = await apiContext.put('/api/v2/tasks/1/eisenhower', {
			headers: {Authorization: `Bearer ${memberToken}`},
			data: {urgent: false, important: false},
		})
		expect(response.ok()).toBe(true)

		await openMatrix(page)
		await expect(area(page, 'do')).toContainText('Do task')
		expect(await storedClassification(apiContext, userToken, 1)).toMatchObject({urgent: true, important: true})
	})
})
