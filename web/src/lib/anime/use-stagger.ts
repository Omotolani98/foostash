import { useEffect, useRef, type RefObject } from "react"
import { animate, stagger } from "animejs"
import { DURATION, EASING, STAGGER_DELAY } from "./constants"

export function useStagger<T>(
  items: T[],
  containerRef: RefObject<HTMLElement | null>
): void {
  const prevCount = useRef(0)

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    const children = el.children
    if (items.length > 0 && prevCount.current === 0 && children.length > 0) {
      for (let i = 0; i < children.length; i++) {
        ;(children[i] as HTMLElement).style.opacity = "0"
      }

      const anim = animate(children, {
        opacity: [0, 1],
        translateY: [8, 0],
        duration: DURATION,
        ease: EASING,
        delay: stagger(STAGGER_DELAY),
      })

      prevCount.current = items.length
      return () => {
        anim.pause()
      }
    }

    prevCount.current = items.length
  }, [items.length, containerRef])
}
