import { useEffect, useMemo, useRef, useState, type FormEvent } from "react"
import { Link, useParams } from "react-router-dom"
import { ArrowLeft, Eye, EyeOff, Plus, Trash2, Pencil, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Badge } from "@/components/ui/badge"
import { api, ApiError } from "@/lib/api"
import { useTextReveal } from "@/lib/anime/use-text-reveal"
import { useStagger } from "@/lib/anime/use-stagger"
import { useScrollReveal } from "@/lib/anime/use-scroll-reveal"

export function SecretsPage() {
  const { slug = "", env = "" } = useParams()
  const [secrets, setSecrets] = useState<Record<string, string>>({})
  const [version, setVersion] = useState<number | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [revealed, setRevealed] = useState<Record<string, boolean>>({})
  const [open, setOpen] = useState(false)
  const [editingKey, setEditingKey] = useState<string | null>(null)
  const [formKey, setFormKey] = useState("")
  const [formValue, setFormValue] = useState("")
  const [saving, setSaving] = useState(false)
  const headingRef = useRef<HTMLHeadingElement>(null)
  const tbodyRef = useRef<HTMLTableSectionElement>(null)
  const tableWrapperRef = useRef<HTMLDivElement>(null)
  useTextReveal(headingRef, env)

  const load = async () => {
    setLoading(true)
    try {
      const res = await api.pullSecrets(slug, env)
      setSecrets(res.secrets ?? {})
      setVersion(res.version)
      setError(null)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to load secrets")
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [slug, env])

  const keys = useMemo(() => Object.keys(secrets).sort(), [secrets])

  useStagger(keys, tbodyRef)
  useScrollReveal(tableWrapperRef)

  const openCreate = () => {
    setEditingKey(null)
    setFormKey("")
    setFormValue("")
    setOpen(true)
  }

  const openEdit = (key: string) => {
    setEditingKey(key)
    setFormKey(key)
    setFormValue(secrets[key] ?? "")
    setOpen(true)
  }

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      await api.setSecrets(slug, env, { [formKey]: formValue })
      setOpen(false)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to save secret")
    } finally {
      setSaving(false)
    }
  }

  const onDelete = async (key: string) => {
    if (!confirm(`Delete secret "${key}"?`)) return
    try {
      await api.deleteSecret(slug, env, key)
      await load()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed to delete secret")
    }
  }

  const toggleReveal = (key: string) => {
    setRevealed((r) => ({ ...r, [key]: !r[key] }))
  }

  return (
    <div className="space-y-6">
      <div>
        <Button asChild variant="ghost" size="sm" className="-ml-3">
          <Link to={`/projects/${slug}`}>
            <ArrowLeft className="mr-2 h-4 w-4" />
            Environments
          </Link>
        </Button>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="flex items-center gap-2">
            <h1 ref={headingRef} className="font-display text-xl text-foreground">
              {env}
            </h1>
            {version !== null && <Badge variant="muted">v{version}</Badge>}
          </div>
          <p className="text-sm text-muted-foreground">
            <span className="font-mono">{slug}</span> — {keys.length} secret
            {keys.length === 1 ? "" : "s"}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={load} disabled={loading}>
            <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} />
            Refresh
          </Button>
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button onClick={openCreate}>
                <Plus className="mr-2 h-4 w-4" />
                Add secret
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{editingKey ? "Update secret" : "Add secret"}</DialogTitle>
                <DialogDescription>
                  Keys must be uppercase with underscores (e.g. DATABASE_URL).
                </DialogDescription>
              </DialogHeader>
              <form onSubmit={onSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="key">Key</Label>
                  <Input
                    id="key"
                    value={formKey}
                    onChange={(e) => setFormKey(e.target.value.toUpperCase())}
                    placeholder="DATABASE_URL"
                    required
                    disabled={editingKey !== null}
                    className="font-mono"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="value">Value</Label>
                  <Input
                    id="value"
                    value={formValue}
                    onChange={(e) => setFormValue(e.target.value)}
                    required
                    className="font-mono"
                  />
                </div>
                <DialogFooter>
                  <Button type="submit" disabled={saving}>
                    {saving ? "Saving…" : "Save"}
                  </Button>
                </DialogFooter>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      {error && (
        <div className="rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading…</p>
      ) : keys.length === 0 ? (
        <div className="py-20 text-center">
          <p className="text-sm text-muted-foreground">No secrets yet</p>
        </div>
      ) : (
        <div ref={tableWrapperRef}>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[35%]">Key</TableHead>
                <TableHead>Value</TableHead>
                <TableHead className="w-[120px] text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody ref={tbodyRef}>
              {keys.map((k) => (
                <TableRow key={k}>
                  <TableCell className="font-mono text-sm">{k}</TableCell>
                  <TableCell className="font-mono text-sm text-muted-foreground">
                    {revealed[k] ? secrets[k] : "••••••••"}
                  </TableCell>
                  <TableCell className="text-right">
                    <div className="flex justify-end gap-0.5">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => toggleReveal(k)}
                      >
                        {revealed[k] ? (
                          <EyeOff className="h-4 w-4" />
                        ) : (
                          <Eye className="h-4 w-4" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => openEdit(k)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => onDelete(k)}
                        className="text-muted-foreground hover:text-destructive"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  )
}
