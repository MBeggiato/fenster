import {hashKey, queryOptions, useMutation, type QueryClient, type QueryKey} from '@tanstack/vue-query'
import {eisenhowerTasksList, taskEisenhowerDelete, taskEisenhowerUpdate} from '@/client/generated'
import type {PaginatedTask, TaskEisenhowerClassification} from '@/client/generated'
import {contextMutationOptions} from './contextMutation'
import {
	normalizePageNumber,
	normalizeTask,
	taskKeys,
	type EisenhowerParams,
	type EisenhowerQuadrant,
	type PaginatedTaskResponse,
	type TaskResponse,
} from './tasks'
import {dropMembership, invalidateTaskMembership, mapTaskEverywhere, removeFromPagedLists, taskQueryKeys} from './taskCache'
import {totalPagesFor} from './pagination'

export type {EisenhowerParams, EisenhowerQuadrant}

export type EisenhowerFlags = {urgent: boolean, important: boolean}

// Order matches the matrix layout: top row important, left column urgent.
export const EISENHOWER_QUADRANTS = [
	{quadrant: 'do', urgent: true, important: true},
	{quadrant: 'schedule', urgent: false, important: true},
	{quadrant: 'delegate', urgent: true, important: false},
	{quadrant: 'eliminate', urgent: false, important: false},
] as const satisfies readonly ({quadrant: EisenhowerQuadrant} & EisenhowerFlags)[]

export function quadrantFor(classification: EisenhowerFlags | null | undefined): EisenhowerQuadrant {
	if (!classification) return 'unclassified'
	return EISENHOWER_QUADRANTS.find(item =>
		item.urgent === classification.urgent && item.important === classification.important,
	)!.quadrant
}

export function flagsFor(quadrant: EisenhowerQuadrant): EisenhowerFlags | null {
	const item = EISENHOWER_QUADRANTS.find(entry => entry.quadrant === quadrant)
	return item ? {urgent: item.urgent, important: item.important} : null
}

export function classificationOf(task: Pick<TaskResponse, 'eisenhower'>): EisenhowerFlags | null {
	const classification = task.eisenhower
	if (!classification?.classified) return null
	return {urgent: classification.urgent ?? false, important: classification.important ?? false}
}

function normalizeTaskPage(page: PaginatedTask): PaginatedTaskResponse {
	return {
		...page,
		items: (page.items ?? []).map(normalizeTask),
		total: page.total ?? 0,
		page: page.page ?? 1,
		per_page: page.per_page ?? 0,
		total_pages: page.total_pages ?? 0,
	}
}

export function eisenhowerTasksQuery(quadrant: EisenhowerQuadrant, params: EisenhowerParams = {}, requestedPage = 1) {
	const page = normalizePageNumber(requestedPage)
	return queryOptions({
		queryKey: taskKeys.eisenhowerList(quadrant, params, page),
		queryFn: async ({signal}) => normalizeTaskPage((await eisenhowerTasksList({
			query: {...params, quadrant, page},
			signal,
		})).data),
	})
}

export type ClassifyInput = {
	task: TaskResponse,
	// null resets the task to unclassified.
	classification: EisenhowerFlags | null,
}

function classifiedAs(taskId: number, flags: EisenhowerFlags | null): TaskEisenhowerClassification | undefined {
	return flags ? {task_id: taskId, classified: true, ...flags} : undefined
}

// Moves the card between the cached matrix lists. It only lands in lists that
// share the filters of a list it left, so a filtered matrix never gains a task
// its filters exclude; everything else waits for the refetch.
function moveBetweenQuadrants(client: QueryClient, task: TaskResponse, target: EisenhowerQuadrant) {
	const sourceScopes = new Set<string>()
	for (const [key, list] of client.getQueriesData<PaginatedTaskResponse>({queryKey: taskKeys.eisenhowerLists})) {
		if (list?.items.some(item => item.id === task.id) && taskKeys.quadrantOf(key) !== target) {
			sourceScopes.add(hashKey(key.slice(3, -1)))
		}
	}
	removeFromPagedLists(
		client,
		taskKeys.eisenhowerLists,
		(key: QueryKey) => taskKeys.quadrantOf(key) === target ? undefined : dropMembership,
		task.id,
	)

	const moved = {...task, eisenhower: classifiedAs(task.id, flagsFor(target))}
	const targetScopes = new Map<string, QueryKey[]>()
	for (const [key, list] of client.getQueriesData<PaginatedTaskResponse>({queryKey: taskKeys.eisenhowerLists})) {
		if (!list || taskKeys.quadrantOf(key) !== target) continue
		if (list.items.some(item => item.id === task.id)) continue
		const scope = hashKey(key.slice(3, -1))
		if (!sourceScopes.has(scope)) continue
		targetScopes.set(scope, [...targetScopes.get(scope) ?? [], key])
	}
	for (const keys of targetScopes.values()) {
		for (const key of keys) {
			client.setQueryData<PaginatedTaskResponse>(key, list => {
				if (!list) return list
				const total = list.total + 1
				return {
					...list,
					items: list.page === 1 ? [moved, ...list.items] : list.items,
					total,
					total_pages: totalPagesFor(list, total),
				}
			})
		}
	}
}

export function classifyTaskMutationOptions() {
	return contextMutationOptions({
		mutationFn: async ({task, classification}: ClassifyInput) => {
			if (classification === null) {
				await taskEisenhowerDelete({path: {task: task.id}})
				return undefined
			}
			return (await taskEisenhowerUpdate({path: {task: task.id}, body: classification})).data
		},
		optimistic: {
			queryKeys: ({task}, client) => [taskKeys.eisenhowerLists, ...taskQueryKeys(client, task.id)],
			update: ({task, classification}, client) => {
				moveBetweenQuadrants(client, task, quadrantFor(classification))
				mapTaskEverywhere(client, task.id, current => ({
					...current,
					eisenhower: classifiedAs(task.id, classification),
				}))
			},
		},
		onSuccess: (stored, {task}, client) => mapTaskEverywhere(client, task.id, current => ({
			...current,
			eisenhower: stored,
		})),
		// The lists were patched above, so they are only marked stale instead of reloading every page.
		onSettled: ({task}, client) => invalidateTaskMembership(client, task.id),
	})
}

export function useClassifyTaskMutation() {
	return useMutation(classifyTaskMutationOptions())
}
