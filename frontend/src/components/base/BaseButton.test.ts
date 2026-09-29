import {describe, it, expect} from 'vitest'
import {mount, flushPromises} from '@vue/test-utils'
import {createRouter, createMemoryHistory} from 'vue-router'

import BaseButton from './BaseButton.vue'

const router = createRouter({
	history: createMemoryHistory(),
	routes: [{path: '/', component: {template: '<div/>'}}, {path: '/labels', name: 'labels', component: {template: '<div/>'}}],
})

describe('BaseButton', () => {
	it('emits click for router links, so menus can close when an item navigates', async () => {
		const wrapper = mount(BaseButton, {props: {to: {name: 'labels'}}, global: {plugins: [router]}})
		await wrapper.find('a').trigger('click')
		await flushPromises()

		expect(wrapper.emitted('click')).toHaveLength(1)
		expect(router.currentRoute.value.name).toBe('labels')
	})

	it('emits click for plain links', async () => {
		const wrapper = mount(BaseButton, {props: {href: 'https://example.com'}})
		await wrapper.find('a').trigger('click')

		expect(wrapper.emitted('click')).toHaveLength(1)
	})

	it('swallows clicks on aria-disabled links', async () => {
		const wrapper = mount(BaseButton, {props: {to: {name: 'labels'}, ariaDisabled: true}, global: {plugins: [router]}})
		await wrapper.find('a').trigger('click')

		expect(wrapper.emitted('click')).toBeUndefined()
	})
})
