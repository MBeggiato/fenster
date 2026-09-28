import {shallowRef, computed, type Ref} from 'vue'
import {usePointerSwipe, useVibrate} from '@vueuse/core'

export type SwipeAction = 'left' | 'right'
export type SwipeAxis = 'horizontal' | 'vertical' | null

/**
 * Below `threshold` px of total movement the gesture is still undecided — it could become a
 * vertical scroll or a horizontal swipe. Once decided, the caller should stick with that axis for
 * the rest of the gesture instead of re-evaluating on every move (a diagonal wobble mid-swipe
 * shouldn't hand control back to the page scroller).
 */
export function resolveSwipeAxis(dx: number, dy: number, threshold = 10): SwipeAxis {
	const adx = Math.abs(dx)
	const ady = Math.abs(dy)
	if (Math.max(adx, ady) < threshold) {
		return null
	}
	return adx >= ady ? 'horizontal' : 'vertical'
}

export interface SwipeCommitOptions {
	/** Fraction of the row's width that counts as a committed swipe. @default 0.3 */
	commitRatio?: number
	/** px/ms speed above which a swipe commits regardless of distance travelled. @default 0.5 */
	velocityThreshold?: number
}

/**
 * Pure decision: does a horizontal drag of `offsetX` px (out of a row `width` px, taking
 * `elapsedMs`) commit to a left/right action, or spring back? No DOM involved.
 */
export function resolveSwipeCommit(
	offsetX: number,
	width: number,
	elapsedMs: number,
	{commitRatio = 0.3, velocityThreshold = 0.5}: SwipeCommitOptions = {},
): SwipeAction | null {
	if (width <= 0 || offsetX === 0) {
		return null
	}
	const ratio = Math.abs(offsetX) / width
	const velocity = elapsedMs > 0 ? Math.abs(offsetX) / elapsedMs : 0
	if (ratio < commitRatio && velocity < velocityThreshold) {
		return null
	}
	return offsetX > 0 ? 'right' : 'left'
}

export interface UseSwipeActionsOptions extends SwipeCommitOptions {
	/** Called once a swipe commits. */
	onCommit: (action: SwipeAction) => void
	/** A committed action that isn't allowed (e.g. no due date to reschedule) just springs back. */
	canCommit?: (action: SwipeAction) => boolean
	/** px of total movement before an axis is decided. @default 10 */
	axisThreshold?: number
	/**
	 * A gesture starting on an element this returns true for is left alone entirely (e.g. a
	 * checkbox, a link, or the list's drag handle — they have their own pointer/touch handling).
	 */
	ignoreStart?: (event: PointerEvent) => boolean
}

/**
 * Wires pointer events on `target` into the swipe-to-act gesture used by mobile task rows:
 * axis-locks onto horizontal vs. vertical early (so vertical list scroll keeps working), tracks a
 * live `offsetX` for the caller to render a reveal/translate effect from, and commits to an action
 * via `resolveSwipeCommit` on release.
 */
export function useSwipeActions(target: Ref<HTMLElement | null>, options: UseSwipeActionsOptions) {
	const offsetX = shallowRef(0)
	const isDragging = shallowRef(false)
	const {vibrate} = useVibrate({pattern: 20})

	let axis: SwipeAxis = null
	let startTime = 0
	let ignored = false

	const {distanceX, distanceY} = usePointerSwipe(target, {
		threshold: 1,
		onSwipeStart: (e) => {
			ignored = options.ignoreStart?.(e) ?? false
			axis = null
			startTime = performance.now()
		},
		onSwipe: () => {
			if (ignored) {
				return
			}

			// distanceX/Y are posStart - posEnd: positive distanceX means the finger moved left.
			const dx = -distanceX.value
			const dy = -distanceY.value

			if (axis === null) {
				axis = resolveSwipeAxis(dx, dy, options.axisThreshold)
			}
			if (axis !== 'horizontal') {
				offsetX.value = 0
				return
			}

			isDragging.value = true
			const width = target.value?.clientWidth ?? 0
			offsetX.value = width > 0 ? Math.max(-width, Math.min(width, dx)) : dx
		},
		onSwipeEnd: () => {
			const width = target.value?.clientWidth ?? 0
			const elapsed = performance.now() - startTime
			const action = !ignored && axis === 'horizontal'
				? resolveSwipeCommit(offsetX.value, width, elapsed, options)
				: null

			if (action && (options.canCommit?.(action) ?? true)) {
				vibrate()
				options.onCommit(action)
			}

			ignored = false
			axis = null
			offsetX.value = 0
			isDragging.value = false
		},
	})

	return {offsetX: computed(() => offsetX.value), isDragging: computed(() => isDragging.value)}
}
