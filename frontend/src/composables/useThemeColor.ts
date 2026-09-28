import {watchEffect} from 'vue'
import {useColorScheme} from '@/composables/useColorScheme'

// Keeps the mobile browser chrome (status bar / PWA title bar) in sync with
// the app's own light/dark theme instead of the static color from index.html.
const THEME_COLOR_LIGHT = '#1973ff'
const THEME_COLOR_DARK = '#111827' // matches --grey-50 in dark mode

export function useThemeColor() {
	const {isDark} = useColorScheme()

	watchEffect(() => {
		document
			.querySelector('meta[name="theme-color"]')
			?.setAttribute('content', isDark.value ? THEME_COLOR_DARK : THEME_COLOR_LIGHT)
	})
}
