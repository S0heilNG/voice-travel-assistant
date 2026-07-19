import { useState } from 'react'
import { parseText } from './api.js'

// Temporary test harness for the /api/parse connection.
// To be replaced by the real conversational voice UI in later steps.
function App() {
  const [text, setText] = useState('')
  const [result, setResult] = useState(null)
  const [error, setError] = useState(null)
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e) {
    e.preventDefault()
    setLoading(true)
    setError(null)
    setResult(null)
    try {
      const data = await parseText(text)
      setResult(data)
    } catch (err) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 600, margin: '40px auto', padding: '0 16px' }}>
      <h1>دستیار سفر صوتی — تست اتصال API</h1>
      <form onSubmit={handleSubmit} style={{ display: 'flex', gap: 8 }}>
        <input
          type="text"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="مثلاً: بلیط تهران به مشهد برای فردا"
          style={{ flex: 1, padding: 8, fontSize: 16, fontFamily: 'inherit' }}
        />
        <button type="submit" disabled={loading || !text.trim()} style={{ padding: '8px 20px', fontFamily: 'inherit' }}>
          جستجو
        </button>
      </form>

      {loading && <p>در حال پردازش...</p>}
      {error && <p style={{ color: 'crimson' }}>خطا: {error}</p>}
      {result && (
        <pre
          style={{
            marginTop: 20,
            padding: 16,
            background: '#1e1e1e',
            color: '#d4d4d4',
            borderRadius: 8,
            direction: 'ltr',
            textAlign: 'left',
            overflowX: 'auto',
          }}
        >
          {JSON.stringify(result, null, 2)}
        </pre>
      )}
    </div>
  )
}

export default App
