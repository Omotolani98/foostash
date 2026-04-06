import { useEffect, useRef, type RefObject } from "react"
import { animate } from "animejs"
import { DURATION, EASING, SLIDE_DISTANCE } from "./constants"

export function usePageTransition(deps: unknown[]): RefObject<HTMLDivElement | null> {
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    el.style.opacity = "0"
    el.style.transform = `translateY(${SLIDE_DISTANCE}px)`

    const anim = animate(el, {
      opacity: [0, 1],
      translateY: [SLIDE_DISTANCE, 0],
      duration: DURATION,
      ease: EASING,
    })

    return () => {
      anim.pause()
    }
  }, deps)

  return ref
}
