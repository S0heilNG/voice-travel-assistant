import { useEffect, useReducer, useState } from 'react'
import { parseText } from './api.js'
import { useSpeechRecognition } from './useSpeechRecognition.js'
import './App.css'

// Phases: idle → listening → processing → (clarifying ↔ listening) → result | error.
// clarifying is entered when accumulated slots are still missing required fields.
const initialState = { phase: 'idle', question: '', error: null }

function reducer(state, action) {
  switch (action.type) {
    case 'LISTENING':
      return { ...state, phase: 'listening', error: null }
    case 'PROCESSING':
      return { ...state, phase: 'processing', error: null }
    case 'CLARIFY':
      return { ...state, phase: 'clarifying', question: action.question, error: null }
    case 'RESUME_CLARIFY':
      return { ...state, phase: 'clarifying' }
    case 'RESULT':
      return { ...state, phase: 'result', error: null }
    case 'ERROR':
      return { ...state, phase: 'error', error: action.error }
    case 'RESET':
      return initialState
    default:
      return state
  }
}

// --- Slot accumulation (frontend-side context; backend stays stateless) ---

// First utterance seeds every slot; later answers only fill slots that are
// still null, so a follow-up ("از تهران فردا") never overwrites what we
// already know (destination=کیش). Intent is locked from the first utterance.
function mergeSlots(prev, result) {
  if (!prev) {
    return {
      intent: result.intent,
      origin: result.origin,
      destination: result.destination,
      date: result.date,
      adults: result.adults,
    }
  }
  return {
    intent: prev.intent,
    origin: prev.origin ?? result.origin,
    destination: prev.destination ?? result.destination,
    date: prev.date ?? result.date,
    adults: prev.adults ?? result.adults,
  }
}

// Required fields per intent, computed over accumulated slots (mirrors the
// backend's per-parse rule: origin is required only for flights).
function computeMissing(slots) {
  const missing = []
  if (slots.intent === 'flight_search' && !slots.origin) missing.push('origin')
  if (!slots.destination) missing.push('destination')
  if (!slots.date) missing.push('date')
  return missing
}

// Builds the 780.ir flight URL from accumulated slots (same scheme as the
// backend's internal/searchurl). Returns null when it can't be built yet or
// for hotel intent (no 780 URL scheme wired up for hotels yet).
function buildSearchUrl(slots) {
  if (!slots || slots.intent !== 'flight_search') return null
  if (!slots.origin || !slots.destination || !slots.date) return null
  return (
    `https://780.ir/tourism/flights/${slots.origin.iata}-${slots.destination.iata}` +
    `?adult=${slots.adults ?? 1}&child=0&infant=0&departureDate=${slots.date}&sort=lowPrice`
  )
}

// Warm, PRD-toned Persian question covering all missing fields at once.
function buildQuestion(missing, slots) {
  const dest = slots.destination?.name
  const has = (f) => missing.includes(f)

  if (has('origin') && has('destination') && has('date')) {
    return 'خیلی خوب! از کجا، به کجا و چه تاریخی می‌خواید سفر کنید؟'
  }
  if (has('origin') && has('destination')) {
    return 'از کدوم شهر و به کجا می‌خواید پرواز کنید؟'
  }
  if (has('destination') && has('date')) {
    return 'به کجا و چه تاریخی می‌خواید برید؟'
  }
  if (has('origin') && has('date')) {
    return dest
      ? `عالیه! از کدوم شهر و چه تاریخی می‌خواید برید ${dest}؟`
      : 'از کدوم شهر و چه تاریخی می‌خواید سفر کنید؟'
  }
  if (has('origin')) {
    return dest
      ? `عالیه! از کدوم شهر می‌خواید به ${dest} پرواز کنید؟`
      : 'از کدوم شهر پرواز می‌کنید؟'
  }
  if (has('destination')) {
    return 'به کجا می‌خواید برید؟'
  }
  if (has('date')) {
    return dest ? `چه تاریخی می‌خواید به ${dest} برید؟` : 'چه تاریخی می‌خواید سفر کنید؟'
  }
  return 'چه کمکی می‌تونم بکنم؟'
}

