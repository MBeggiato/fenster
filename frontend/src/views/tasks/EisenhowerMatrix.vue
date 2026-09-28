<template>
	<div class="eisenhower-matrix has-text-start">
		<h2 class="title mbe-2">
			{{ $t('task.eisenhower.title') }}
		</h2>
		<p class="matrix-intro">
			{{ $t('task.eisenhower.intro') }}
		</p>

		<div
			class="matrix-filters"
			role="search"
		>
			<div class="field">
				<label class="label">{{ $t('task.attributes.project') }}</label>
				<ProjectSearch
					:model-value="selectedProject"
					@update:modelValue="setProject"
				/>
			</div>
			<div class="field">
				<label class="label">{{ $t('task.attributes.labels') }}</label>
				<Multiselect
					:model-value="selectedLabels"
					:multiple="true"
					:search-results="foundLabels"
					:show-empty="true"
					label="title"
					:placeholder="$t('task.eisenhower.filter.labelsPlaceholder')"
					@update:modelValue="setLabels"
					@search="findLabels"
				/>
			</div>
			<div class="field">
				<label
					class="label"
					for="eisenhower-search"
				>{{ $t('task.eisenhower.filter.search') }}</label>
				<input
					id="eisenhower-search"
					v-model="searchInput"
					class="input"
					type="search"
					:placeholder="$t('task.eisenhower.filter.searchPlaceholder')"
				>
			</div>
			<FancyCheckbox
				:model-value="showDone"
				class="matrix-show-done"
				@update:modelValue="setShowDone"
			>
				{{ $t('task.eisenhower.filter.showDone') }}
			</FancyCheckbox>
		</div>

		<p class="matrix-drag-hint">
			{{ $t('task.eisenhower.dragHint') }}
		</p>

		<div class="matrix-grid">
			<span
				class="axis axis-urgent"
				aria-hidden="true"
			>{{ $t('task.eisenhower.urgent') }}</span>
			<span
				class="axis axis-not-urgent"
				aria-hidden="true"
			>{{ $t('task.eisenhower.notUrgent') }}</span>
			<span
				class="axis axis-important"
				aria-hidden="true"
			>{{ $t('task.eisenhower.important') }}</span>
			<span
				class="axis axis-not-important"
				aria-hidden="true"
			>{{ $t('task.eisenhower.notImportant') }}</span>

			<EisenhowerQuadrant
				v-for="{quadrant} in EISENHOWER_QUADRANTS"
				:key="quadrant"
				:class="`area-${quadrant}`"
				:quadrant="quadrant"
				:title="$t(`task.eisenhower.quadrants.${quadrant}.title`)"
				:hint="$t(`task.eisenhower.quadrants.${quadrant}.hint`)"
				:params="params"
				:project-id="newTaskProjectId"
			/>
		</div>

		<EisenhowerQuadrant
			class="matrix-unclassified"
			quadrant="unclassified"
			:title="$t('task.eisenhower.quadrants.unclassified.title')"
			:hint="$t('task.eisenhower.quadrants.unclassified.hint')"
			:params="params"
			:project-id="newTaskProjectId"
		/>
	</div>
</template>

<script setup lang="ts">
import {computed, ref, watch, watchEffect} from 'vue'
import {useRoute, useRouter, type LocationQueryRaw} from 'vue-router'
import {useI18n} from 'vue-i18n'
import {watchDebounced} from '@vueuse/core'

import EisenhowerQuadrant from '@/components/tasks/eisenhower/EisenhowerQuadrant.vue'
import ProjectSearch from '@/components/tasks/partials/ProjectSearch.vue'
import Multiselect from '@/components/input/Multiselect.vue'
import FancyCheckbox from '@/components/input/FancyCheckbox.vue'
import {EISENHOWER_QUADRANTS, type EisenhowerParams} from '@/client/queries/eisenhower'
import type {ProjectResponse} from '@/client/queries/projects'
import type {Label} from '@/client/generated'
import {useProjects} from '@/composables/useProjects'
import {useLabels} from '@/composables/useLabels'
import {useAuthStore} from '@/stores/auth'
import {setTitle} from '@/helpers/setTitle'

const props = withDefaults(defineProps<{
	projectId?: number
	labelIds?: number[]
	search?: string
	showDone?: boolean
}>(), {
	projectId: 0,
	labelIds: () => [],
	search: '',
	showDone: false,
})

