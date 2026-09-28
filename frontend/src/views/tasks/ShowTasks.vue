<template>
	<div
		v-cy="'showTasks'"
		class="is-max-width-desktop has-text-start"
	>
		<h2 class="mbe-2 title">
			{{ pageTitle }}
		</h2>
		<Message
			v-if="filteredLabels.length > 0"
			class="label-filter-info mbe-2"
		>
			<i18n-t
				keypath="task.show.filterByLabel"
				tag="span"
				class="filter-label-text"
			>
				<template #label>
					<XLabel
						v-for="label in filteredLabels"
						:key="label.id"
						:label="label"
					/>
				</template>
			</i18n-t>
			<BaseButton
				v-tooltip="$t('task.show.clearLabelFilter')"
				class="clear-filter-button"
				:aria-label="$t('task.show.clearLabelFilter')"
				@click="clearLabelFilter"
			>
				<Icon icon="times" />
			</BaseButton>
		</Message>
		<Message
			v-if="savedFilterIgnored"
			class="mbe-2"
		>
			{{ $t('task.show.savedFilterIgnored') }}
		</Message>
		<p
			v-if="!showAll"
			class="show-tasks-options"
		>
			<DatepickerWithRange
				:model-value="{dateFrom: dateFrom ?? null, dateTo: dateTo ?? null}"
				@update:modelValue="setDate"
			>
				<template #trigger="{toggle}">
					<XButton
						variant="primary"
						:shadow="false"
						class="mbe-2"
						@click.prevent.stop="toggle()"
					>
						{{ $t('task.show.select') }}
					</XButton>
				</template>
			</DatepickerWithRange>

			<!-- Desktop keeps the checkboxes inline; mobile tucks them behind a compact sheet trigger. -->
			<template v-if="!isMobile">
				<FancyCheckbox
					:model-value="showNulls"
					class="mie-2"
					@update:modelValue="setShowNulls"
				>
					{{ $t('task.show.noDates') }}
				</FancyCheckbox>
				<FancyCheckbox
					:model-value="showOverdue"
					@update:modelValue="setShowOverdue"
				>
					{{ $t('task.show.overdue') }}
				</FancyCheckbox>
			</template>
			<Popup
				v-else
				sheet-on-mobile
				:sheet-title="$t('mobile.screens.upcoming.filters')"
			>
				<template #trigger="{toggle}">
					<XButton
						variant="secondary"
						:shadow="false"
						class="mbe-2"
						icon="filter"
						@click.prevent.stop="toggle()"
					>
						{{ $t('mobile.screens.upcoming.filters') }}
					</XButton>
				</template>
				<template #content>
					<div class="show-tasks-filters-sheet">
						<FancyCheckbox
							:model-value="showNulls"
							@update:modelValue="setShowNulls"
						>
							{{ $t('task.show.noDates') }}
						</FancyCheckbox>
						<FancyCheckbox
							:model-value="showOverdue"
							@update:modelValue="setShowOverdue"
						>
							{{ $t('task.show.overdue') }}
						</FancyCheckbox>
					</div>
				</template>
			</Popup>
		</p>
		<template v-if="!loading && (!tasks || tasks.length === 0) && showNothingToDo">
			<h3 class="has-text-centered mbs-6">
				{{ $t('task.show.noTasks') }}
			</h3>
			<LlamaCool class="llama-cool" />
		</template>

		<Card
			v-if="hasTasks && !isMobile"
			:padding="false"
			class="has-overflow"
			:has-content="false"
			:loading="loading"
		>
			<ul class="p-2 tasks">
				<li
					v-for="task in tasks"
					:key="task.id"
				>
					<SingleTaskInProject
						:show-project="true"
						:the-task="task"
						:can-mark-as-done="(projectList.projects[task.project_id]?.max_permission ?? 0) > PERMISSIONS.READ"
						@taskUpdated="updateTasks"
					/>
				</li>
			</ul>
		</Card>
		<div
			v-else-if="hasTasks"
			class="tasks-grouped loader-container"
			:class="{'is-loading': loading}"
		>
			<section
				v-for="group in groupedTasks"
				:key="group.key"
				class="tasks-grouped__day"
			>
				<h3 class="tasks-grouped__day-heading">
					{{ group.label }}
				</h3>
				<ul class="p-2 tasks">
					<li
						v-for="task in group.tasks"
						:key="task.id"
					>
						<SingleTaskInProject
							:show-project="true"
							:the-task="task"
							:can-mark-as-done="(projectList.projects[task.project_id]?.max_permission ?? 0) > PERMISSIONS.READ"
							@taskUpdated="updateTasks"
						/>
					</li>
				</ul>
			</section>
		</div>
		<div
			v-else
			:class="{ 'is-loading': loading}"
			class="spinner"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch, watchEffect} from 'vue'
