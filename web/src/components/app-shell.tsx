import { Link } from "react-router-dom"
import { LogOut } from "lucide-react"
import { Button } from "@/components/ui/button"
import { AnimatedOutlet } from "@/components/animated-outlet"
import { useAuth } from "@/lib/auth"

export function AppShell() {
  const { session, logout } = useAuth()

  return (
    <div className="min-h-screen">
      <header className="border-b border-border bg-card">
        <div className="mx-auto flex max-w-5xl items-center justify-between px-6 py-3">
          <Link to="/projects" className="font-display text-lg text-primary">
            foostash
          </Link>
          <div className="flex items-center gap-3">
            <span className="hidden text-sm text-muted-foreground sm:inline">
              {session?.email}
            </span>
            <Button variant="ghost" size="sm" onClick={logout}>
              <LogOut className="mr-2 h-4 w-4" />
              Sign out
            </Button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-8">
        <AnimatedOutlet />
      </main>
    </div>
  )
}
