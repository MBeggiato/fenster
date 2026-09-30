// Node script.
/* global console, fetch, process */
// Fills a fresh Fenster server with demo data through the HTTP API.
// Env: BASE_URL, STATE_FILE (where token + ids are written for capture.mjs).
import {writeFileSync} from 'node:fs'

const BASE = process.env.BASE_URL
let token = ''

async function api(method, path, body) {
	const res = await fetch(`${BASE}/api/${path}`, {
		method,
		headers: {'Content-Type': 'application/json', ...(token && {Authorization: `Bearer ${token}`})},
		body: body === undefined ? undefined : JSON.stringify(body),
	})
	if (!res.ok) throw new Error(`${method} ${path}: ${res.status} ${await res.text()}`)
	return res.json().catch(() => null)
}

const day = 86400000
const at = (offsetDays, hour = 17) => {
	const d = new Date(Date.now() + offsetDays * day)
	d.setHours(hour, 0, 0, 0)
	return d.toISOString()
}
const checklist = items => '<ul data-type="taskList">' + items.map(([t, c]) =>
	`<li data-checked="${c}" data-type="taskItem"><label><input type="checkbox"${c ? ' checked' : ''}><span></span></label><div><p>${t}</p></div></li>`).join('') + '</ul>'

await api('POST', 'v1/register', {username: 'alex', email: 'alex@example.com', password: 'demo-password-1234'})
token = (await api('POST', 'v1/login', {username: 'alex', password: 'demo-password-1234'})).token
await api('POST', 'v1/user/settings/general', {name: 'Alex Morgan', language: 'en', default_project_id: 0, week_start: 1, timezone: 'Europe/Berlin', overdue_tasks_reminders_enabled: false, overdue_tasks_reminders_time: '09:00'})

const labels = {}
for (const [title, hex] of [['design', 'a855f7'], ['errand', 'f59e0b'], ['deep work', '3b82f6'], ['waiting', '9ca3af']]) {
	labels[title] = (await api('PUT', 'v1/labels', {title, hex_color: hex})).id
}

const mkProject = async (title, hex, extra = {}) => (await api('PUT', 'v1/projects', {title, hex_color: hex, ...extra}))
const web = await mkProject('Website relaunch', '3b82f6', {description: 'New marketing site and blog for the spring launch.', is_favorite: true})
const blog = await mkProject('Blog', '60a5fa', {parent_project_id: web.id})
const home = await mkProject('Home', '22c55e')
const groceries = await mkProject('Groceries', 'f97316')
const trip = await mkProject('Trip to Lisbon', 'a855f7')

const tasks = {}
async function task(project, title, {due, priority = 0, done = false, labels: ls = [], description} = {}) {
	const t = await api('PUT', `v1/projects/${project.id}/tasks`, {
		title, priority, description, due_date: due ?? null,
	})
	for (const l of ls) await api('PUT', `v1/tasks/${t.id}/labels`, {label_id: labels[l]})
	if (done) await api('POST', `v1/tasks/${t.id}`, {...t, done: true})
	tasks[title] = t.id
	return t
}

// Website relaunch
await task(web, 'Finalize homepage hero copy', {due: at(0, 15), priority: 4, labels: ['design'],
	description: '<p>Final round with Sam. The headline should be short and the sub-line should explain the value in one sentence.</p>' + checklist([['Draft three headline options', true], ['Review with Sam', true], ['Pick a winner', false], ['Hand over to development', false]])})
await task(web, 'Fix contrast issues on pricing page', {due: at(-2), priority: 3, labels: ['design']})
await task(web, 'Set up staging environment', {due: at(1, 12), priority: 2, labels: ['deep work']})
await task(web, 'Write launch announcement', {due: at(3), priority: 3, labels: ['deep work']})
await task(web, 'Wait for legal review of privacy page', {due: at(4), labels: ['waiting']})
await task(web, 'Optimize hero images', {priority: 1, labels: ['design']})
await task(web, 'Migrate blog posts to the new CMS', {due: at(6), priority: 2})
await task(web, 'Configure analytics and cookie banner', {priority: 2})
await task(web, 'Cross-browser QA pass', {due: at(8), priority: 3})
await task(web, 'Collect customer testimonials', {labels: ['waiting']})
await task(web, 'Design new logo variants', {done: true, labels: ['design']})
await task(web, 'Choose typography and colour palette', {done: true, labels: ['design']})
await task(web, 'Sitemap and information architecture', {done: true})
await task(blog, 'Outline post: how we rebuilt our site', {due: at(2), labels: ['deep work']})
await task(blog, 'Proofread "Spring release notes"', {due: at(0, 18), priority: 1})
await task(blog, 'Pick cover images for the next three posts', {labels: ['design']})

