import { Outlet, useLocation } from "react-router-dom"
import { usePageTransition } from "@/lib/anime/use-page-transition"

export function AnimatedOutlet() {
  const location = useLocation()
  const ref = usePageTransition([location.pathname])

  return (
    <div ref={ref} key={location.pathname}>
      <Outlet />
    </div>
  )
}