import {useRoute, useRouter} from 'vue-router'
import {useI18n} from 'vue-i18n'

import {formatDate} from '@/helpers/time/formatDate'
import {setTitle} from '@/helpers/setTitle'

import BaseButton from '@/components/base/BaseButton.vue'
import Icon from '@/components/misc/Icon'
import Message from '@/components/misc/Message.vue'
import Popup from '@/components/misc/Popup.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'
import DatepickerWithRange from '@/components/date/DatepickerWithRange.vue'
import XLabel from '@/components/tasks/partials/Label.vue'
import {DATE_RANGES} from '@/components/date/dateRanges'
import LlamaCool from '@/assets/llama-cool.svg?component'
import {useAuthStore} from '@/stores/auth'
import {useProjects} from '@/composables/useProjects'
import {useLabels} from '@/composables/useLabels'
import {useIsMobile} from '@/composables/useIsMobile'
import type {TaskFilterParams, TaskResponse} from '@/client/queries/tasks'
import {useTasks} from '@/composables/useTasks'
import type {TaskScope} from '@/client/queries/tasks'
import {PERMISSIONS} from '@/constants/permissions'
import {parseDateOrNull} from '@/helpers/parseDateOrNull'
import {addDays, isSameDay} from '@/helpers/time/dateMath'

const props = withDefaults(defineProps<{
	dateFrom?: Date | string,
	dateTo?: Date | string,
	showNulls?: boolean,
	showOverdue?: boolean,
	labelIds?: string[],
}>(), {
	showNulls: false,
	showOverdue: false,
	dateFrom: undefined,
	dateTo: undefined,
	labelIds: undefined,
})

const emit = defineEmits<{
	'tasksLoaded': true,
	'clearLabelFilter': void,
}>()

const authStore = useAuthStore()
const projectList = useProjects()
const {getLabelById} = useLabels()
const isMobile = useIsMobile()

const route = useRoute()
const router = useRouter()
const {t} = useI18n({useScope: 'global'})

const taskScope = ref<TaskScope | null>(null)
const taskQuery = useTasks(
	() => taskScope.value ?? {},
	{enabled: () => authStore.authenticated && taskScope.value !== null},
)
const tasks = taskQuery.tasks
const showNothingToDo = ref<boolean>(false)


setTimeout(() => showNothingToDo.value = true, 100)

const showAll = computed(() => typeof props.dateFrom === 'undefined' || typeof props.dateTo === 'undefined')

const filteredLabels = computed(() => {
	if (!props.labelIds || props.labelIds.length === 0) {
		return []
	}
	return props.labelIds
		.map(id => getLabelById(Number(id)))
		.filter(label => label !== null && label !== undefined)
})

const savedFilterIgnored = computed(() => {
	return filteredLabels.value.length > 0
		&& filterIdUsedOnOverview.value
		&& typeof projectList.projects[filterIdUsedOnOverview.value] !== 'undefined'
})

const pageTitle = computed(() => {
	// We need to define "key" because it is the first parameter in the array and we need the second
	const predefinedRange = Object.entries(DATE_RANGES)
		.find(([, value]) => props.dateFrom === value[0] && props.dateTo === value[1])
		?.[0]
	if (typeof predefinedRange !== 'undefined') {
		return t(`input.datepickerRange.ranges.${predefinedRange}`)
	}

	return showAll.value
		? t('task.show.titleCurrent')
		: t('task.show.fromuntil', {
			from: formatDate(props.dateFrom, 'LL'),
			until: formatDate(props.dateTo, 'LL'),
		})
})
const hasTasks = computed(() => tasks.value && tasks.value.length > 0)

interface TaskGroup {
	key: string
	label: string
	tasks: TaskResponse[]
}

// Grouped client-side from the already-fetched (due_date-sorted) list, no extra queries.
const groupedTasks = computed<TaskGroup[]>(() => {
	const now = new Date()
	const groups = new Map<string, TaskGroup>()
	for (const task of tasks.value ?? []) {
		const due = parseDateOrNull(task.due_date)
		const key = due ? due.toDateString() : 'none'
		let group = groups.get(key)
		if (!group) {
			let label: string
			if (!due) {
				label = t('mobile.screens.upcoming.noDueDate')
			} else if (isSameDay(due, now)) {
				label = t('input.datepicker.today')
			} else if (isSameDay(due, addDays(now, 1))) {
				label = t('input.datepicker.tomorrow')
			} else {
				label = formatDate(due, 'dddd, MMM D')
			}
			group = {key, label, tasks: []}
			groups.set(key, group)
		}
		group.tasks.push(task)
	}
	return [...groups.values()]
})
const userAuthenticated = computed(() => authStore.authenticated)
const loading = taskQuery.isFetching
const filterIdUsedOnOverview = computed(() => authStore.settings?.frontendSettings?.filterIdUsedOnOverview)