// Home
await task(home, 'Call the plumber about the kitchen tap', {due: at(-1), priority: 4, labels: ['errand']})
await task(home, 'Renew car insurance', {due: at(5), priority: 3})
await task(home, 'Water the plants', {due: at(0, 19)})
await task(home, 'Book dentist appointment', {due: at(2), labels: ['errand']})
await task(home, 'Declutter the hallway closet', {priority: 1})
await task(home, 'Pay electricity bill', {due: at(-3), priority: 2, done: true})
await task(home, 'Replace bathroom light bulbs', {done: true, labels: ['errand']})
await task(home, 'Tax return: gather receipts', {due: at(10), priority: 2, labels: ['deep work'],
	description: '<p>Everything needs to be in one folder before the end of the month.</p>' + checklist([['Employer statement', true], ['Donation receipts', false], ['Home office costs', false]])})

// Groceries
for (const [t, d] of [['Oat milk', 0], ['Sourdough bread', 0], ['Tomatoes and basil', 1], ['Olive oil', null], ['Greek yoghurt', 1], ['Coffee beans', null], ['Lemons', 0]]) {
	await task(groceries, t, {due: d === null ? undefined : at(d, 18), labels: ['errand']})
}
await task(groceries, 'Eggs', {done: true})

// Trip
await task(trip, 'Book flights to Lisbon', {due: at(-1), priority: 4, done: false})
await task(trip, 'Reserve hotel in Alfama', {due: at(3), priority: 3})
await task(trip, 'Plan the Sintra day trip', {due: at(14), labels: ['deep work'],
	description: '<p>Start early to beat the crowds.</p>' + checklist([['Check train times', true], ['Buy Pena Palace tickets', false], ['Pack comfortable shoes', false]])})
await task(trip, 'Download offline maps', {priority: 1})
await task(trip, 'Ask Jo for restaurant tips', {labels: ['waiting']})
await task(trip, 'Renew passport', {done: true, priority: 3})

// Comments and an assignee-free rich task
await api('PUT', `v1/tasks/${tasks['Finalize homepage hero copy']}/comments`, {comment: '<p>Sam liked option two. Let\'s go with it unless legal objects.</p>'})

// Kanban for Website relaunch
const views = await api('GET', `v1/projects/${web.id}/views`)
const kanban = views.find(v => v.view_kind === 'kanban')
const buckets = await api('GET', `v1/projects/${web.id}/views/${kanban.id}/buckets`)
// Default buckets are To-Do, Doing, Done (the last one is the done bucket, keep it last).
const [b0, b1, b2] = buckets
const bp = `v1/projects/${web.id}/views/${kanban.id}/buckets`
await api('POST', `${bp}/${b0.id}`, {...b0, title: 'Backlog'})
await api('POST', `${bp}/${b1.id}`, {...b1, title: 'In progress'})
await api('POST', `${bp}/${b2.id}`, {...b2, title: 'Done'})
const review = await api('PUT', bp, {title: 'Review', position: (b1.position + b2.position) / 2})
const ids = [b0.id, b1.id, review.id, b2.id]
const plan = {
	0: ['Optimize hero images', 'Collect customer testimonials', 'Configure analytics and cookie banner', 'Migrate blog posts to the new CMS', 'Cross-browser QA pass'],
	1: ['Finalize homepage hero copy', 'Set up staging environment', 'Write launch announcement'],
	2: ['Fix contrast issues on pricing page', 'Wait for legal review of privacy page'],
	3: ['Design new logo variants', 'Choose typography and colour palette', 'Sitemap and information architecture'],
}
for (const [b, titles] of Object.entries(plan)) {
	for (const t of titles) await api('POST', `v1/projects/${web.id}/views/${kanban.id}/buckets/${ids[b]}/tasks`, {task_id: tasks[t]})
}

// Eisenhower classification (v2)
for (const [t, urgent, important] of [
	['Finalize homepage hero copy', true, true], ['Fix contrast issues on pricing page', true, true],
	['Call the plumber about the kitchen tap', true, false], ['Write launch announcement', false, true],
	['Renew car insurance', false, true], ['Water the plants', true, false], ['Declutter the hallway closet', false, false],
]) await api('PUT', `v2/tasks/${tasks[t]}/eisenhower`, {urgent, important})

// A running focus session for the focus page (a finished one can't be backdated)
await api('POST', 'v2/pomodoro/sessions', {task_id: tasks['Finalize homepage hero copy'], phase: 'focus', planned_seconds: 1500})

writeFileSync(process.env.STATE_FILE, JSON.stringify({token, webId: web.id, kanbanViewId: kanban.id, listViewId: views.find(v => v.view_kind === 'list').id, taskId: tasks['Finalize homepage hero copy'], mobileTaskId: tasks['Finalize homepage hero copy']}))
console.log(`Seeded ${Object.keys(tasks).length} tasks`)
