import {test, expect} from '../../support/fixtures'

const iPhone8 = {width: 375, height: 667}

test.describe('The Menu', () => {
	test.beforeEach(async ({authenticatedPage: page}) => {
		await page.goto('/')
	})

	test('Is visible by default on desktop', async ({authenticatedPage: page}) => {
		await expect(page.locator('.menu-container')).toHaveClass(/is-active/)
	})

	test('Can be hidden on desktop', async ({authenticatedPage: page}) => {
		await page.locator('button.menu-show-button:visible').click()
		await expect(page.locator('.menu-container')).not.toHaveClass(/is-active/)
	})

	test('Can be toggled with keyboard shortcut on desktop', async ({authenticatedPage: page}) => {
		await expect(page.locator('.menu-container')).toHaveClass(/is-active/)
		await page.locator('body').click()

		await page.locator('body').press('ControlOrMeta+e')
		await expect(page.locator('.menu-container')).not.toHaveClass(/is-active/)

		await page.locator('body').press('ControlOrMeta+e')
		await expect(page.locator('.menu-container')).toHaveClass(/is-active/)
	})

	test('Has no drawer on mobile, the tab bar replaces it', async ({authenticatedPage: page}) => {
		await page.setViewportSize(iPhone8)
		await expect(page.locator('.menu-container')).toHaveCount(0)
		await expect(page.locator('.mobile-tab-bar')).toBeVisible()
	})

	test('Tab bar navigates between destinations on mobile', async ({authenticatedPage: page}) => {
		await page.setViewportSize(iPhone8)
		await page.locator('.mobile-tab-bar__item', {hasText: 'Projects'}).click()
		await expect(page).toHaveURL(/\/projects$/)
	})
})
