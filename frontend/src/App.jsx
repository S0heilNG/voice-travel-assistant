import { useEffect, useReducer, useState } from 'react'
import { parseText } from './api.js'
import { useSpeechRecognition } from './useSpeechRecognition.js'
import './App.css'

// Phases: idle → listening → processing → result | error.
// From result/error the mic (or retry) sends the user back to idle/listening.
const initialState = { phase: 'idle', result: null, error: null }

function reducer(state, action) {
  switch (action.type) {
    case 'LISTENING':
      return { phase: 'listening', result: null, error: null }
    case 'PROCESSING':
      return { phase: 'processing', result: null, error: null }
    case 'RESULT':
      return { phase: 'result', result: action.result, error: null }
    case 'ERROR':
      return { phase: 'error', result: null, error: action.error }
    case 'RESET':
      return initialState
    default:
      return state
  }
}

// Persian labels for the NLU's `missing` field keys, used both for the
// debug hint here and as groundwork for the clarification loop (next step).
const MISSING_LABELS = {
  origin: 'مبدا',
  destination: 'مقصد',
  date: 'تاریخ',
}

function App() {
  const [state, dispatch] = useReducer(reducer, initialState)
  const {
    isSupported,
    isListening,
    transcript,
    interimTranscript,
    error: speechError,
    startListening,
    resetTranscript,
  } = useSpeechRecognition({ lang: 'fa-IR' })

  const [useTextMode, setUseTextMode] = useState(!isSupported)
  const [textValue, setTextValue] = useState('')
  // The exact text sent to the NLU (from voice or typing), shown in the
  // result so we can see what Web Speech actually heard.
  const [lastTranscript, setLastTranscript] = useState('')

  async function submit(text) {
    const trimmed = text.trim()
    if (!trimmed) return
    setLastTranscript(trimmed)
    dispatch({ type: 'PROCESSING' })
    try {
      const data = await parseText(trimmed)
      dispatch({ type: 'RESULT', result: data })
    } catch (err) {
      dispatch({ type: 'ERROR', error: err.message })
    }
  }

  // A final voice transcript arrived → send it to the NLU, then clear it so
  // this effect doesn't re-fire on the emptied value.
  useEffect(() => {
    if (transcript) {
      submit(transcript)
      resetTranscript()
    }
  }, [transcript]) // eslint-disable-line react-hooks/exhaustive-deps

  // Speech recognition raised an error (mic denied, network, ...).
  useEffect(() => {
    if (speechError) {
      dispatch({ type: 'ERROR', error: speechError })
    }
  }, [speechError])

  // Recognition ended while still in the listening phase with no transcript:
  // the user said nothing. Go back to idle instead of hanging on listening.
  useEffect(() => {
    if (!isListening && state.phase === 'listening' && !transcript) {
      dispatch({ type: 'RESET' })
    }
  }, [isListening, state.phase, transcript])

  function handleMicClick() {
    dispatch({ type: 'LISTENING' })
    startListening()
  }

  function handleTextSubmit(e) {
    e.preventDefault()
    submit(textValue)
  }

  const showTextInput = !isSupported || useTextMode

  return (
    <div className="app">
      <h1>دستیار سفر صوتی</h1>

      {showTextInput ? (
        <>
          <form className="text-form" onSubmit={handleTextSubmit}>
            <input
              type="text"
              value={textValue}
              onChange={(e) => setTextValue(e.target.value)}
              placeholder="مثلاً: بلیط تهران به مشهد برای فردا"
            />
            <button type="submit" disabled={state.phase === 'processing' || !textValue.trim()}>
              جستجو
            </button>
          </form>
          {isSupported && (
            <button className="link-button" onClick={() => setUseTextMode(false)}>
              یا با صدا صحبت کنید
            </button>
          )}
        </>
      ) : (
        <>
          <button
            className={`mic-button${state.phase === 'listening' ? ' listening' : ''}`}
            onClick={handleMicClick}
            disabled={state.phase === 'listening' || state.phase === 'processing'}
            aria-label="شروع ضبط صدا"
          >
            {state.phase === 'listening' ? '●' : '🎤'}
          </button>

          <div className="status-text">
            {state.phase === 'idle' && 'برای شروع صحبت کنید'}
            {state.phase === 'listening' && 'در حال گوش دادن...'}
            {(state.phase === 'result' || state.phase === 'error') && 'برای جستجوی دوباره، میکروفون را بزنید'}
          </div>

          {state.phase === 'listening' && <div className="interim">{interimTranscript}</div>}

          <button className="link-button" onClick={() => setUseTextMode(true)}>
            یا تایپ کنید
          </button>
        </>
      )}

      {state.phase === 'processing' && (
        <>
          <div className="spinner" />
          <div className="status-text">در حال پردازش...</div>
        </>
      )}

      {state.phase === 'error' && (
        <div className="error-box">
          <p>خطا: {state.error}</p>
          <button className="retry-button" onClick={() => dispatch({ type: 'RESET' })}>
            تلاش دوباره
          </button>
        </div>
      )}

      {state.phase === 'result' && (
        <div className="result">
          <p className="heard">متن تشخیص داده‌شده: «{lastTranscript}»</p>
          {state.result.missing && state.result.missing.length > 0 && (
            <p className="missing-hint">
              موارد ناقص: {state.result.missing.map((f) => MISSING_LABELS[f] || f).join('، ')}
            </p>
          )}
          <pre className="result-json">{JSON.stringify(state.result, null, 2)}</pre>
        </div>
      )}
    </div>
  )
}

export default App
