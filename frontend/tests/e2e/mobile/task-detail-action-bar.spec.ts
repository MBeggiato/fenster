import dayjs from 'dayjs'

import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {createDefaultViews} from '../project/prepareProjects'
import {PRIORITIES} from '../../../src/constants/priorities'

const MOBILE_VIEWPORT = {width: 390, height: 844}
const TASK_ID = 1

test.describe('Mobile task detail action bar', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test('Sets a due date and a priority through the sticky action bar', async ({authenticatedPage: page, apiContext, userToken}) => {
		const headers = {Authorization: `Bearer ${userToken}`}
		const [project] = await ProjectFactory.create(1)
		await createDefaultViews(project.id)
		await TaskFactory.create(1, {
			id: TASK_ID,
			project_id: project.id,
			done: false,
		})

		await page.goto(`/tasks/${TASK_ID}`)

		const actionBar = page.locator('.mobile-task-action-bar')
		await expect(actionBar).toBeVisible()

		// --- Due date: the action bar's "due" button reveals the (usually hidden) due
		// date field and opens its datepicker as a sheet, same as MobileTaskActionBar.vue's @due handler. ---
		const tomorrow = dayjs().add(1, 'day').format('YYYY-MM-DD')
		await actionBar.locator('.mobile-task-action-bar__btn', {hasText: 'Due Date'}).click()

		const dueSheet = page.locator('.bottom-sheet__panel')
		await expect(dueSheet).toBeVisible()
		await dueSheet.locator(`.calendar-month__day[data-date="${tomorrow}"]`).click()

		const dueDateSaved = page.waitForResponse(response =>
			response.url().endsWith(`/tasks/${TASK_ID}`) && response.request().method() === 'PATCH',
		)
		// Closing the sheet (native <dialog> cancel on Escape, same as Modal.vue's @cancel) flushes
		// the pending change through Datepicker's closeOnChange -> saveTask().
		await page.keyboard.press('Escape')
		const dueDateResponse = await dueDateSaved
		expect(dueDateResponse.ok()).toBe(true)
		await expect(page.locator('.bottom-sheet')).toHaveCount(0)
		await expect(page.locator('.global-notification')).toContainText('The task was saved successfully.')

		// --- Priority: the action bar's "priority" button opens a dedicated sheet with PrioritySelect. ---
		await actionBar.locator('.mobile-task-action-bar__btn', {hasText: 'Priority'}).click()

		const prioritySheet = page.locator('.bottom-sheet__panel')
		await expect(prioritySheet).toBeVisible()
		await expect(prioritySheet.locator('.bottom-sheet__title')).toContainText('Priority')

		const prioritySaved = page.waitForResponse(response =>
			response.url().endsWith(`/tasks/${TASK_ID}`) && response.request().method() === 'PATCH',
		)
		await prioritySheet.locator('select').selectOption({label: 'High'})
		const priorityResponse = await prioritySaved
		expect(priorityResponse.ok()).toBe(true)

		await page.keyboard.press('Escape')
		await expect(page.locator('.bottom-sheet')).toHaveCount(0)

		// Persistence: verify via the API and again after a full reload.
		const stored = await apiContext.get(`tasks/${TASK_ID}`, {headers})
		expect(stored.ok()).toBe(true)
		const storedTask = await stored.json()
		expect(dayjs(storedTask.due_date).format('YYYY-MM-DD')).toBe(tomorrow)
		expect(storedTask.priority).toBe(PRIORITIES.HIGH)

		await page.reload()
		await expect(actionBar).toBeVisible()
		await actionBar.locator('.mobile-task-action-bar__btn', {hasText: 'Priority'}).click()
		await expect(page.locator('.bottom-sheet__panel select')).toHaveValue(String(PRIORITIES.HIGH))
	})
})
