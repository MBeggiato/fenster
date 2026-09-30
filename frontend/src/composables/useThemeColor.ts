import {watchEffect} from 'vue'
import {useColorScheme} from '@/composables/useColorScheme'

// Keeps the browser chrome / PWA status bar in sync with the app's own theme,
// which the user can force independently of the OS setting (the media-query
// meta tags in index.html only follow the OS).
const THEME_COLOR_LIGHT = '#f3f4f6' // --site-background, light
const THEME_COLOR_DARK = '#1f2937' // --site-background (--grey-100), dark

export function useThemeColor() {
	const {isDark} = useColorScheme()

	watchEffect(() => {
		document
			.querySelectorAll('meta[name="theme-color"]')
			.forEach(meta => meta.setAttribute('content', isDark.value ? THEME_COLOR_DARK : THEME_COLOR_LIGHT))
	})
}
