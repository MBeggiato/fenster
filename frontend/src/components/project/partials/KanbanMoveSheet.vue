<template>
	<Modal
		:enabled="enabled"
		variant="sheet"
		:title="$t('mobile.kanban.moveToTitle')"
		@close="$emit('close')"
	>
		<DropdownItem
			v-for="bucket in buckets"
			:key="bucket.id"
			:class="{'is-active': bucket.id === currentBucketId}"
			:disabled="bucket.id === currentBucketId"
			icon="th"
			:icon-class="{'has-text-primary': bucket.id === currentBucketId}"
			@click="$emit('select', bucket.id)"
		>
			{{ bucket.title }}
			<span class="kanban-move-sheet__count">{{ bucket.count }}</span>
		</DropdownItem>
	</Modal>
</template>

<script lang="ts" setup>
import Modal from '@/components/misc/Modal.vue'
import DropdownItem from '@/components/misc/DropdownItem.vue'
import type {BucketResponse} from '@/client/queries/kanban'

defineProps<{
	enabled: boolean,
	buckets: BucketResponse[],
	currentBucketId?: number | null,
}>()

defineEmits<{
	close: [],
	select: [bucketId: number],
}>()
</script>

<style lang="scss" scoped>
:deep(.dropdown-item) {
	min-block-size: 44px;
	font-size: var(--font-size-md);
}

.kanban-move-sheet__count {
	margin-inline-start: var(--space-2);
	color: var(--grey-400);
	font-size: var(--font-size-sm);
}
</style>
