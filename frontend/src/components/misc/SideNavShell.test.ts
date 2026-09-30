import {describe, it, expect, vi, afterEach} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createRouter, createMemoryHistory} from 'vue-router'

import SideNavShell from './SideNavShell.vue'

const stub = {template: '<div/>'}

async function setup(path: string, exact = false) {
	const router = createRouter({
		history: createMemoryHistory(),
		routes: [
			{path: '/general', name: 'general', component: stub},
			{path: '/tokens', name: 'tokens', component: stub, children: [{path: 'new', name: 'tokens.new', component: stub}]},
		],
	})
	await router.push(path)
	const wrapper = mount(SideNavShell, {
		props: {
			exact,
			navigationItems: [{title: 'General', routeName: 'general'}, {title: 'Tokens', routeName: 'tokens'}],
			extraLinks: [{url: 'https://example.com', text: 'Help'}],
		},
		global: {plugins: [router], mocks: {$t: (k: string) => k}, stubs: {Icon: true}},
	})
	return {router, wrapper, select: wrapper.find('select')}
}

describe('SideNavShell mobile picker', () => {
	afterEach(() => vi.restoreAllMocks())

	it('selects the current page, including child routes', async () => {
		const {select} = await setup('/tokens/new')
		expect((select.element as HTMLSelectElement).value).toBe('tokens')
	})

	it('shows no page as selected for a child route when exact', async () => {
		const {select} = await setup('/tokens/new', true)
		expect((select.element as HTMLSelectElement).value).toBe('')
	})

	it('navigates to the picked page', async () => {
		const {router, select} = await setup('/general')
		await select.setValue('tokens')
		await flushPromises()
		expect(router.currentRoute.value.name).toBe('tokens')
	})

	it('opens extra links in a new tab and stays on the page', async () => {
		const open = vi.spyOn(window, 'open').mockImplementation(() => null)
		const {router, select} = await setup('/general')
		await select.setValue('extra-0')
		await flushPromises()
		expect(open).toHaveBeenCalledWith('https://example.com', '_blank', 'noopener,noreferrer')
		expect(router.currentRoute.value.name).toBe('general')
		expect((select.element as HTMLSelectElement).value).toBe('general')
	})
})
