import {test, expect} from '../../support/fixtures'

const MOBILE_VIEWPORT = {width: 390, height: 844}

test.describe('Mobile tab bar and more sheet', () => {
	test.use({viewport: MOBILE_VIEWPORT, hasTouch: true, isMobile: true})

	test.beforeEach(async ({authenticatedPage: page}) => {
		await page.goto('/')
	})

	test('Navigates across the tab bar destinations', async ({authenticatedPage: page}) => {
		await expect(page.locator('.mobile-tab-bar')).toBeVisible()
		await expect(page.locator('.mobile-tab-bar__item', {hasText: 'Home'})).toHaveClass(/is-active/)

		await page.locator('.mobile-tab-bar__item', {hasText: 'Upcoming'}).click()
		await expect(page).toHaveURL(/\/tasks\/by\/upcoming$/)

		await page.locator('.mobile-tab-bar__item', {hasText: 'Projects'}).click()
		await expect(page).toHaveURL(/\/projects$/)

		await page.locator('.mobile-tab-bar__item', {hasText: 'Focus'}).click()
		await expect(page).toHaveURL(/\/pomodoro$/)

		await page.locator('.mobile-tab-bar__item', {hasText: 'Home'}).click()
		await expect(page).toHaveURL(/\/$/)
	})

	test('Opens the capture sheet via the tab bar plus button', async ({authenticatedPage: page}) => {
		await page.locator('.mobile-tab-bar__capture').click()

		const panel = page.locator('.bottom-sheet__panel')
		await expect(panel).toBeVisible()
		await expect(panel.locator('.bottom-sheet__title')).toContainText('New task')
		await expect(panel.locator('.add-task-textarea')).toBeVisible()
	})

	test('Opens the more sheet from the avatar and navigates to Labels', async ({authenticatedPage: page}) => {
		await page.locator('.mobile-header__avatar').click()

		const panel = page.locator('.bottom-sheet__panel')
		await expect(panel).toBeVisible()
		await expect(panel.locator('.bottom-sheet__title')).toContainText('More')

		await panel.locator('.dropdown-item').filter({hasText: 'Labels'}).click()

		await expect(page).toHaveURL(/\/labels$/)
		await expect(page.locator('.bottom-sheet')).toHaveCount(0)
	})
})
