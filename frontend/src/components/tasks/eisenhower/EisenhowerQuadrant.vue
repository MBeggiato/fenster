<template>
	<section
		class="eisenhower-quadrant"
		:class="`is-${quadrant}`"
		:data-quadrant="quadrant"
		:aria-labelledby="headingId"
	>
		<header class="quadrant-header">
			<h3
				:id="headingId"
				class="quadrant-title"
			>
				{{ title }}
				<span
					v-if="total > 0"
					class="quadrant-count"
				>{{ total }}</span>
			</h3>
			<p class="quadrant-hint">
				{{ hint }}
			</p>
		</header>

		<form
			class="quadrant-add"
			@submit.prevent="addTask"
		>
			<input
				v-model="newTaskTitle"
				class="input"
				type="text"
				:placeholder="$t('project.list.addPlaceholder')"
				:aria-label="$t('task.eisenhower.addLabel', {quadrant: title})"
				:disabled="isCreating"
			>
			<p
				v-if="createError"
				class="help is-danger"
			>
				{{ createError }}
			</p>
		</form>

		<draggable
			:model-value="visibleTasks"
			:group="{name: 'eisenhower'}"
			item-key="id"
			tag="ul"
			class="quadrant-tasks"
			:data-quadrant="quadrant"
			:animation="150"
			:delay-on-touch-only="true"
			:delay="300"
			ghost-class="task-ghost"
			@end="handleDrop"
		>
			<template #item="{element: task}">
				<li :data-task-id="task.id">
					<SingleTaskInProject
						:show-project="true"
						:the-task="task"
						:can-mark-as-done="(projects.projects[task.project_id]?.max_permission ?? 0) > PERMISSIONS.READ"
					/>
				</li>
			</template>
		</draggable>

		<p
			v-if="visibleTasks.length === 0 && !isLoading"
			class="quadrant-empty"
		>
			{{ $t('task.eisenhower.empty') }}
		</p>

		<div
			v-if="isLoading"
			class="is-loading loader-container quadrant-loading"
		/>
		<XButton
			v-if="hasMore"
			variant="tertiary"
			class="quadrant-more"
			:loading="isFetching"
			@click="loadMore"
		>
			{{ $t('task.eisenhower.loadMore') }}
		</XButton>
	</section>
</template>

<script setup lang="ts">
import {computed, ref, useId} from 'vue'
import {useI18n} from 'vue-i18n'
import draggable from 'zhyswan-vuedraggable'

import SingleTaskInProject from '@/components/tasks/partials/SingleTaskInProject.vue'
import {
	flagsFor,
	useClassifyTaskMutation,
	type EisenhowerParams,
	type EisenhowerQuadrant,
} from '@/client/queries/eisenhower'
import {normalizeTask} from '@/client/queries/tasks'
import {useEisenhowerQuadrant} from '@/composables/useEisenhowerQuadrant'
import {useProjects} from '@/composables/useProjects'
import {useQuickAddTask} from '@/composables/useQuickAddTask'
import {PERMISSIONS} from '@/constants/permissions'
import {error} from '@/message'

const props = defineProps<{
	quadrant: EisenhowerQuadrant
	title: string
	hint: string
	params: EisenhowerParams
	// Where tasks typed here are created when the title names no project.
	projectId: number
}>()

const {t} = useI18n({useScope: 'global'})
const headingId = useId()
const projects = useProjects()

const {tasks, total, hasMore, isLoading, isFetching, loadMore} = useEisenhowerQuadrant(
	() => props.quadrant,
	() => props.params,
)

// A task marked done here stays in the cache until the next refetch.
const visibleTasks = computed(() => props.params.include_done
	? tasks.value
	: tasks.value.filter(task => !task.done))

const classify = useClassifyTaskMutation()

function handleDrop(e: {to: HTMLElement, from: HTMLElement, item: HTMLElement}) {
	if (e.to === e.from) return
	const target = e.to.dataset.quadrant as EisenhowerQuadrant | undefined
	const taskId = Number(e.item.dataset.taskId)
	const task = tasks.value.find(item => item.id === taskId)
	if (!target || !task) return
	classify.mutate({task, classification: flagsFor(target)})
}

const {createNewTask} = useQuickAddTask()
const newTaskTitle = ref('')
const isCreating = ref(false)
const createError = ref('')

async function addTask() {
	const title = newTaskTitle.value.trim()
	if (title === '' || isCreating.value) return
	isCreating.value = true
	createError.value = ''
	try {
		let created
		try {
			created = await createNewTask({title, project_id: props.projectId})
		} catch (e) {
			if (e instanceof Error && e.message === 'NO_PROJECT') {
				createError.value = t('project.create.addProjectRequired')
			}
			return
		}
		newTaskTitle.value = ''
		const classification = flagsFor(props.quadrant)
		if (classification === null) return
		try {
			await classify.mutateAsync({task: normalizeTask(created), classification})
		} catch {
			// The task exists but stays unclassified; the mutation already reported why.
			error({message: t('task.eisenhower.createdUnclassified', {task: title})})
		}
	} finally {
		isCreating.value = false
	}
}
</script>

<style lang="scss" scoped>
.eisenhower-quadrant {
	display: flex;
	flex-direction: column;
	gap: var(--space-2);
	min-block-size: 12rem;
	padding: var(--space-3);
	border-radius: $radius;
	background: var(--white);
	box-shadow: var(--shadow-sm);
	border-block-start: 4px solid var(--quadrant-accent, var(--grey-300));

	&.is-do {
		--quadrant-accent: var(--danger);
	}

	&.is-schedule {
		--quadrant-accent: var(--primary);
	}

	&.is-delegate {
		--quadrant-accent: var(--warning);
	}

	&.is-eliminate {
		--quadrant-accent: var(--grey-400);
	}
}

.quadrant-title {
	display: flex;
	align-items: center;
	gap: var(--space-2);
	margin: 0;
	font-size: 1.1rem;
	font-weight: 700;
}

.quadrant-count {
	padding-inline: var(--space-2);
	border-radius: 1rem;
	background: var(--grey-100);
	color: var(--grey-600);
	font-size: .8rem;
	font-weight: 400;
}

.quadrant-hint {
	margin: 0;
	color: var(--grey-500);
	font-size: .85rem;
}

// Keeps an empty area a drop target.
.quadrant-tasks {
	flex: 1;
	min-block-size: 3rem;
	margin: 0;
	list-style: none;
}

// Sits on top of the empty list and lets drops through to it.
.quadrant-empty {
	margin: -3rem 0 0;
	padding: var(--space-4) var(--space-2);
	pointer-events: none;
	color: var(--grey-500);
	text-align: center;
	font-size: .9rem;
}

.quadrant-loading {
	min-block-size: 3rem;
}

.quadrant-more {
	align-self: center;
}

.task-ghost {
	border-radius: $radius;
	background: var(--grey-100);
	border: 2px dashed var(--grey-300);

	* {
		opacity: 0;
	}
}
</style>
