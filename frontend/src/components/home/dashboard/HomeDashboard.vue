<template>
	<div class="home-dashboard">
		<h2 class="home-dashboard__heading">
			{{ $t('home.dashboard.heading') }}
		</h2>

		<div class="home-dashboard__tiles">
			<div
				v-for="tile in tiles"
				:key="tile.key"
				class="home-dashboard__tile"
			>
				<Icon
					:icon="tile.icon"
					aria-hidden="true"
					class="home-dashboard__tile-icon"
				/>
				<Skeleton
					v-if="tile.loading"
					shape="text"
					width="2ch"
					class="home-dashboard__tile-value"
				/>
				<span
					v-else
					class="home-dashboard__tile-value"
				>{{ tile.value }}</span>
				<span class="home-dashboard__tile-label">{{ tile.label }}</span>
			</div>
		</div>

		<div class="home-dashboard__focus">
			<Icon
				:icon="['far', 'clock']"
				aria-hidden="true"
				class="home-dashboard__focus-icon"
			/>
			<div class="home-dashboard__focus-body">
				<template v-if="pomodoro.todayLoading.value">
					<Skeleton
						shape="text"
						width="4ch"
						class="home-dashboard__focus-value"
					/>
					<Skeleton
						shape="text"
						width="8ch"
						class="home-dashboard__focus-label"
					/>
				</template>
				<template v-else>
					<span class="home-dashboard__focus-value">{{ formatDuration(pomodoro.todayFocusSeconds.value) }}</span>
					<span class="home-dashboard__focus-label">
						{{ $t('home.dashboard.focus.sessionsToday', pomodoro.completedToday.value) }}
					</span>
				</template>
			</div>
		</div>

		<div
			v-if="recentProjects.length > 0"
			class="home-dashboard__recent"
		>
			<h2 class="home-dashboard__heading">
				{{ $t('home.lastViewed') }}
			</h2>
			<ProjectCardGrid
				v-cy="'projectCardGrid'"
				:projects="recentProjects"
				:show-even-number-of-projects="true"
			/>
		</div>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import type {IconProp} from '@fortawesome/fontawesome-svg-core'
import {useI18n} from 'vue-i18n'

import Icon from '@/components/misc/Icon'
import Skeleton from '@/components/misc/Skeleton.vue'
import ProjectCardGrid from '@/components/project/partials/ProjectCardGrid.vue'

import {useTasks} from '@/composables/useTasks'
import {usePomodoro} from '@/composables/usePomodoro'
import {useProjects} from '@/composables/useProjects'
import {useAuthStore} from '@/stores/auth'
import {formatDuration} from '@/helpers/time/formatDuration'
import {getHistory} from '@/modules/projectHistory'
import type {ProjectResponse} from '@/client/queries/projects'
import type {TaskScope} from '@/client/queries/tasks'

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const projectList = useProjects()
const pomodoro = usePomodoro()

const authenticated = () => authStore.authenticated

// Only the total count is needed, so per_page is kept at the minimum instead of
// pulling a full page of tasks just to read its length.
function countScope(filter: string): TaskScope {
	return {
		params: {
			filter,
			filter_timezone: authStore.settings.timezone,
			per_page: 1,
		},
	}
}

const overdueQuery = useTasks(
	() => countScope('done = false && due_date < now/d'),
	{enabled: authenticated},
)
const dueTodayQuery = useTasks(
	() => countScope('done = false && due_date >= now/d && due_date < now/d+1d'),
	{enabled: authenticated},
)
const dueWeekQuery = useTasks(
	() => countScope('done = false && due_date >= now/d && due_date < now/d+7d'),
	{enabled: authenticated},
)
const doneWeekQuery = useTasks(
	() => countScope('done = true && done_at >= now/d-7d'),
	{enabled: authenticated},
)

const tiles = computed<{key: string, icon: IconProp, value: number, loading: boolean, label: string}[]>(() => [
	{
		key: 'overdue',
		icon: 'exclamation-circle',
		value: overdueQuery.total.value,
		loading: overdueQuery.isPending.value,
		label: t('home.dashboard.tiles.overdue'),
	},
	{
		key: 'dueToday',
		icon: 'calendar',
		value: dueTodayQuery.total.value,
		loading: dueTodayQuery.isPending.value,
		label: t('home.dashboard.tiles.dueToday'),
	},
	{
		key: 'dueWeek',
		icon: ['far', 'calendar-alt'],
		value: dueWeekQuery.total.value,
		loading: dueWeekQuery.isPending.value,
		label: t('home.dashboard.tiles.dueWeek'),
	},
	{
		key: 'doneWeek',
		icon: 'check-double',
		value: doneWeekQuery.total.value,
		loading: doneWeekQuery.isPending.value,
		label: t('home.dashboard.tiles.doneWeek'),
	},
])

// Projects visited recently, most recent first; re-read against the live project
// list so a renamed or deleted project never shows stale data. Hidden entirely
// when the user turned the "last viewed" setting off.
const recentProjects = computed<ProjectResponse[]>(() => {
	if (!authStore.authenticated || authStore.settings.frontendSettings.showLastViewed === false) {
		return []
	}
	return getHistory()
		.map(entry => projectList.projects[entry.id])
		.filter((project): project is ProjectResponse => Boolean(project))
})
</script>

<style lang="scss" scoped>
.home-dashboard__heading {
	margin-block-end: var(--space-3);
}

.home-dashboard__tiles {
	display: grid;
	grid-template-columns: repeat(2, 1fr);
	gap: var(--space-4);
	margin-block-end: var(--space-4);

	@media screen and (min-width: $tablet) {
		grid-template-columns: repeat(4, 1fr);
	}

	// A horizontally scrollable strip reads better than a cramped 2-column grid on a phone.
	@include mobile {
		display: flex;
		overflow-x: auto;
		scroll-snap-type: x mandatory;
		-webkit-overflow-scrolling: touch;
		gap: var(--space-3);
		margin-inline: calc(-1 * var(--space-2));
		padding-inline: var(--space-2);
		scrollbar-width: none;

		&::-webkit-scrollbar {
			display: none;
		}
	}
}

.home-dashboard__tile {
	display: flex;
	flex-direction: column;
	gap: var(--space-1);
	padding: var(--space-4);
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: var(--radius-lg);
	box-shadow: var(--shadow-sm);

	@include mobile {
		flex: 0 0 auto;
		inline-size: 40vw;
		max-inline-size: 10rem;
		scroll-snap-align: start;
	}
}

.home-dashboard__tile-icon {
	color: var(--grey-400);
}

.home-dashboard__tile-value {
	font-size: var(--font-size-xl);
	font-weight: var(--font-weight-bold);
	font-variant-numeric: tabular-nums;
}

.home-dashboard__tile-label {
	color: var(--text-muted);
	font-size: var(--font-size-sm);
}

.home-dashboard__focus {
	display: flex;
	align-items: center;
	gap: var(--space-4);
	padding: var(--space-4);
	margin-block-end: var(--space-6);
	background: var(--white);
	border: 1px solid var(--card-border-color);
	border-radius: var(--radius-lg);
	box-shadow: var(--shadow-sm);
}

.home-dashboard__focus-icon {
	color: var(--grey-400);
	font-size: var(--font-size-xl);
}

.home-dashboard__focus-body {
	display: flex;
	flex-direction: column;
}

.home-dashboard__focus-value {
	font-size: var(--font-size-xl);
	font-weight: var(--font-weight-bold);
	font-variant-numeric: tabular-nums;
}

.home-dashboard__focus-label {
	color: var(--text-muted);
	font-size: var(--font-size-sm);
}
</style>
