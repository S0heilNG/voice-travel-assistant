const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:8080'

export async function parseText(text) {
  const response = await fetch(`${API_BASE}/api/parse`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text }),
  })

  if (!response.ok) {
    const errorBody = await response.json().catch(() => null)
    const message = errorBody?.error || `درخواست با خطا مواجه شد (${response.status})`
    throw new Error(message)
  }

  return response.json()
}
