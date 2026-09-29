import {test, expect} from '../../support/fixtures'
import {ProjectFactory} from '../../factories/project'
import {TaskFactory} from '../../factories/task'
import {BucketFactory} from '../../factories/bucket'
import {TaskBucketFactory} from '../../factories/task_buckets'
import {createDefaultViews} from '../project/prepareProjects'

const MOBILE_VIEWPORT = {width: 390, height: 844}

test.describe('Mobile kanban', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test('A bucket pager chip scrolls the board to that bucket', async ({authenticatedPage: page}) => {
		const [project] = await ProjectFactory.create(1)
		const views = await createDefaultViews(project.id)
		const kanbanView = views[3]
		// 3 full-width mobile buckets overflow the 390px viewport, so the pager has something to scroll.
		const buckets = await BucketFactory.create(3, {
			project_view_id: kanbanView.id,
		})

		await page.goto(`/projects/${project.id}/${kanbanView.id}`)

		await expect(page.locator('.bucket-pager')).toBeVisible()

		const lastChip = page.locator('.bucket-pager__chip').filter({hasText: buckets[2].title})
		await lastChip.click()

		await expect(lastChip).toHaveClass(/is-active/)
		await expect(page.locator(`.bucket[data-bucket-id="${buckets[2].id}"]`)).toBeInViewport()
	})

	test('The card\'s move action moves a task to another bucket via the move sheet', async ({authenticatedPage: page}) => {
		const [project] = await ProjectFactory.create(1)
		const views = await createDefaultViews(project.id)
		const kanbanView = views[3]
		const buckets = await BucketFactory.create(2, {
			project_view_id: kanbanView.id,
		})
		const [task] = await TaskFactory.create(1, {
			project_id: project.id,
			title: 'Move me',
		})
		await TaskBucketFactory.create(1, {
			task_id: task.id,
			bucket_id: buckets[0].id,
			project_view_id: kanbanView.id,
		})

		await page.goto(`/projects/${project.id}/${kanbanView.id}`)

		const card = page.locator('.kanban .bucket .tasks .task').filter({hasText: task.title})
		await expect(card).toBeVisible()

		await card.locator('.kanban-card__move').click()

		const panel = page.locator('.bottom-sheet__panel')
		await expect(panel).toBeVisible()
		await expect(panel.locator('.bottom-sheet__title')).toContainText('Move to…')

		// taskBucketUpdate: PUT /projects/{project}/views/{view}/buckets/{bucket}/tasks
		const moveRequest = page.waitForResponse(response =>
			response.url().includes(`/buckets/${buckets[1].id}/tasks`) &&
			response.request().method() === 'PUT',
		)
		await panel.locator('.dropdown-item').filter({hasText: buckets[1].title}).click()
		const moveResponse = await moveRequest
		expect(moveResponse.ok()).toBe(true)

		await expect(page.locator('.bottom-sheet')).toHaveCount(0)
		await expect(
			page.locator(`.bucket[data-bucket-id="${buckets[1].id}"] .tasks .task`).filter({hasText: task.title}),
		).toBeVisible()
		await expect(
			page.locator(`.bucket[data-bucket-id="${buckets[0].id}"] .tasks .task`).filter({hasText: task.title}),
		).toHaveCount(0)

		await page.reload()
		await expect(
			page.locator(`.bucket[data-bucket-id="${buckets[1].id}"] .tasks .task`).filter({hasText: task.title}),
		).toBeVisible()
	})
})
