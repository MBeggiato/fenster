import {describe, it, expect, beforeEach, vi} from 'vitest'
import {setActivePinia, createPinia} from 'pinia'

import {useAuthStore} from './auth'
import {AUTH_TYPES} from '@/modelTypes/IUser'

const {httpGetMock, getTokenMock, fakeHttp} = vi.hoisted(() => {
	const httpGetMock = vi.fn()
	return {
		httpGetMock,
		getTokenMock: vi.fn(() => null as string | null),
		fakeHttp: () => ({
			post: vi.fn(),
			get: httpGetMock,
			interceptors: {request: {use: vi.fn()}, response: {use: vi.fn()}},
		}),
	}
})

vi.mock('@/helpers/auth', () => ({
	refreshToken: vi.fn(),
	getToken: getTokenMock,
	saveToken: vi.fn(),
	removeToken: vi.fn(),
}))

vi.mock('@/router', () => ({default: {push: vi.fn()}}))
vi.mock('@/client/queryClient', () => ({queryClient: {clear: vi.fn()}}))
vi.mock('@/composables/useWebSocket', () => ({
	useWebSocket: () => ({disconnect: vi.fn(), connect: vi.fn(), closeStaleConnection: vi.fn()}),
}))
vi.mock('@/helpers/fetcher', () => ({
	HTTPFactory: () => fakeHttp(),
	AuthenticatedHTTPFactory: () => fakeHttp(),
	getApiBaseUrl: () => 'http://localhost/api/v1/',
}))
vi.mock('@/helpers/redirectToProvider', () => ({
	getRedirectUrlFromCurrentFrontendPath: vi.fn(),
	redirectToProvider: vi.fn(),
	redirectToProviderOnLogout: vi.fn(),
}))

function userJwt(exp = Math.floor(Date.now() / 1000) + 3600) {
	return `h.${btoa(JSON.stringify({id: 1, type: AUTH_TYPES.USER, exp}))}.s`
}

const tick = () => new Promise(resolve => setTimeout(resolve, 0))

describe('auth store checkAuth', () => {
	beforeEach(() => {
		setActivePinia(createPinia())
		httpGetMock.mockReset()
		getTokenMock.mockReset().mockReturnValue(userJwt())
	})

	it('does not wait for the user refresh when already authenticated with a valid JWT', async () => {
		const store = useAuthStore()
		store.setAuthenticated(true)
		store.setUser({id: 1, type: AUTH_TYPES.USER, exp: 1} as never, false)
		httpGetMock.mockReturnValue(new Promise(() => undefined))

		await expect(store.checkAuth()).resolves.toBe(true)
		expect(httpGetMock).toHaveBeenCalledTimes(1)

		// A second call while the refresh is in flight must not start another one.
		await store.checkAuth()
		expect(httpGetMock).toHaveBeenCalledTimes(1)
	})

	it('still awaits the user refresh when not yet authenticated', async () => {
		const store = useAuthStore()
		let resolveGet!: (v: unknown) => void
		httpGetMock.mockReturnValue(new Promise(resolve => {
			resolveGet = resolve
		}))

		let done = false
		const p = store.checkAuth().then(() => {
			done = true
		})
		await tick()
		expect(done).toBe(false)

		resolveGet({data: {id: 1, settings: {}}})
		await p
		expect(store.authenticated).toBe(true)
	})
})
