<template>
	<div class="pomodoro-stats">
		<div class="pomodoro-stats__range">
			<XButton
				v-for="option in RANGE_OPTIONS"
				:key="option"
				variant="secondary"
				:class="{'is-active': range === option}"
				@click="range = option"
			>
				{{ $t(`pomodoro.stats.range.${option}`) }}
			</XButton>
			<DateRangeInput
				v-if="range === 'custom'"
				v-model="customRange"
			/>
		</div>

		<Loading
			v-if="query.isPending.value"
			variant="small"
		/>
		<Nothing v-else-if="!hasData">
			{{ $t('pomodoro.stats.empty') }}
		</Nothing>
		<template v-else>
			<div class="pomodoro-stats__totals">
				<div class="pomodoro-stats__total">
					<span class="pomodoro-stats__total-value">{{ formatDuration(stats.focus_seconds ?? 0) }}</span>
					<span class="pomodoro-stats__total-label">{{ $t('pomodoro.stats.focusTime') }}</span>
				</div>
				<div class="pomodoro-stats__total">
					<span class="pomodoro-stats__total-value">{{ stats.completed ?? 0 }}</span>
					<span class="pomodoro-stats__total-label">{{ $t('pomodoro.stats.completed') }}</span>
				</div>
				<div class="pomodoro-stats__total">
					<span class="pomodoro-stats__total-value">{{ stats.interrupted ?? 0 }}</span>
					<span class="pomodoro-stats__total-label">{{ $t('pomodoro.stats.interrupted') }}</span>
				</div>
				<div class="pomodoro-stats__total">
					<span class="pomodoro-stats__total-value">{{ completionRate }}</span>
					<span class="pomodoro-stats__total-label">{{ $t('pomodoro.stats.completionRate') }}</span>
				</div>
			</div>

			<section class="pomodoro-stats__section">
				<h3 class="pomodoro-stats__heading">
					{{ $t('pomodoro.stats.perDay') }}
				</h3>
				<ol class="pomodoro-stats__bars">
					<li
						v-for="day in stats.days ?? []"
						:key="day.date"
						class="pomodoro-stats__bar-row"
					>
						<span class="pomodoro-stats__bar-label">{{ day.date }}</span>
						<span class="pomodoro-stats__bar-track">
							<span
								class="pomodoro-stats__bar"
								:style="{inlineSize: `${barWidth(day.focus_seconds ?? 0)}%`}"
							>
								<span class="pomodoro-stats__bar-count">{{ day.completed ?? 0 }}</span>
							</span>
						</span>
						<span class="pomodoro-stats__bar-value">{{ formatDuration(day.focus_seconds ?? 0) }}</span>
					</li>
				</ol>
			</section>

			<section
				v-for="group in groups"
				:key="group.heading"
				class="pomodoro-stats__section"
			>
				<h3 class="pomodoro-stats__heading">
					{{ group.heading }}
				</h3>
				<table class="table has-actions is-fullwidth">
					<thead>
						<tr>
							<th>{{ group.label }}</th>
							<th>{{ $t('pomodoro.stats.focusTime') }}</th>
							<th>{{ $t('pomodoro.stats.completed') }}</th>
							<th>{{ $t('pomodoro.stats.interrupted') }}</th>
							<th v-if="group.showEstimate">
								{{ $t('pomodoro.stats.estimate') }}
							</th>
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="row in group.rows"
							:key="`${group.heading}-${row.id}-${row.title}`"
						>
							<td>{{ row.title }}</td>
							<td>{{ formatDuration(row.focus_seconds ?? 0) }}</td>
							<td>{{ row.completed ?? 0 }}</td>
							<td>{{ row.interrupted ?? 0 }}</td>
							<td v-if="group.showEstimate">
								{{ row.estimate ? `${row.completed ?? 0} / ${row.estimate}` : '—' }}
							</td>
						</tr>
					</tbody>
				</table>
			</section>
		</template>
	</div>
</template>

<script setup lang="ts">
import {computed, ref} from 'vue'
import {useQuery} from '@tanstack/vue-query'
import {useI18n} from 'vue-i18n'

import XButton from '@/components/input/Button.vue'
import DateRangeInput from '@/components/input/DateRangeInput.vue'
import Loading from '@/components/misc/Loading.vue'
import Nothing from '@/components/misc/Nothing.vue'

import {pomodoroStatsQuery} from '@/client/queries/pomodoro'
import {formatDuration} from '@/helpers/time/formatDuration'
import {useAuthStore} from '@/stores/auth'

const RANGE_OPTIONS = ['thisWeek', 'lastWeek', 'thisMonth', 'custom'] as const
type RangeOption = typeof RANGE_OPTIONS[number]

const {t} = useI18n()
const authStore = useAuthStore()

const range = ref<RangeOption>('thisWeek')
const customRange = ref<{start: Date | null, end: Date | null}>({start: null, end: null})

