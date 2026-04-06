import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link } from "react-router-dom"
import { Plus, Trash2, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { api, ApiError, type Project } from "@/lib/api"
import { useTextReveal } from "@/lib/anime/use-text-reveal"
import { useStagger } from "@/lib/anime/use-stagger"
import { useScrollReveal } from "@/lib/anime/use-scroll-reveal"

export function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [open, setOpen] = useState(false)
  const [slug, setSlug] = useState("")
  const [name, setName] = useState("")
  const [creating, setCreating] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  useTextReveal(headingRef, "Projects")
  useStagger(projects, listRef)
  useScrollReveal(listRef)

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.listProjects()
      setProjects(res.projects ?? [])
      setError(null)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load projects")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const onCreate = async (e: FormEvent) => {
    e.preventDefault()
    setCreating(true)
    try {
      await api.createProject(slug, name)
      setOpen(false)
      setSlug("")
      setName("")
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create project")
    } finally {
      setCreating(false)
    }
  }

  const onDelete = async (s: string) => {
    if (!confirm(`Delete project "${s}"? This cannot be undone.`)) return
    try {
      await api.deleteProject(s)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to delete project")
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 ref={headingRef} className="font-display text-xl text-foreground">
          Projects
        </h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              New project
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create project</DialogTitle>
              <DialogDescription>
                Slugs are lowercase, hyphen-separated identifiers.
              </DialogDescription>
            </DialogHeader>
            <form onSubmit={onCreate} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="slug">Slug</Label>
                <Input
                  id="slug"
                  value={slug}
                  onChange={(e) => setSlug(e.target.value)}
                  placeholder="my-app"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  placeholder="My App"
                  required
                />
              </div>
              <DialogFooter>
                <Button type="submit" disabled={creating}>
                  {creating ? "Creating…" : "Create"}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {error && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : projects.length === 0 ? (
        <div className="py-20 text-center">
          <p className="text-sm text-muted-foreground">No projects yet</p>
        </div>
      ) : (
        <div ref={listRef} className="divide-y divide-border rounded-lg border border-border">
          {projects.map((p) => (
            <div key={p.id} className="flex items-center justify-between px-4 py-3">
              <div className="min-w-0">
                <p className="text-sm font-medium text-foreground">{p.name}</p>
                <p className="font-mono text-xs text-muted-foreground">{p.slug}</p>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onDelete(p.slug)}
                  className="text-muted-foreground hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
                <Button asChild variant="ghost" size="icon">
                  <Link to={`/projects/${p.slug}`}>
                    <ArrowRight className="h-4 w-4" />
                  </Link>
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