function SlotSummary({ slots }) {
  if (!slots) return null
  const chips = []
  if (slots.origin) chips.push(`مبدا: ${slots.origin.name} ✓`)
  if (slots.destination) chips.push(`مقصد: ${slots.destination.name} ✓`)
  if (slots.date) chips.push(`تاریخ: ${slots.date} ✓`)
  if (chips.length === 0) return null
  return (
    <div className="slot-summary">
      {chips.map((c) => (
        <span key={c} className="chip">
          {c}
        </span>
      ))}
    </div>
  )
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
  // Accumulated conversation context. null = no conversation in progress.
  const [accumulatedSlots, setAccumulatedSlots] = useState(null)
  // Exact text sent to the NLU (for the debug display).
  const [lastTranscript, setLastTranscript] = useState('')

  async function submit(rawText) {
    const trimmed = rawText.trim()
    if (!trimmed) return
    setLastTranscript(trimmed)
    dispatch({ type: 'PROCESSING' })

    // Mid-conversation answers are prefixed with the intent keyword so bare
    // replies ("فردا", "پنجشنبه") still trigger intent + date/city extraction
    // in the stateless, single-sentence backend parser.
    const inConversation = accumulatedSlots !== null
    const textToParse = inConversation
      ? `${accumulatedSlots.intent === 'hotel_search' ? 'هتل' : 'بلیط'} ${trimmed}`
      : trimmed

    try {
      const data = await parseText(textToParse)

      // First utterance we can't understand at all → don't enter the loop.
      if (!inConversation && data.intent === 'unknown') {
        dispatch({
          type: 'ERROR',
          error: 'متوجه منظورتون نشدم. لطفاً کامل‌تر بگید، مثلاً: «بلیط تهران به مشهد برای فردا».',
        })
        return
      }

      const merged = mergeSlots(accumulatedSlots, data)
      setAccumulatedSlots(merged)

      const missing = computeMissing(merged)
      if (missing.length === 0) {
        dispatch({ type: 'RESULT' })
      } else {
        dispatch({ type: 'CLARIFY', question: buildQuestion(missing, merged) })
      }
    } catch (err) {
      dispatch({ type: 'ERROR', error: err.message })
    }
  }

  // A final voice transcript arrived → send it, then clear so this effect
  // doesn't re-fire on the emptied value.
  useEffect(() => {
    if (transcript) {
      submit(transcript)
      resetTranscript()
    }
  }, [transcript]) // eslint-disable-line react-hooks/exhaustive-deps

  // Speech recognition raised an error (mic denied, network, no-speech, ...).
  useEffect(() => {
    if (speechError) {
      dispatch({ type: 'ERROR', error: speechError })
    }
  }, [speechError])

  // Recognition ended while listening with no transcript: user said nothing.
  // Resume the pending clarification if mid-conversation, else back to idle.
  useEffect(() => {
    if (!isListening && state.phase === 'listening' && !transcript) {
      dispatch(accumulatedSlots ? { type: 'RESUME_CLARIFY' } : { type: 'RESET' })
    }
  }, [isListening, state.phase, transcript, accumulatedSlots])

  function handleMicClick() {
    dispatch({ type: 'LISTENING' })
    startListening()
  }

  function handleTextSubmit(e) {
    e.preventDefault()
    const value = textValue
    setTextValue('')
    submit(value)
  }

  function handleNewSearch() {
    setAccumulatedSlots(null)
    setLastTranscript('')
    setTextValue('')
    resetTranscript()
    dispatch({ type: 'RESET' })
  }

  function handleErrorRetry() {
    // Preserve a mid-conversation context if we have one; else start fresh.
    if (accumulatedSlots) dispatch({ type: 'RESUME_CLARIFY' })
    else handleNewSearch()
  }

  const showTextInput = !isSupported || useTextMode
  const awaitingInput = state.phase === 'idle' || state.phase === 'clarifying'
  const searchUrl = state.phase === 'result' ? buildSearchUrl(accumulatedSlots) : null
  const showQuestion =
    state.question && ['clarifying', 'listening', 'processing'].includes(state.phase)

  return (
    <div className="app">
      <h1>دستیار سفر صوتی</h1>

      {showQuestion && <p className="question">{state.question}</p>}

      {accumulatedSlots && state.phase !== 'result' && state.phase !== 'idle' && (
        <SlotSummary slots={accumulatedSlots} />
      )}

      {/* Input: mic (voice) or text field */}
      {showTextInput
        ? awaitingInput && (
            <>
              <form className="text-form" onSubmit={handleTextSubmit}>
                <input
                  type="text"
                  value={textValue}
                  onChange={(e) => setTextValue(e.target.value)}
                  placeholder={
                    state.phase === 'clarifying'
                      ? 'پاسخ شما...'
                      : 'مثلاً: بلیط تهران به مشهد برای فردا'
                  }
                />
                <button type="submit" disabled={!textValue.trim()}>
                  {state.phase === 'clarifying' ? 'ادامه' : 'جستجو'}
                </button>
              </form>
              {isSupported && (
                <button className="link-button" onClick={() => setUseTextMode(false)}>
                  یا با صدا صحبت کنید
                </button>
              )}
            </>
          )
        : (awaitingInput || state.phase === 'listening') && (
            <>
              <button
                className={`mic-button${state.phase === 'listening' ? ' listening' : ''}`}
                onClick={handleMicClick}
                disabled={state.phase === 'listening'}
                aria-label="شروع ضبط صدا"
              >
                {state.phase === 'listening' ? '●' : '🎤'}
              </button>
              <div className="status-text">
                {state.phase === 'idle' && 'برای شروع صحبت کنید'}
                {state.phase === 'listening' && 'در حال گوش دادن...'}
                {state.phase === 'clarifying' && 'برای پاسخ، میکروفون را بزنید'}
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
          <p>{state.error}</p>
          <button className="retry-button" onClick={handleErrorRetry}>
            تلاش دوباره
          </button>
        </div>
      )}

      {state.phase === 'result' && (
        <div className="result">
          <p className="heard">متن تشخیص داده‌شده: «{lastTranscript}»</p>
          <SlotSummary slots={accumulatedSlots} />
          {searchUrl ? (
            <button
              className="cta"
              onClick={() => window.open(searchUrl, '_blank', 'noopener')}
            >
              جستجو در ۷۸۰
            </button>
          ) : (
            <p className="missing-hint">
              جستجوی هتل هنوز به ۷۸۰ وصل نشده — در قدم‌های بعدی اضافه می‌شود.
            </p>
          )}
          <button className="retry-button" onClick={handleNewSearch}>
            جستجوی جدید
          </button>
          <pre className="result-json">{JSON.stringify(accumulatedSlots, null, 2)}</pre>
        </div>
      )}
    </div>
  )
}

export default App
