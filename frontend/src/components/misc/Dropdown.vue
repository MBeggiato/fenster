<template>
	<div
		ref="dropdown"
		class="dropdown"
		@pointerenter="initialMount = true"
		@keydown="onKeydown"
	>
		<slot
			name="trigger"
			:close="close"
			:toggle-open="toggleOpen"
			:open="open"
		>
			<BaseButton
				class="dropdown-trigger is-flex"
				:aria-label="triggerLabel"
				:aria-expanded="open"
				@click="toggleOpen"
			>
				<Icon
					:icon="triggerIcon"
					class="icon"
				/>
			</BaseButton>
		</slot>

		<Modal
			v-if="asSheet && open"
			variant="sheet"
			:title="sheetTitle"
			@close="closeFromSheet"
		>
			<div
				class="dropdown-content is-sheet"
				@click="onSheetContentClick"
			>
				<slot :close="close" />
			</div>
		</Modal>

		<CustomTransition
			v-else
			name="fade"
		>
			<div
				v-if="initialMount || open"
				v-show="open"
				ref="dropdownMenu"
				class="dropdown-menu"
				:style="dropdownMenuStyle"
			>
				<div class="dropdown-content">
					<slot :close="close" />
				</div>
			</div>
		</CustomTransition>
	</div>
</template>

<script setup lang="ts">
import {ref, nextTick, watch, computed} from 'vue'
import {onClickOutside} from '@vueuse/core'
import {computePosition, autoPlacement, offset, shift} from '@floating-ui/dom'
import type {IconProp} from '@fortawesome/fontawesome-svg-core'

import CustomTransition from '@/components/misc/CustomTransition.vue'
import BaseButton from '@/components/base/BaseButton.vue'
import Modal from '@/components/misc/Modal.vue'
import {useIsMobile} from '@/composables/useIsMobile'

const props = withDefaults(defineProps<{
	triggerIcon?: IconProp
	triggerLabel?: string
	// Renders the menu as a bottom sheet on mobile instead of a floating dropdown-menu, mirroring
	// Popup.vue's sheetOnMobile. Defaults on (unlike Popup's opt-in) so existing consumers get a
	// sheet for free; pass `false` for a consumer where a sheet is the wrong call.
	sheetOnMobile?: boolean
	sheetTitle?: string
}>(), {
	triggerIcon: 'ellipsis-h',
	triggerLabel: undefined,
	sheetOnMobile: true,
	sheetTitle: '',
})

const emit = defineEmits<{
	'close': [event: Event]
}>()

const isMobile = useIsMobile()
const asSheet = computed(() => props.sheetOnMobile && isMobile.value)

defineSlots<{
	'trigger': (props: {
		close: () => void,
		toggleOpen: () => void, 
		open: boolean
	}) => void,
	'default': (props: {close: () => void}) => void
}>()

const initialMount = ref(false)
const open = ref(false)

const dropdown = ref<HTMLElement>()
const dropdownMenu = ref<HTMLElement>()
const dropdownPosition = ref({x: 0, y: 0})
const dropdownMenuOffset = computed(() => 4)

function close() {
	open.value = false
}

// Sheet mode: Modal owns backdrop-tap/Escape dismissal itself and emits 'close' with no
// payload, so mirror that here for the one consumer (ProjectKanban's bucket menu) that
// listens for the dropdown's own 'close' event to reset local state.
function closeFromSheet(e?: Event) {
	close()
	emit('close', e as Event)
}

// Selecting an item (anything rendered by DropdownItem, which always carries the
// `dropdown-item` class) closes the sheet, like tapping outside a floating dropdown
// would. Consumers that need the sheet to stay open after a click (e.g. an inline
// "set limit" input) already stop propagation on that click, same as they do for the
// floating variant today.
function onSheetContentClick(e: MouseEvent) {
	if (!(e.target as HTMLElement)?.closest?.('.dropdown-item')) {
		return
	}
	close()
	emit('close', e)
}

async function updatePosition() {
	if (!dropdown.value || !dropdownMenu.value) {
		return
	}

	await nextTick()

	const {x, y} = await computePosition(dropdown.value, dropdownMenu.value, {
		placement: 'bottom-end',
		strategy: 'absolute',
		middleware: [
			offset(dropdownMenuOffset.value),
			autoPlacement({
				allowedPlacements: ['bottom-end', 'top-end', 'bottom-start', 'top-start'],
				padding: 8,
			}),
			shift({padding: 8}),
		],
	})

	dropdownPosition.value = {x, y}
}

const dropdownMenuStyle = computed(() => ({
	left: `${dropdownPosition.value.x}px`,
	top: `${dropdownPosition.value.y}px`,
	'--hover-offset': `${dropdownMenuOffset.value}px`,
}))

function toggleOpen() {
	open.value = !open.value
}

function onKeydown(e: KeyboardEvent) {
	if (e.key !== 'Escape' || !open.value) {
		return
	}
	e.stopPropagation()
	close()
	focusTrigger()
}

// Return focus to the trigger, which is the first focusable element that lives
// outside the popup menu.
function focusTrigger() {
	const focusables = dropdown.value?.querySelectorAll<HTMLElement>('button, a[href], input, [tabindex]')
	for (const el of focusables ?? []) {
		if (!el.closest('.dropdown-menu')) {
			el.focus()
			return
		}
	}
}

watch(open, (isOpen) => {
	if (isOpen) {
		updatePosition()
	}
})

onClickOutside(dropdown, (e) => {
	// The sheet is teleported to <body> by Modal, so it never sits inside `dropdown`'s
	// DOM subtree — every tap inside it would otherwise look like an outside click.
	// Modal already handles its own backdrop-tap/Escape dismissal (see closeFromSheet).
	if (!open.value || asSheet.value) {
		return
	}
	close()
	emit('close', e)
})
</script>

<style lang="scss" scoped>
.dropdown {
	display: inline-flex;
	position: relative;
}

.dropdown-menu::before {
  content: "";
  position: absolute;
  inset: calc(var(--hover-offset) * -1);
}

.dropdown-menu {
	min-inline-size: 12rem;
	position: absolute;
	z-index: 20;
	display: block;
}

.dropdown-content {
	background: var(--glass-overlay-bg);
	backdrop-filter: var(--glass-filter-strong);
	border: 1px solid var(--glass-hairline);
	border-radius: var(--radius-md);
	padding-block-end: var(--space-2);
	padding-block-start: var(--space-2);
	box-shadow: var(--glass-specular), var(--shadow-lg);
}

// Modal already supplies the sheet's own glass background/border/shadow; reset the
// floating-variant chrome here so it isn't nested inside itself, and bump tap targets to
// 44px, same tokens MoreSheet.vue uses for its own DropdownItem list.
.dropdown-content.is-sheet {
	background: none;
	backdrop-filter: none;
	border: none;
	border-radius: 0;
	box-shadow: none;
	padding-block: var(--space-2);

	:deep(.dropdown-item) {
		min-block-size: 44px;
		font-size: var(--font-size-md);
	}
}

.dropdown-divider {
	background-color: var(--border-light);
	border: none;
	display: block;
	block-size: 1px;
	margin: var(--space-2) 0;
}
</style>