interface dateStrings {
	dateFrom: string,
	dateTo: string,
}

function setDate(dates: dateStrings) {
	router.push({
		name: route.name as string,
		query: {
			from: dates.dateFrom ?? props.dateFrom,
			to: dates.dateTo ?? props.dateTo,
			showOverdue: props.showOverdue ? 'true' : 'false',
			showNulls: props.showNulls ? 'true' : 'false',
		},
	})
}

function setShowOverdue(show: boolean) {
	router.push({
		name: route.name as string,
		query: {
			...route.query,
			showOverdue: show ? 'true' : 'false',
		},
	})
}

function setShowNulls(show: boolean) {
	router.push({
		name: route.name as string,
		query: {
			...route.query,
			showNulls: show ? 'true' : 'false',
		},
	})
}

function clearLabelFilter() {
	emit('clearLabelFilter')
}

async function loadPendingTasks(from: Date|string, to: Date|string, filterId: number | null | undefined) {
	// FIXME: HACK! This should never happen.
	// Since this route is authentication only, users would get an error message if they access the page unauthenticated.
	// Since this component is mounted as the home page before unauthenticated users get redirected
	// to the login page, they will almost always see the error message.
	if (!userAuthenticated.value) {
		return
	}

	const params: TaskFilterParams = {
		sort_by: ['due_date', 'id'],
		order_by: ['asc', 'desc'],
		filter: 'done = false',
		filter_include_nulls: props.showNulls,
		q: '',
		expand: ['comment_count', 'is_unread'],
	}

	if (!showAll.value) {

		params.filter += ` && due_date < '${to instanceof Date ? to.toISOString() : to}'`

		// NOTE: Ideally we could also show tasks with a start or end date in the specified range, but the api
		//       is not capable (yet) of combining multiple filters with 'and' and 'or'.

		if (!props.showOverdue) {
			params.filter += ` && due_date > '${from instanceof Date ? from.toISOString() : from}'`
		}
	}

	// Add label filtering
	if (props.labelIds && props.labelIds.length > 0) {
		const labelFilter = `labels in ${props.labelIds.join(', ')}`
		params.filter += params.filter ? ` && ${labelFilter}` : labelFilter
	}

	let projectId = null
	if (showAll.value && filterId && typeof projectList.projects[filterId] !== 'undefined'
		&& (!props.labelIds || props.labelIds.length === 0)) {
		projectId = filterId
	}

	taskScope.value = {project: projectId, params: {...params, filter_timezone: authStore.settings.timezone}}
}

watch(taskQuery.data, data => { if (data) emit('tasksLoaded', true) })

function updateTasks() { return taskQuery.refetch() }

// Keep sidebar setting changes from reloading tasks.
watch(
	[
		() => props.dateFrom,
		() => props.dateTo,
		filterIdUsedOnOverview,
		() => props.showOverdue,
		() => props.showNulls,
		() => props.labelIds,
	],
	([from, to, filterId]) => loadPendingTasks(from, to, filterId),
	{immediate: true},
)
watchEffect(() => setTitle(pageTitle.value))
</script>

<style lang="scss" scoped>
.tasks {
	list-style: none;
	margin: 0;
}

.show-tasks-options {
	display: flex;
	flex-direction: column;

	@include mobile {
		flex-direction: row;
		align-items: center;
		gap: var(--space-2);
	}
}

.show-tasks-filters-sheet {
	display: flex;
	flex-direction: column;
	gap: var(--space-3);
}

.tasks-grouped__day + .tasks-grouped__day {
	margin-block-start: var(--space-4);
}

.tasks-grouped__day-heading {
	position: sticky;
	// Rests just below the fixed mobile header instead of under it.
	inset-block-start: calc(var(--mobile-header-height) + env(safe-area-inset-top));
	z-index: 1;
	margin: 0;
	padding: var(--space-2) var(--space-1);
	background: var(--site-background);
	font-size: var(--font-size-sm);
	font-weight: var(--font-weight-bold);
	color: var(--text-muted);
}

.llama-cool {
	margin: var(--space-12) auto 0;
	display: block;
}

.label-filter-info {
	margin-block-end: var(--space-4);
	
	.clear-filter-button {
		margin-inline-start: auto;
		padding: var(--space-1) var(--space-2);
		
		&:hover {
			color: var(--danger);
		}
	}

	:deep(.message.info) {
		inline-size: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: var(--space-2);
	}
}
</style>
