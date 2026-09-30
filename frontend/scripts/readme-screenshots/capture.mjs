// Node script; browser globals are used inside page.evaluate callbacks.
/* global console, document, localStorage, process, window */
// Captures the README screenshots. Env: BASE_URL, STATE_FILE, OUT_DIR.
import {readFileSync} from 'node:fs'
import {chromium, devices} from '@playwright/test'

const {BASE_URL: base, OUT_DIR: out} = process.env
const s = JSON.parse(readFileSync(process.env.STATE_FILE, 'utf8'))

const {defaultBrowserType, ...iphone} = devices['iPhone 13']
const desktop = {viewport: {width: 1440, height: 900}, deviceScaleFactor: 2}
const mobile = {...iphone, deviceScaleFactor: 2}

const browser = await chromium.launch()

async function shoot(name, options, path, {colorScheme = 'light', notch = false, before} = {}) {
	const context = await browser.newContext({...options, colorScheme, serviceWorkers: 'block', reducedMotion: 'reduce'})
	await context.addInitScript(token => {
		localStorage.token = token
		localStorage.hideAddToHomeScreenMessage = 'true'
		localStorage.setItem('language', 'en')
	}, s.token)
	const page = await context.newPage()
	if (notch) {
		const cdp = await context.newCDPSession(page)
		await cdp.send('Emulation.setSafeAreaInsetsOverride', {insets: {top: 47, bottom: 34, left: 0, right: 0}})
	}
	await page.goto(base + path, {waitUntil: 'networkidle'})
	await page.waitForFunction(() => document.fonts.ready.then(() => true))
	await page.waitForFunction(() => !document.querySelector('.is-loading, .loader, .skeleton, [class*="skeleton"], .spinner'))
	if (before) await before(page)
	await page.waitForTimeout(1200)
	await page.screenshot({path: `${out}/${name}.png`})
	await context.close()
	console.log('captured', name)
}

// The colour scheme follows the user setting (default "auto" = prefers-color-scheme).
await shoot('desktop-home', desktop, '/')
await shoot('desktop-list', desktop, `/projects/${s.webId}/${s.listViewId}`)
// Collapse the sidebar so all four columns fit.
await shoot('desktop-kanban', desktop, `/projects/${s.webId}/${s.kanbanViewId}`, {before: p => p.click('.menu-show-button')})
await shoot('desktop-task', desktop, `/tasks/${s.taskId}`)
await shoot('desktop-dark', desktop, `/projects/${s.webId}/${s.listViewId}`, {colorScheme: 'dark'})

const m = {notch: true}
await shoot('mobile-home', mobile, '/', m)
await shoot('mobile-upcoming', mobile, '/tasks/by/upcoming', m)
// The filters come first on phones; scroll so the quadrant grid fills the shot.
await shoot('mobile-eisenhower', mobile, '/tasks/by/eisenhower', {...m, before: p => p.evaluate(() => {
	const tile = document.querySelector('.eisenhower-tile')
	if (tile) window.scrollBy(0, tile.getBoundingClientRect().top - 120)
})})
await shoot('mobile-task', mobile, `/tasks/${s.mobileTaskId}`, m)
await shoot('mobile-focus', mobile, '/pomodoro', m)

await browser.close()
