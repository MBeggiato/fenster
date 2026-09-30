<template>
	<Teleport :to="teleportTarget">
		<Notifications
			position="bottom left"
			:max="2"
			:ignore-duplicates="true"
			class="global-notification"
			role="status"
			aria-live="polite"
		>
			<template #body="{ item, close }">
				<!-- FIXME: overlay whole notification with button and add event listener on that button instead -->
				<div
					class="vue-notification-template vue-notification"
					:class="[
						item.type,
					]"
					@click="close()"
				>
					<div
						v-if="item.title"
						class="notification-title"
					>
						{{ item.title }}
					</div>
					<div class="notification-content">
						<template v-if="Array.isArray(item.text)">
							<template
								v-for="(t, k) in item.text"
								:key="k"
							>
								{{ t }}<br>
							</template>
						</template>
						<template v-else>
							{{ item.text }}
						</template>
						<span
							v-if="item.duplicates > 0"
							class="duplicate-count"
						>
							×{{ item.duplicates + 1 }}
						</span>
					</div>
					<div
						v-if="item.data?.actions?.length > 0"
						class="mbs-2 notification-actions"
					>
						<XButton
							v-for="(action, i) in item.data.actions"
							:key="'action_' + i"
							:shadow="false"
							class="is-small"
							variant="secondary"
							@click="action.callback"
						>
							{{ action.title }}
						</XButton>
					</div>
				</div>
			</template>
		</Notifications>
	</Teleport>
</template>

<script lang="ts" setup>
import {onBeforeUnmount, onMounted, ref} from 'vue'

const teleportTarget = ref<string | HTMLElement>('body')
let observer: MutationObserver | null = null

function syncTeleportTarget() {
	const dialogs = document.querySelectorAll<HTMLDialogElement>('dialog.modal-dialog[open]')
	teleportTarget.value = dialogs.item(dialogs.length - 1) ?? 'body'
}

onMounted(() => {
	syncTeleportTarget()
	observer = new MutationObserver(syncTeleportTarget)
	observer.observe(document.body, {
		attributes: true,
		attributeFilter: ['open'],
		childList: true,
		subtree: true,
	})
})

onBeforeUnmount(() => {
	observer?.disconnect()
	observer = null
})
</script>

<style scoped lang="scss">
.vue-notification {
	z-index: 9999;

	/*
	 * The library injects its own per-type background (info/success/warn/error) at
	 * runtime; that color-coding is the semantic signal so it stays untouched here.
	 * backdrop-filter would be inert behind an opaque fill, so only the structural
	 * glass properties (bevel, radius, elevation) are layered on top.
	 */
	border-radius: var(--radius-md);
	box-shadow: var(--glass-specular), var(--shadow-md);
}

.global-notification {
	@include mobile {
		inset-block-end: calc(var(--mobile-tabbar-height) + env(safe-area-inset-bottom) + var(--space-4)) !important;
		inset-inline: var(--space-4) !important;
		inline-size: auto !important;
	}
}

.duplicate-count {
	font-size: var(--font-size-xs);
	font-weight: var(--font-weight-bold);
	margin-inline-start: var(--space-1);
}

.notification-actions {
	display: flex;
	justify-content: flex-end;
	gap: var(--space-2);
}

</style>
