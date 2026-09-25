import {computed, ref, toValue, watch, type MaybeRefOrGetter} from 'vue'
import {useQueries} from '@tanstack/vue-query'
import {eisenhowerTasksQuery, type EisenhowerParams, type EisenhowerQuadrant} from '@/client/queries/eisenhower'
import type {TaskResponse} from '@/client/queries/tasks'

// One area of the matrix, loaded page by page: each page is its own query, and
// loadMore adds the next one.
export function useEisenhowerQuadrant(
	quadrant: MaybeRefOrGetter<EisenhowerQuadrant>,
	params: MaybeRefOrGetter<EisenhowerParams>,
	enabled: MaybeRefOrGetter<boolean> = true,
) {
	const pageCount = ref(1)
	watch(() => [toValue(quadrant), toValue(params)], () => { pageCount.value = 1 }, {deep: true})

	const pages = useQueries({
		queries: computed(() => Array.from({length: pageCount.value}, (_, index) => ({
			...eisenhowerTasksQuery(toValue(quadrant), toValue(params), index + 1),
			enabled: toValue(enabled),
		}))),
	})

	const tasks = computed<TaskResponse[]>(() => {
		const seen = new Set<number>()
		// An optimistic insert on page one can repeat a task a later page still holds.
		return pages.value.flatMap(page => page.data?.items ?? []).filter(task => {
			if (seen.has(task.id)) return false
			seen.add(task.id)
			return true
		})
	})
	const total = computed(() => pages.value[0]?.data?.total ?? 0)
	const lastPage = computed(() => pages.value[pages.value.length - 1]?.data)
	const hasMore = computed(() => lastPage.value !== undefined && lastPage.value.page < lastPage.value.total_pages)
	const isLoading = computed(() => pages.value.some(page => page.isPending && page.fetchStatus === 'fetching'))
	const isFetching = computed(() => pages.value.some(page => page.isFetching))

	function loadMore() {
		if (hasMore.value) pageCount.value++
	}

	return {tasks, total, hasMore, isLoading, isFetching, loadMore}
}
