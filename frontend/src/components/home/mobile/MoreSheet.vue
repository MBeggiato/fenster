<template>
	<Modal
		:enabled="enabled"
		variant="sheet"
		:title="$t('mobile.moreSheet.title')"
		@close="$emit('close')"
	>
		<DropdownItem
			:to="{name: 'tasks.eisenhower'}"
			icon="table-cells-large"
			@click="$emit('close')"
		>
			{{ $t('navigation.eisenhower') }}
		</DropdownItem>
		<DropdownItem
			:to="{name: 'labels.index'}"
			icon="tags"
			@click="$emit('close')"
		>
			{{ $t('label.title') }}
		</DropdownItem>
		<DropdownItem
			:to="{name: 'teams.index'}"
			icon="users"
			@click="$emit('close')"
		>
			{{ $t('team.title') }}
		</DropdownItem>
		<DropdownItem
			v-if="timeTrackingEnabled"
			:to="{name: 'time-tracking'}"
			:icon="['far', 'clock']"
			@click="$emit('close')"
		>
			{{ $t('timeTracking.title') }}
		</DropdownItem>
		<DropdownItem
			:to="{name: 'user.settings'}"
			icon="cog"
			@click="$emit('close')"
		>
			{{ $t('navigation.settings') }}
		</DropdownItem>
		<DropdownItem
			v-if="adminPanelEnabled && authStore.info?.isAdmin"
			:to="{name: 'admin.overview'}"
			icon="tachometer-alt"
			@click="$emit('close')"
		>
			{{ $t('admin.title') }}
		</DropdownItem>
		<DropdownItem
			icon="sign-out-alt"
			class="has-text-danger"
			@click="logout"
		>
			{{ $t('user.auth.logout') }}
		</DropdownItem>
	</Modal>
</template>

<script lang="ts" setup>
import {computed} from 'vue'

import DropdownItem from '@/components/misc/DropdownItem.vue'

import {PRO_FEATURE} from '@/constants/proFeatures'

import {useAuthStore} from '@/stores/auth'
import {useConfigStore} from '@/stores/config'

defineProps<{
	enabled: boolean,
}>()

const emit = defineEmits<{
	(e: 'close'): void,
}>()

const authStore = useAuthStore()
const configStore = useConfigStore()

const timeTrackingEnabled = computed(() => configStore.isProFeatureEnabled(PRO_FEATURE.TIME_TRACKING))
const adminPanelEnabled = computed(() => configStore.isProFeatureEnabled(PRO_FEATURE.ADMIN_PANEL))

function logout() {
	emit('close')
	authStore.logout()
}
</script>

<style lang="scss" scoped>
:deep(.dropdown-item) {
	min-block-size: 44px;
	font-size: var(--font-size-md);
}
</style>
