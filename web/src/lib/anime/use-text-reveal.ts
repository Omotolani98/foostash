import { useEffect, type RefObject } from "react"
import { animate, stagger } from "animejs"
import { EASING, TEXT_DURATION, TEXT_STAGGER_DELAY } from "./constants"

export function useTextReveal(
  ref: RefObject<HTMLElement | null>,
  text: string
): void {
  useEffect(() => {
    const el = ref.current
    if (!el || !text) return

    const original = el.textContent

    el.innerHTML = ""
    for (let i = 0; i < text.length; i++) {
      const span = document.createElement("span")
      span.style.display = "inline-block"
      span.style.opacity = "0"
      if (text[i] === " ") {
        span.innerHTML = "&nbsp;"
      } else {
        span.textContent = text[i]
      }
      el.appendChild(span)
    }

    const anim = animate(el.children, {
      opacity: [0, 1],
      translateY: [8, 0],
      duration: TEXT_DURATION,
      ease: EASING,
      delay: stagger(TEXT_STAGGER_DELAY),
    })

    return () => {
      anim.pause()
      if (el) el.textContent = original
    }
  }, [ref, text])
}
