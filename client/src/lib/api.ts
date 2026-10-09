// client/src/lib/api.ts

const API_BASE = '/api/v1'

export interface Job {
  id: string
  media_id: string
  type: string
  status: 'queued' | 'running' | 'completed' | 'failed'
  progress: number
  created_at: string
  source_url: string
}

export interface Media {
  id: string
  title: string
  source: string
  status: 'pending' | 'active' | 'expired'
  size_bytes: number
  storage_key: string
}

export interface InspectItem {
  index: number
  id: string
  type?: 'video' | 'photo'
  title: string
  duration: number
  timestamp: number
  thumbnail: string
}

export interface InspectResult {
  title: string
  count: number
  items: InspectItem[]
}

export async function inspectUrl(url: string): Promise<InspectResult> {
  const res = await fetch(`${API_BASE}/inspect`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ url })
  })
  
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(err.error || 'Failed to inspect URL')
  }
  
  return res.json()
}

export async function createJob(url: string, items?: string[]): Promise<any> {
  const payload: any = { url }
  if (items && items.length > 0) {
    payload.items = items
  }
  const res = await fetch(`${API_BASE}/jobs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload)
  })
  
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(err.error || 'Failed to create job')
  }
  
  return res.json()
}

export async function getJobs(): Promise<{items: Job[]}> {
  const res = await fetch(`${API_BASE}/jobs`)
  if (!res.ok) throw new Error('Failed to fetch jobs')
  return res.json()
}

export async function getVault(): Promise<{items: Media[]}> {
  const res = await fetch(`${API_BASE}/vault`)
  if (!res.ok) throw new Error('Failed to fetch vault')
  return res.json()
}

export async function getHistory(): Promise<{items: Job[]}> {
  const res = await fetch(`${API_BASE}/history`)
  if (!res.ok) throw new Error('Failed to fetch history')
  return res.json()
}

export async function recoverMedia(id: string): Promise<Job> {
  const res = await fetch(`${API_BASE}/vault/${id}/recover`, { method: 'POST' })
  if (!res.ok) throw new Error('Failed to recover media')
  return res.json()
}
