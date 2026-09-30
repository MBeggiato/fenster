<template>
	<nav
		class="mobile-task-action-bar d-print-none"
		:aria-label="$t('mobile.taskDetail.actionBarLabel')"
	>
		<button
			type="button"
			class="mobile-task-action-bar__btn mobile-task-action-bar__btn--primary"
			:class="{'is-done': done}"
			:disabled="!canWrite"
			@click="$emit('toggleDone')"
		>
			<Icon icon="check-double" />
			<span>{{ done ? $t('mobile.taskDetail.reopen') : $t('mobile.taskDetail.done') }}</span>
		</button>
		<template v-if="canWrite">
			<button
				type="button"
				class="mobile-task-action-bar__btn"
				@click="$emit('due')"
			>
				<Icon icon="calendar" />
				<span>{{ $t('task.attributes.dueDate') }}</span>
			</button>
			<button
				type="button"
				class="mobile-task-action-bar__btn"
				@click="$emit('priority')"
			>
				<Icon icon="exclamation-circle" />
				<span>{{ $t('task.attributes.priority') }}</span>
			</button>
			<button
				type="button"
				class="mobile-task-action-bar__btn"
				@click="$emit('labels')"
			>
				<Icon icon="tags" />
				<span>{{ $t('task.attributes.labels') }}</span>
			</button>
		</template>
		<button
			type="button"
			class="mobile-task-action-bar__btn"
			@click="$emit('more')"
		>
			<Icon icon="ellipsis-h" />
			<span>{{ $t('mobile.taskDetail.more') }}</span>
		</button>
	</nav>
</template>

<script lang="ts" setup>
defineProps<{
	canWrite: boolean,
	done: boolean,
}>()

defineEmits<{
	(e: 'toggleDone'): void,
	(e: 'due'): void,
	(e: 'priority'): void,
	(e: 'labels'): void,
	(e: 'more'): void,
}>()
</script>

<style lang="scss" scoped>
.mobile-task-action-bar {
	position: fixed;
	inset-inline-start: 0;
	inset-inline-end: 0;
	inset-block-end: 0;
	// above the mobile tab bar (z-index: 30) so it takes over the thumb zone
	// when the task detail renders as a standalone page instead of a modal
	z-index: 31;

	display: flex;
	align-items: stretch;
	justify-content: space-around;
	block-size: calc(4rem + env(safe-area-inset-bottom));
	padding-block-end: env(safe-area-inset-bottom);

	background: var(--chrome-bg-mobile);
	backdrop-filter: none;
	box-shadow: var(--glass-specular);

	user-select: none;
	-webkit-touch-callout: none;
}

.mobile-task-action-bar__btn {
	flex: 1 1 0;
	min-inline-size: 44px;
	min-block-size: 44px;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: var(--space-1);
	border: none;
	background: transparent;
	font-size: var(--font-size-xs);
	color: var(--grey-700);
	transition: transform 120ms, opacity 120ms;

	&:active:not(:disabled) {
		opacity: .6;
		transform: scale(.96);
	}

	@media (prefers-reduced-motion: reduce) {
		transition: none;
	}

	&:disabled {
		opacity: .4;
	}
}

.mobile-task-action-bar__btn--primary {
	color: var(--success);

	&.is-done {
		color: var(--grey-400);
	}
}
</style>