const {t} = useI18n({useScope: 'global'})
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const projectList = useProjects()
const {filterLabelsByQuery, getLabelsByIds} = useLabels()

watchEffect(() => setTitle(t('task.eisenhower.title')))

function updateQuery(changes: LocationQueryRaw) {
	const query = {...route.query, ...changes}
	for (const [key, value] of Object.entries(query)) {
		if (value === undefined || value === '' || value === null) delete query[key]
	}
	router.replace({name: route.name ?? undefined, query})
}

const selectedProject = computed(() => props.projectId > 0
	? projectList.projects[props.projectId] ?? null
	: null)

function setProject(project: ProjectResponse | null) {
	updateQuery({project: project?.id ? String(project.id) : undefined})
}

const selectedLabels = computed(() => getLabelsByIds(props.labelIds))
const foundLabels = ref<Label[]>([])

function findLabels(query: string) {
	foundLabels.value = filterLabelsByQuery(selectedLabels.value, query)
}

function setLabels(labels: Label | Label[] | null) {
	const ids = (Array.isArray(labels) ? labels : labels ? [labels] : [])
		.map(label => label.id)
		.filter((id): id is number => typeof id === 'number')
	updateQuery({labels: ids.length > 0 ? ids.join(',') : undefined})
}

const searchInput = ref('')
watch(() => props.search, search => { searchInput.value = search }, {immediate: true})
watchDebounced(searchInput, search => {
	if (search !== props.search) updateQuery({q: search || undefined})
}, {debounce: 300})

function setShowDone(show: boolean) {
	updateQuery({done: show ? 'true' : undefined})
}

const params = computed<EisenhowerParams>(() => {
	const filters: string[] = []
	if (props.projectId > 0) filters.push(`project in ${props.projectId}`)
	if (props.labelIds.length > 0) filters.push(`labels in ${props.labelIds.join(', ')}`)
	return {
		filter: filters.join(' && '),
		filter_timezone: authStore.settings.timezone,
		q: props.search,
		include_done: props.showDone,
		sort_by: ['due_date', 'priority', 'id'],
		order_by: ['asc', 'desc', 'asc'],
		expand: ['eisenhower'],
	}
})

// New tasks go to the filtered project, otherwise to the default one.
const newTaskProjectId = computed(() => props.projectId > 0
	? props.projectId
	: authStore.settings.defaultProjectId ?? 0)
</script>

<style lang="scss" scoped>
.eisenhower-matrix {
	--axis-size: 1.5rem;
}

.matrix-intro,
.matrix-drag-hint {
	color: var(--grey-500);
}

.matrix-filters {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
	gap: var(--space-3) var(--space-4);
	align-items: end;
	margin-block: var(--space-4);

	.field {
		margin: 0;
	}
}

.matrix-show-done {
	align-self: center;
}

.matrix-grid {
	display: grid;
	grid-template-columns: var(--axis-size) minmax(0, 1fr) minmax(0, 1fr);
	grid-template-rows: var(--axis-size) auto auto;
	grid-template-areas:
		". urgent not-urgent"
		"important do schedule"
		"not-important delegate eliminate";
	gap: var(--space-4);
	margin-block: var(--space-4);
}

.axis {
	display: flex;
	align-items: center;
	justify-content: center;
	color: var(--grey-600);
	font-size: .8rem;
	font-weight: 700;
	letter-spacing: .05em;
	text-transform: uppercase;
}

.axis-important,
.axis-not-important {
	writing-mode: vertical-rl;
	transform: rotate(180deg);
}

.axis-urgent { grid-area: urgent; }
.axis-not-urgent { grid-area: not-urgent; }
.axis-important { grid-area: important; }
.axis-not-important { grid-area: not-important; }
.area-do { grid-area: do; }
.area-schedule { grid-area: schedule; }
.area-delegate { grid-area: delegate; }
.area-eliminate { grid-area: eliminate; }

@media screen and (max-width: $tablet) {
	.matrix-grid {
		grid-template-columns: minmax(0, 1fr);
		grid-template-rows: none;
		grid-template-areas:
			"do"
			"schedule"
			"delegate"
			"eliminate";
	}

	.axis {
		display: none;
	}
}
</style>
