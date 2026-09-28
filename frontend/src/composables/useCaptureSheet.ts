import {ref} from 'vue'
import {createGlobalState} from '@vueuse/core'

// Shared open/close state for the mobile capture sheet (built in Wave 1).
// The tab bar's capture button and the `?capture=1` deep link both just flip `isOpen`.
export const useCaptureSheet = createGlobalState(() => {
	const isOpen = ref(false)

	function open() {
		isOpen.value = true
	}

	function close() {
		isOpen.value = false
	}

	return {isOpen, open, close}
})
