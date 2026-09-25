import {beforeEach, describe, expect, it, vi} from 'vitest'
import {QueryClient} from '@tanstack/vue-query'
import {normalizeTask, taskKeys, type EisenhowerParams, type PaginatedTaskResponse} from './tasks'
import {classifyTaskMutationOptions, quadrantFor, type ClassifyInput} from './eisenhower'
import {deleteTaskMutationOptions} from './taskMutations'

const sdk = vi.hoisted(() => ({
	taskEisenhowerUpdate: vi.fn(),
	taskEisenhowerDelete: vi.fn(),
	tasksDelete: vi.fn(),
}))
vi.mock('@/client/generated', () => sdk)
vi.mock('@/message', () => ({
	error: vi.fn(),
	success: vi.fn(),
	translatedError: (message: string) => new Error(message),
}))
vi.mock('@/client/requestContext', () => ({
	captureClientRequestContext: () => 1,
	assertClientRequestContext: vi.fn(),
	isClientRequestContextCurrent: vi.fn(() => true),
}))

const params: EisenhowerParams = {filter: '', include_done: false}
const otherParams: EisenhowerParams = {filter: 'project in 2', include_done: false}

function page(ids: number[], total: number, pageNumber = 1): PaginatedTaskResponse {
	return {
		items: ids.map(id => normalizeTask({id, title: `task ${id}`, project_id: 1})),
		total,
		page: pageNumber,
		per_page: 2,
		total_pages: Math.ceil(total / 2),
	}
}

function ids(client: QueryClient, key: readonly unknown[]) {
	return client.getQueryData<PaginatedTaskResponse>(key)?.items.map(task => task.id)
}

function seed() {
	const client = new QueryClient()
	client.setQueryData(taskKeys.eisenhowerList('do', params, 1), page([1, 2], 3))
	client.setQueryData(taskKeys.eisenhowerList('do', params, 2), page([3], 3, 2))
	client.setQueryData(taskKeys.eisenhowerList('schedule', params, 1), page([4], 1))
	// Same quadrant, different filters: the task never showed there, so it must not appear.
	client.setQueryData(taskKeys.eisenhowerList('schedule', otherParams, 1), page([5], 1))
	client.setQueryData(taskKeys.eisenhowerList('unclassified', params, 1), page([6], 1))
	return client
}

function classify(client: QueryClient, input: ClassifyInput) {
	return client.getMutationCache().build(client, classifyTaskMutationOptions()).execute(input)
}

describe('eisenhower cache', () => {
	beforeEach(() => vi.resetAllMocks())

	it('maps flags to quadrants', () => {
		expect(quadrantFor(null)).toBe('unclassified')
		expect(quadrantFor({urgent: true, important: true})).toBe('do')
		expect(quadrantFor({urgent: false, important: true})).toBe('schedule')
		expect(quadrantFor({urgent: true, important: false})).toBe('delegate')
		expect(quadrantFor({urgent: false, important: false})).toBe('eliminate')
	})

	it('moves a card between quadrants before the request settles', async () => {
		const client = seed()
		const task = client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 1))!.items[0]
		sdk.taskEisenhowerUpdate.mockImplementation(async () => {
			expect(ids(client, taskKeys.eisenhowerList('do', params, 1))).toEqual([2])
			expect(ids(client, taskKeys.eisenhowerList('schedule', params, 1))).toEqual([1, 4])
			return {data: {task_id: 1, urgent: false, important: true, classified: true}}
		})

		await classify(client, {task, classification: {urgent: false, important: true}})

		expect(sdk.taskEisenhowerUpdate).toHaveBeenCalledWith({path: {task: 1}, body: {urgent: false, important: true}})
		const source = client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 1))!
		const sourcePage2 = client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 2))!
		const target = client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('schedule', params, 1))!
		expect([source.total, source.total_pages]).toEqual([2, 1])
		expect(sourcePage2.total).toBe(2)
		expect([target.total, target.items[0].eisenhower?.important]).toEqual([2, true])
		expect(ids(client, taskKeys.eisenhowerList('schedule', otherParams, 1))).toEqual([5])
	})

	it('restores every list when the request fails', async () => {
		const client = seed()
		const task = client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 1))!.items[0]
		sdk.taskEisenhowerUpdate.mockRejectedValue(new Error('denied'))

		await expect(classify(client, {task, classification: {urgent: true, important: false}})).rejects.toThrow('denied')

		expect(ids(client, taskKeys.eisenhowerList('do', params, 1))).toEqual([1, 2])
		expect(client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 1))!.total).toBe(3)
	})

	it('resets a card to unclassified and clears its classification everywhere', async () => {
		const client = seed()
		const task = normalizeTask({
			id: 4,
			title: 'task 4',
			project_id: 1,
			eisenhower: {task_id: 4, urgent: false, important: true, classified: true},
		})
		client.setQueryData(taskKeys.detail(4, ['eisenhower']), task)
		sdk.taskEisenhowerDelete.mockResolvedValue({data: undefined})

		await classify(client, {task, classification: null})

		expect(sdk.taskEisenhowerDelete).toHaveBeenCalledWith({path: {task: 4}})
		expect(ids(client, taskKeys.eisenhowerList('schedule', params, 1))).toEqual([])
		expect(ids(client, taskKeys.eisenhowerList('unclassified', params, 1))).toEqual([4, 6])
		expect(client.getQueryData<ReturnType<typeof normalizeTask>>(taskKeys.detail(4, ['eisenhower']))!.eisenhower)
			.toBeUndefined()
	})

	it('drops a deleted task from the matrix and counts it out of every page', async () => {
		const client = seed()
		sdk.tasksDelete.mockResolvedValue({})

		await client.getMutationCache().build(client, deleteTaskMutationOptions()).execute(3)

		expect(ids(client, taskKeys.eisenhowerList('do', params, 2))).toEqual([])
		expect(client.getQueryData<PaginatedTaskResponse>(taskKeys.eisenhowerList('do', params, 1))!.total).toBe(2)
	})
})
