import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {createDefaultViews} from '../project/prepareProjects'

const MOBILE_VIEWPORT = {width: 390, height: 844}

test.describe('Mobile swipe to done', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test('Swiping a task row right marks it done, and the undo toast reverts it', async ({authenticatedPage: page, apiContext, userToken}) => {
		const headers = {Authorization: `Bearer ${userToken}`}
		const [project] = await ProjectFactory.create(1)
		await createDefaultViews(project.id)
		const [task] = await TaskFactory.create(1, {
			project_id: project.id,
			title: 'Swipe me done',
			done: false,
		})

		await page.goto(`/projects/${project.id}/1`)

		const taskRow = page.locator('.single-task').filter({hasText: task.title})
		await expect(taskRow).toBeVisible()

		const box = await taskRow.boundingBox()
		if (!box) {
			throw new Error('task row has no bounding box')
		}
		const y = box.y + box.height / 2

		const donePatch = page.waitForResponse(response =>
			response.url().endsWith(`/tasks/${task.id}`) && response.request().method() === 'PATCH',
		)

		// useSwipeActions.ts commits a swipe once the drag covers >=30% of the row's width
		// (resolveSwipeCommit's default commitRatio) — dragging most of the row's width guarantees
		// that regardless of its actual measured width. usePointerSwipe (vueuse) listens for Pointer
		// Events, which Chromium synthesizes from real mouse input too, so a plain mouse drag works
		// here without needing raw touch dispatch.
		await page.mouse.move(box.x + box.width * 0.15, y)
		await page.mouse.down()
		await page.mouse.move(box.x + box.width * 0.5, y, {steps: 5})
		await page.mouse.move(box.x + box.width * 0.9, y, {steps: 5})
		await page.mouse.up()

		const doneResponse = await donePatch
		expect(doneResponse.ok()).toBe(true)

		await expect(page.locator('.global-notification')).toContainText('The task was successfully marked as done.')

		const stored = await apiContext.get(`tasks/${task.id}`, {headers})
		expect((await stored.json()).done).toBe(true)

		const undoPatch = page.waitForResponse(response =>
			response.url().endsWith(`/tasks/${task.id}`) && response.request().method() === 'PATCH',
		)
		await page.locator('.global-notification .notification-actions .button', {hasText: 'Undo'}).click()
		const undoResponse = await undoPatch
		expect(undoResponse.ok()).toBe(true)

		const storedAfterUndo = await apiContext.get(`tasks/${task.id}`, {headers})
		expect((await storedAfterUndo.json()).done).toBe(false)
	})

	test('A swipe that does not clear the commit threshold springs back without changing the task', async ({authenticatedPage: page, apiContext, userToken}) => {
		const headers = {Authorization: `Bearer ${userToken}`}
		const [project] = await ProjectFactory.create(1)
		await createDefaultViews(project.id)
		const [task] = await TaskFactory.create(1, {
			project_id: project.id,
			title: 'Stays put',
			done: false,
		})

		await page.goto(`/projects/${project.id}/1`)

		const taskRow = page.locator('.single-task').filter({hasText: task.title})
		await expect(taskRow).toBeVisible()

		const box = await taskRow.boundingBox()
		if (!box) {
			throw new Error('task row has no bounding box')
		}
		const y = box.y + box.height / 2

		// A short, slow drag (~8% of the row) stays under both the 30% commit-ratio threshold and
		// the velocity floor (resolveSwipeCommit's elapsed time runs from pointerdown to pointerup,
		// so the pause before releasing keeps velocity well under the 0.5px/ms floor too), so it
		// should spring back instead of committing.
		await page.mouse.move(box.x + box.width * 0.15, y)
		await page.mouse.down()
		await page.mouse.move(box.x + box.width * 0.23, y, {steps: 5})
		await page.waitForTimeout(500)
		await page.mouse.up()

		const stored = await apiContext.get(`tasks/${task.id}`, {headers})
		expect((await stored.json()).done).toBe(false)
		await expect(taskRow).not.toHaveClass(/is-completing/)
	})
})
