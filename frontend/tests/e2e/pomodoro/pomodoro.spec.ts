import {test, expect} from '../../support/fixtures'
import type {Page} from '@playwright/test'

import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {PomodoroSessionFactory} from '../../factories/pomodoro_session'
import {TaskPomodoroEstimateFactory} from '../../factories/task_pomodoro_estimate'
import {LinkShareFactory} from '../../factories/link_sharing'
import {createDefaultViews} from '../project/prepareProjects'
import {setupApiUrl} from '../../support/authenticateUser'

const badge = '[data-cy="pomodoroBadge"]'

async function seedTask(userId: number, title = 'Write release notes') {
	const [project] = await ProjectFactory.create(1, {id: 1, owner_id: userId})
	await createDefaultViews(project.id)
	const [task] = await TaskFactory.create(1, {
		id: 1,
		project_id: project.id,
		created_by_id: userId,
		title,
	})
	return {project, task}
}

// mm:ss in the badge, so a test can watch it move.
async function badgeTime(page: Page): Promise<string> {
	return (await page.locator(`${badge} .pomodoro-badge__time`).innerText()).trim()
}

test.describe('Pomodoro', () => {
	test('shows the sidebar entry and the focus page', async ({authenticatedPage: page}) => {
		await page.goto('/')
		await expect(page.locator('.menu-container').getByRole('link', {name: 'Focus'})).toBeVisible()

		await page.goto('/pomodoro')
		await expect(page.locator('[data-cy="pomodoroStartPhase"]')).toBeVisible()
		// Idle shows the configured focus length rather than a running countdown.
		await expect(page.locator('.pomodoro-ring__time')).toHaveText('25:00')
	})

	test('starts a focus session from the task detail and counts it', async ({authenticatedPage: page, currentUser}) => {
		const {task} = await seedTask(currentUser.id)

		await page.goto(`/tasks/${task.id}`)
		await page.locator('[data-cy="taskStartFocus"]').click()

		// The header badge picks the session up from the shared query cache.
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeVisible()
		await expect(page.locator(`${badge} .pomodoro-badge__dot--focus`)).toBeVisible()

		// The badge follows to another page: the timer is app-wide, not per view.
		await page.goto('/')
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeVisible()

		// The focus page shows the same session, with its task.
		await page.goto('/pomodoro')
		await expect(page.locator('.pomodoro-view__task')).toContainText('Write release notes')
		await expect(page.locator('.pomodoro-ring__phase')).toHaveText('Focus')
	})

	test('pauses and resumes without consuming the phase', async ({authenticatedPage: page, currentUser}) => {
		await seedTask(currentUser.id)

		await page.goto('/pomodoro')
		await page.locator('[data-cy="pomodoroStartPhase"]').click()
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeVisible()

		await page.locator('[data-cy="pomodoroToggle"]').click()
		await expect(page.locator('.pomodoro-badge__time--paused')).toBeVisible()

		// A paused phase holds its remainder rather than draining.
		const paused = await badgeTime(page)
		await page.waitForTimeout(3000)
		expect(await badgeTime(page)).toBe(paused)

		await page.locator('[data-cy="pomodoroToggle"]').click()
		await expect(page.locator('.pomodoro-badge__time--paused')).toBeHidden()

		// Running again, so the countdown moves.
		const resumed = await badgeTime(page)
		await expect
			.poll(() => badgeTime(page), {timeout: 5000})
			.not.toBe(resumed)
	})

	test('stopping records the session as interrupted', async ({authenticatedPage: page, currentUser, apiContext, userToken}) => {
		const {task} = await seedTask(currentUser.id)

		await page.goto(`/tasks/${task.id}`)
		await page.locator('[data-cy="taskStartFocus"]').click()
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeVisible()

		await page.locator('[data-cy="pomodoroStop"]').click()

		// Back to idle: the badge offers a start again.
		await expect(page.locator('[data-cy="pomodoroStart"]')).toBeVisible()
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeHidden()

		const response = await apiContext.get('../v2/pomodoro/sessions', {
			headers: {Authorization: `Bearer ${userToken}`},
		})
		expect(response.ok()).toBe(true)
		const body = await response.json()
		expect(body.items).toHaveLength(1)
		expect(body.items[0].interrupted).toBe(true)
		expect(body.items[0].status).toBe('finished')
		expect(body.items[0].task_id).toBe(task.id)
	})

	test('closes a session whose phase ran out while the browser was shut', async ({authenticatedPage: page, currentUser}) => {
		const {task} = await seedTask(currentUser.id)
		const startedAt = new Date(Date.now() - 60 * 60 * 1000)
		// Left running an hour ago with a 25 minute phase: the server finalizes it
		// lazily on the next request, so the page must not show a live timer.
		await PomodoroSessionFactory.create(1, {
			id: 1,
			user_id: currentUser.id,
			task_id: task.id,
			started_at: startedAt.toISOString(),
			ended_at: null,
		})

		await page.goto('/pomodoro')
		await expect(page.locator('[data-cy="pomodoroStartPhase"]')).toBeVisible()
		await expect(page.locator(`${badge} .pomodoro-badge__time`)).toBeHidden()
		// Completed, not interrupted: it ran its full course.
		await expect(page.locator('.pomodoro-view__tomato').first()).toBeVisible()
		await expect(page.locator('.pomodoro-view__tomato--interrupted')).toHaveCount(0)
	})

	test('shows the focus history and the estimate on the task detail', async ({authenticatedPage: page, currentUser}) => {
		const {task} = await seedTask(currentUser.id)
		await PomodoroSessionFactory.create(1, {id: 1, user_id: currentUser.id, task_id: task.id})
		await PomodoroSessionFactory.create(1, {
			id: 2,
			user_id: currentUser.id,
			task_id: task.id,
			interrupted: true,
		}, false)
		await TaskPomodoroEstimateFactory.create(1, {task_id: task.id, user_id: currentUser.id, estimate: 5})

		await page.goto(`/tasks/${task.id}`)
		const row = page.locator('.task-pomodoro')
		await expect(row).toBeVisible()
		await expect(row.locator('.task-pomodoro__summary')).toContainText('1 of 5')
		await expect(row.locator('.task-pomodoro__tomato--interrupted')).toHaveCount(1)
		await expect(page.locator('[data-cy="taskPomodoroEstimate"]')).toHaveValue('5')
	})

	test('edits the estimate in place', async ({authenticatedPage: page, currentUser, apiContext, userToken}) => {
		const {task} = await seedTask(currentUser.id)

		await page.goto(`/tasks/${task.id}`)
		const input = page.locator('[data-cy="taskPomodoroEstimate"]')
		await input.fill('7')
		await input.blur()

		await expect.poll(async () => {
			const response = await apiContext.get(`../v2/tasks/${task.id}?expand=pomodoro`, {
				headers: {Authorization: `Bearer ${userToken}`},
			})
			return (await response.json()).pomodoro?.estimate
		}, {timeout: 5000}).toBe(7)

		// Emptying the field clears the estimate rather than storing a zero.
		await input.fill('')
		await input.blur()
		await expect.poll(async () => {
			const response = await apiContext.get(`../v2/tasks/${task.id}?expand=pomodoro`, {
				headers: {Authorization: `Bearer ${userToken}`},
			})
			return (await response.json()).pomodoro?.estimate
		}, {timeout: 5000}).toBe(0)
	})

	test('shows today in the stats tab', async ({authenticatedPage: page, currentUser}) => {
		const {task} = await seedTask(currentUser.id)
		await PomodoroSessionFactory.create(1, {id: 1, user_id: currentUser.id, task_id: task.id})

		await page.goto('/pomodoro')
		await page.getByRole('button', {name: 'Stats'}).click()

		await expect(page.locator('.pomodoro-stats__totals')).toContainText('25m')
		await expect(page.locator('.pomodoro-stats__bars')).toBeVisible()
		await expect(page.locator('.pomodoro-stats__section').filter({hasText: 'Top tasks'}))
			.toContainText('Write release notes')
	})

	test('hides the badge for a link share', async ({page, currentUser}) => {
		const {project} = await seedTask(currentUser.id)
		const [share] = await LinkShareFactory.create(1, {
			id: 1,
			project_id: project.id,
			permission: 0,
			shared_by_id: currentUser.id,
		})

		// A share page is served from index.html, whose baked-in relative API_URL
		// would never reach the API on its own port.
		await setupApiUrl(page)
		await page.goto(`/share/${share.hash}/auth`)

		await expect(page.locator('h1.title')).toContainText(project.title)
		// A link share has no user to own a timer.
		await expect(page.locator(badge)).toHaveCount(0)
	})
})
