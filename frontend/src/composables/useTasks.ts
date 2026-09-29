import {computed, toValue, type MaybeRefOrGetter} from 'vue'
import {useQuery, keepPreviousData} from '@tanstack/vue-query'
import {tasksQuery, type TaskScope} from '@/client/queries/tasks'

export type UseTasksOptions = {
	enabled?: MaybeRefOrGetter<boolean>
	page?: MaybeRefOrGetter<number>
	staleTime?: number
	// Show the previous result while a changed scope (filters, date range) loads, instead of emptying the list.
	keepPrevious?: boolean
}

export function useTasks(
	scope: MaybeRefOrGetter<TaskScope>,
	{enabled = true, page = 1, staleTime, keepPrevious = false}: UseTasksOptions = {},
) {
	const query = useQuery(computed(() => ({
		...tasksQuery(toValue(scope), toValue(page)),
		enabled: toValue(enabled),
		...(staleTime !== undefined && {staleTime}),
		...(keepPrevious && {placeholderData: keepPreviousData}),
	})))
	const tasks = computed(() => query.data.value?.items ?? [])
	const total = computed(() => query.data.value?.total ?? 0)
	const totalPages = computed(() => query.data.value?.total_pages ?? 0)
	return {...query, tasks, total, totalPages}
}
