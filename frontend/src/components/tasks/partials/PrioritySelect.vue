<template>
	<div class="select">
		<select
			v-model="priority"
			:disabled="disabled || undefined"
			:aria-label="$t('task.attributes.priority')"
			:style="{ borderColor: priorityColor }"
		>
			<option :value="PRIORITIES.UNSET">
				{{ $t('task.priority.unset') }}
			</option>
			<option :value="PRIORITIES.LOW">
				{{ $t('task.priority.low') }}
			</option>
			<option :value="PRIORITIES.MEDIUM">
				{{ $t('task.priority.medium') }}
			</option>
			<option :value="PRIORITIES.HIGH">
				{{ $t('task.priority.high') }}
			</option>
			<option :value="PRIORITIES.URGENT">
				{{ $t('task.priority.urgent') }}
			</option>
			<option :value="PRIORITIES.DO_NOW">
				{{ $t('task.priority.doNow') }}
			</option>
		</select>
	</div>
</template>

<script setup lang="ts">
import {computed} from 'vue'
import {PRIORITIES} from '@/constants/priorities'
import {getPriorityColorVar} from '@/helpers/priorityColor'

withDefaults(defineProps<{
	disabled?: boolean
}>(), {
	disabled: false,
})

const priority = defineModel<number>({
	required: true,
	default: 0,
})

const priorityColor = computed(() => priority.value === PRIORITIES.UNSET ? undefined : getPriorityColorVar(priority.value))

</script>
