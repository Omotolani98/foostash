import { useEffect, type RefObject } from "react"
import { animate } from "animejs"
import { EASING, SLIDE_DISTANCE } from "./constants"

export function useScrollReveal(ref: RefObject<HTMLElement | null>): void {
  useEffect(() => {
    const el = ref.current
    if (!el) return

    el.style.opacity = "0"

    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries[0]
        if (!entry.isIntersecting) return

        observer.disconnect()

        animate(el, {
          opacity: [0, 1],
          translateY: [SLIDE_DISTANCE, 0],
          duration: 400,
          ease: EASING,
        })
      },
      { threshold: 0.1, rootMargin: "0px 0px -40px 0px" }
    )

    observer.observe(el)

    return () => {
      observer.disconnect()
    }
  }, [ref])
}
