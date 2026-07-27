// A random, non-identifying id used only to group one browser session's events
// during the limited-user test. It carries NO personal data — just a random
// value — and lives in sessionStorage, so it resets when the tab closes.
export function getSessionId() {
  let id = sessionStorage.getItem('vta_session')
  if (!id) {
    id = crypto.randomUUID ? crypto.randomUUID() : `s-${Date.now()}-${Math.random().toString(36).slice(2)}`
    sessionStorage.setItem('vta_session', id)
  }
  return id
}
