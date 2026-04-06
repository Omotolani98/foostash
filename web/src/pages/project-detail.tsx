import { useEffect, useRef, useState, type FormEvent } from "react"
import { Link, useParams } from "react-router-dom"
import { ArrowLeft, Plus, Trash2, ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { api, ApiError, type Environment } from "@/lib/api"
import { useTextReveal } from "@/lib/anime/use-text-reveal"
import { useStagger } from "@/lib/anime/use-stagger"
import { useScrollReveal } from "@/lib/anime/use-scroll-reveal"

export function ProjectDetailPage() {
  const { slug = "" } = useParams()
  const [envs, setEnvs] = useState<Environment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [open, setOpen] = useState(false)
  const [envSlug, setEnvSlug] = useState("")
  const [creating, setCreating] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  useTextReveal(headingRef, slug)
  useStagger(envs, listRef)
  useScrollReveal(listRef)

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.listEnvs(slug)
      setEnvs(res.environments ?? [])
      setError(null)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load environments")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [slug])

  const onCreate = async (e: FormEvent) => {
    e.preventDefault()
    setCreating(true)
    try {
      await api.createEnv(slug, envSlug)
      setOpen(false)
      setEnvSlug("")
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to create environment")
    } finally {
      setCreating(false)
    }
  }

  const onDelete = async (e: string) => {
    if (!confirm(`Delete environment "${e}"? Secrets will be lost.`)) return
    try {
      await api.deleteEnv(slug, e)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to delete environment")
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <Button asChild variant="ghost" size="sm" className="-ml-3">
          <Link to="/projects">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Projects
          </Link>
        </Button>
      </div>

      <div className="flex items-center justify-between">
        <div>
          <h1 ref={headingRef} className="font-display text-xl text-foreground">
            {slug}
          </h1>
          <p className="text-sm text-muted-foreground">Environments</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              New environment
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create environment</DialogTitle>
              <DialogDescription>E.g. dev, staging, production.</DialogDescription>
            </DialogHeader>
            <form onSubmit={onCreate} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="env">Slug</Label>
                <Input
                  id="env"
                  value={envSlug}
                  onChange={(e) => setEnvSlug(e.target.value)}
                  placeholder="production"
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
      ) : envs.length === 0 ? (
        <div className="py-20 text-center">
          <p className="text-sm text-muted-foreground">No environments yet</p>
        </div>
      ) : (
        <div ref={listRef} className="divide-y divide-border rounded-lg border border-border">
          {envs.map((e) => (
            <div key={e.id} className="flex items-center justify-between px-4 py-3">
              <div className="flex items-center gap-3">
                <span className="font-mono text-sm text-foreground">{e.slug}</span>
                <Badge variant="muted">env</Badge>
              </div>
              <div className="flex items-center gap-1">
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={() => onDelete(e.slug)}
                  className="text-muted-foreground hover:text-destructive"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
                <Button asChild variant="ghost" size="icon">
                  <Link to={`/projects/${slug}/${e.slug}`}>
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