// The week starts on the day the user configured, so the bars line up with the
// rest of the app's calendars.
function startOfWeek(date: Date): Date {
	const weekStart = authStore.settings.weekStart ?? 0
	const start = new Date(date)
	start.setHours(0, 0, 0, 0)
	start.setDate(start.getDate() - ((start.getDay() - weekStart + 7) % 7))
	return start
}

const resolvedRange = computed<{from: Date, to: Date}>(() => {
	const now = new Date()
	switch (range.value) {
		case 'lastWeek': {
			const from = startOfWeek(now)
			from.setDate(from.getDate() - 7)
			const to = new Date(from)
			to.setDate(to.getDate() + 7)
			return {from, to}
		}
		case 'thisMonth': {
			const from = new Date(now.getFullYear(), now.getMonth(), 1)
			return {from, to: now}
		}
		case 'custom': {
			// An incomplete custom range falls back to this week rather than
			// asking the server for an empty or reversed one.
			const {start, end} = customRange.value
			return start && end ? {from: start, to: end} : {from: startOfWeek(now), to: now}
		}
		default:
			return {from: startOfWeek(now), to: now}
	}
})

const timezone = Intl.DateTimeFormat().resolvedOptions().timeZone

const query = useQuery(computed(() => pomodoroStatsQuery(
	resolvedRange.value.from.toISOString(),
	resolvedRange.value.to.toISOString(),
	timezone,
)))

const stats = computed(() => query.data.value ?? {})
const hasData = computed(() => (stats.value.days ?? []).length > 0)

const completionRate = computed(() => {
	const completed = stats.value.completed ?? 0
	const total = completed + (stats.value.interrupted ?? 0)
	return total === 0 ? '—' : `${Math.round((completed / total) * 100)}%`
})

// Bars are scaled to the busiest day, so a quiet week still reads.
const maxDaySeconds = computed(() => Math.max(1, ...(stats.value.days ?? []).map(day => day.focus_seconds ?? 0)))
function barWidth(seconds: number): number {
	return Math.round((seconds / maxDaySeconds.value) * 100)
}

const groups = computed(() => [
	{
		heading: t('pomodoro.stats.topTasks'),
		label: t('pomodoro.stats.task'),
		rows: stats.value.tasks ?? [],
		showEstimate: true,
	},
	{
		heading: t('pomodoro.stats.topProjects'),
		label: t('pomodoro.stats.project'),
		rows: stats.value.projects ?? [],
		showEstimate: false,
	},
].filter(group => group.rows.length > 0))
</script>

<style lang="scss" scoped>
.pomodoro-stats__range {
	display: flex;
	flex-wrap: wrap;
	align-items: center;
	gap: .5rem;
	margin-block-end: 1.5rem;
}

.pomodoro-stats__totals {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr));
	gap: 1rem;
	margin-block-end: 2rem;
}

.pomodoro-stats__total {
	display: flex;
	flex-direction: column;
	gap: .25rem;
	padding: 1rem;
	border-radius: $radius;
	background: var(--grey-100);
}

.pomodoro-stats__total-value {
	font-size: 1.5rem;
	font-weight: 700;
	font-variant-numeric: tabular-nums;
}

.pomodoro-stats__total-label {
	font-size: .8125rem;
	color: var(--grey-500);
}

.pomodoro-stats__section {
	margin-block-end: 2rem;
}

.pomodoro-stats__heading {
	margin-block-end: .75rem;
	font-size: 1rem;
	font-weight: 600;
}

.pomodoro-stats__bars {
	display: flex;
	flex-direction: column;
	gap: .375rem;
	margin: 0;
	padding: 0;
	list-style: none;
}

.pomodoro-stats__bar-row {
	display: grid;
	grid-template-columns: 6rem 1fr 4.5rem;
	align-items: center;
	gap: .75rem;
	font-size: .8125rem;
}

.pomodoro-stats__bar-label,
.pomodoro-stats__bar-value {
	color: var(--grey-500);
	font-variant-numeric: tabular-nums;
}

.pomodoro-stats__bar-value {
	text-align: end;
}

.pomodoro-stats__bar-track {
	display: block;
	block-size: 1.25rem;
	border-radius: $radius;
	background: var(--grey-100);
}

.pomodoro-stats__bar {
	display: flex;
	align-items: center;
	justify-content: flex-end;
	block-size: 100%;
	min-inline-size: 1.5rem;
	padding-inline-end: .375rem;
	border-radius: $radius;
	background: var(--danger);
}

.pomodoro-stats__bar-count {
	color: var(--white);
	font-size: .75rem;
	font-weight: 600;
}

@media screen and (max-width: $tablet) {
	.pomodoro-stats__bar-row {
		grid-template-columns: 5rem 1fr 4rem;
	}
}
</style>
