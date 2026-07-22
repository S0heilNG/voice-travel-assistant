// Empty string is a meaningful value here: it means "same origin", which is
// how the app is served in production (Nginx serves the built frontend and
// proxies /api/ to the backend). So fall back only when the var is absent —
// `||` would turn that empty string back into localhost.
const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

export async function parseText(text) {
  const response = await fetch(`${API_BASE}/api/parse`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      // ngrok's free tier shows an HTML interstitial to browsers on first
      // visit; this header opts the request out so we get JSON, not HTML.
      'ngrok-skip-browser-warning': 'true',
    },
    body: JSON.stringify({ text }),
  })

  if (!response.ok) {
    const errorBody = await response.json().catch(() => null)
    const message = errorBody?.error || `درخواست با خطا مواجه شد (${response.status})`
    throw new Error(message)
  }

  return response.json()
}
