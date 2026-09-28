<template>
	<Modal
		:enabled="isOpen"
		variant="sheet"
		:title="$t('mobile.capture.title')"
		@close="captureSheet.close()"
	>
		<div
			v-if="chips.length > 0"
			class="capture-sheet__chips"
		>
			<button
				v-for="chip in chips"
				:key="chip.label"
				type="button"
				class="capture-sheet__chip"
				@click="addTaskRef?.insertToken(chip.token)"
			>
				{{ chip.label }}
			</button>
		</div>
		<AddTask
			ref="addTaskRef"
			autofocus
			@tasksAdded="onTasksAdded"
		/>
	</Modal>
</template>

<script lang="ts" setup>
import {computed, ref} from 'vue'
import {useI18n} from 'vue-i18n'

import Modal from '@/components/misc/Modal.vue'
import AddTask from '@/components/tasks/AddTask.vue'

import type {Task as ITask} from '@/client/generated'
import {PREFIXES, PrefixMode} from '@/modules/quickAddMagic'
import {useAuthStore} from '@/stores/auth'
import {useCaptureSheet} from '@/composables/useCaptureSheet'
import {success} from '@/message'

const {t} = useI18n({useScope: 'global'})
const authStore = useAuthStore()
const captureSheet = useCaptureSheet()
const {isOpen} = captureSheet
const addTaskRef = ref<typeof AddTask | null>(null)

const mode = computed(() => authStore.settings.frontendSettings.quickAddMagicMode)
const prefixes = computed(() => PREFIXES[mode.value])

// Mode "disabled" turns off all quick-add magic parsing, including dates
// (see modules/quickAddMagic/quickAddMagic.ts) — nothing for chips to insert then.
const chips = computed(() => {
	if (mode.value === PrefixMode.Disabled || !prefixes.value) {
		return []
	}

	return [
		{label: t('mobile.capture.chips.today'), token: 'today '},
		{label: t('mobile.capture.chips.tomorrow'), token: 'tomorrow '},
		{label: `${prefixes.value.project} ${t('mobile.capture.chips.project')}`, token: prefixes.value.project},
		{label: `${prefixes.value.label} ${t('mobile.capture.chips.label')}`, token: prefixes.value.label},
		{label: `${prefixes.value.priority} ${t('mobile.capture.chips.priority')}`, token: prefixes.value.priority},
	]
})

function onTasksAdded(tasks: ITask[]) {
	if (tasks.length === 0) {
		return
	}
	success({message: t('task.createSuccess')})
	captureSheet.close()
}
</script>

<style lang="scss" scoped>
.capture-sheet__chips {
	display: flex;
	flex-wrap: wrap;
	gap: var(--space-2);
	padding: 0 var(--space-4) var(--space-4);
}

.capture-sheet__chip {
	min-block-size: 44px;
	padding: 0 var(--space-3);
	border: 1px solid var(--grey-200);
	border-radius: $radius-rounded;
	background: var(--white);
	color: var(--grey-700);
	font-size: var(--font-size-sm);

	@media (prefers-reduced-motion: no-preference) {
		transition: background-color $transition, border-color $transition;
	}

	&:active {
		background: var(--grey-100);
		border-color: var(--grey-300);
	}
}
</style>
