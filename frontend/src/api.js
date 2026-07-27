// Empty string is a meaningful value here: it means "same origin", which is
// how the app is served in production (Nginx serves the built frontend and
// proxies /api/ to the backend). So fall back only when the var is absent —
// `||` would turn that empty string back into localhost.
import { getSessionId } from './session.js'

const API_BASE = import.meta.env.VITE_API_BASE ?? 'http://localhost:8080'

function headers() {
  return {
    'Content-Type': 'application/json',
    // ngrok's free tier shows an HTML interstitial to browsers on first
    // visit; this header opts the request out so we get JSON, not HTML.
    'ngrok-skip-browser-warning': 'true',
    // Anonymous session id, only for grouping this session's analytics.
    'X-Session-Id': getSessionId(),
  }
}

export async function parseText(text) {
  const response = await fetch(`${API_BASE}/api/parse`, {
    method: 'POST',
    headers: headers(),
    body: JSON.stringify({ text }),
  })

  if (!response.ok) {
    const errorBody = await response.json().catch(() => null)
    const message = errorBody?.error || `درخواست با خطا مواجه شد (${response.status})`
    throw new Error(message)
  }

  return response.json()
}

// Fire-and-forget funnel event for analytics. It must never block the UI or
// throw — any failure is silently ignored.
export function sendEvent(type, detail = '') {
  try {
    fetch(`${API_BASE}/api/events`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify({ type, detail }),
      keepalive: true,
    }).catch(() => {})
  } catch {
    /* ignore */
  }
}
