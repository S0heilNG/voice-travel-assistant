import { useEffect, useReducer, useState } from 'react'
import { parseText } from './api.js'
import { useSpeechRecognition } from './useSpeechRecognition.js'
import {
  CalendarIcon,
  CheckIcon,
  ExternalLinkIcon,
  MicIcon,
  PassengersIcon,
  PinIcon,
  SparkleIcon,
} from './icons.jsx'
import './App.css'

// Phases: idle → listening → processing → (clarifying ↔ listening)
//         → confirming → redirecting, plus error.
const initialState = { phase: 'idle', question: '', error: null, notice: '' }

function reducer(state, action) {
  switch (action.type) {
    case 'LISTENING':
      return { ...state, phase: 'listening', error: null, notice: '' }
    case 'PROCESSING':
      return { ...state, phase: 'processing', error: null, notice: '' }
    case 'CLARIFY':
      return { ...state, phase: 'clarifying', question: action.question, error: null }
    case 'RESUME_CLARIFY':
      return { ...state, phase: 'clarifying' }
    case 'CONFIRM':
      return { ...state, phase: 'confirming', error: null }
    case 'REDIRECT':
      return { ...state, phase: 'redirecting', error: null }
    case 'ERROR':
      return { ...state, phase: 'error', error: action.error }
    case 'RESET':
      return initialState
    case 'RESTART':
      return { ...initialState, notice: action.notice }
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

// --- Display helpers ---

const JALALI_MONTHS = [
  'فروردین',
  'اردیبهشت',
  'خرداد',
  'تیر',
  'مرداد',
  'شهریور',
  'مهر',
  'آبان',
  'آذر',
  'دی',
  'بهمن',
  'اسفند',
]

function toPersianDigits(value) {
  return String(value).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[Number(d)])
}

// "1405-04-30" → "۳۰ تیر ۱۴۰۵"
function formatJalali(date) {
  if (!date) return ''
  const [y, m, d] = date.split('-').map(Number)
  const month = JALALI_MONTHS[m - 1]
  if (!month) return toPersianDigits(date)
  return `${toPersianDigits(d)} ${month} ${toPersianDigits(y)}`
}

// Suggestion chips: the visible label follows the mockup, but the submitted
// text is chosen so it always parses (a bare "سفر آخر هفته" comes back as
// intent=unknown and would dead-end on the error screen).
const SUGGESTIONS = [
  { label: 'بلیط هواپیما', text: 'بلیط هواپیما می‌خوام' },
  { label: 'رزرو هتل', text: 'رزرو هتل' },
  { label: 'سفر آخر هفته', text: 'بلیط برای آخر هفته' },
]

const TITLE_BY_PHASE = {
  listening: 'در حال گوش دادن',
  processing: 'در حال پردازش',
  clarifying: 'یک سوال کوچیک',
  confirming: 'این درسته؟',
  error: 'مشکلی پیش اومد',
}

const STEP_BY_PHASE = { listening: 0, processing: 0, clarifying: 1, confirming: 2 }

function SlotChips({ slots }) {
  if (!slots) return null
  const chips = []
  if (slots.origin) chips.push(`مبدا: ${slots.origin.name} ✓`)
  if (slots.destination) chips.push(`مقصد: ${slots.destination.name} ✓`)
  if (slots.date) chips.push(`تاریخ: ${formatJalali(slots.date)} ✓`)
  if (chips.length === 0) return null
  return (
    <div className="chips">
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
        dispatch({ type: 'CONFIRM' })
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

  // Opened synchronously inside the click so the popup blocker treats it as
  // a user gesture; the redirect screen is then shown as confirmation.
  function handleSearch() {
    if (searchUrl) window.open(searchUrl, '_blank', 'noopener')
    dispatch({ type: 'REDIRECT' })
  }

  // Partial slot editing needs machinery we don't have yet, so correcting
  // simply starts the conversation over.
  function handleCorrect() {
    setAccumulatedSlots(null)
    setLastTranscript('')
    setTextValue('')
    resetTranscript()
    dispatch({ type: 'RESTART', notice: 'باشه، از نو بگید.' })
  }

  const showTextInput = !isSupported || useTextMode
  const searchUrl = buildSearchUrl(accumulatedSlots)
  const isHotel = accumulatedSlots?.intent === 'hotel_search'
  const title = TITLE_BY_PHASE[state.phase]
  const step = STEP_BY_PHASE[state.phase]

  function renderInputArea() {
    if (showTextInput) {
      return (
        <>
          <form className="text-form" onSubmit={handleTextSubmit}>
            <input
              type="text"
              value={textValue}
              onChange={(e) => setTextValue(e.target.value)}
              placeholder={
                state.phase === 'clarifying' ? 'پاسخ شما...' : 'مثلاً: بلیط تهران به مشهد برای فردا'
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
    }
    return (
      <>
        <div className="mic-wrap">
          <button className="mic-button" onClick={handleMicClick} aria-label="شروع ضبط صدا">
            <MicIcon className="icon" />
          </button>
        </div>
        <p className="hint">
          {state.phase === 'clarifying'
            ? 'برای پاسخ، دکمه رو بزن و صحبت کن'
            : 'برای شروع، دکمه رو بزن و صحبت کن'}
        </p>
        <button className="link-button" onClick={() => setUseTextMode(true)}>
          یا تایپ کنید
        </button>
      </>
    )
  }

  return (
    <div className="app">
      {state.phase === 'idle' ? (
        <p className="brand">دستیار سفر هفت‌هشتاد</p>
      ) : (
        title && (
          <div className="header">
            <span className="header-title">{title}</span>
            {step !== undefined && (
              <span className="dots">
                {[0, 1, 2].map((i) => (
                  <span key={i} className={`dot${i === step ? ' active' : ''}`} />
                ))}
              </span>
            )}
          </div>
        )
      )}

      {state.phase === 'idle' && (
        <>
          <h1 className="greeting">سلام! امروز کجا می‌خوای بری؟</h1>
          <div className="suggestions">
            {SUGGESTIONS.map((s) => (
              <button key={s.label} className="suggestion" onClick={() => submit(s.text)}>
                {s.label}
              </button>
            ))}
          </div>
          {state.notice && (
            <div className="notice" style={{ marginTop: 'var(--sp-4)' }}>
              <SparkleIcon className="icon" />
              {state.notice}
            </div>
          )}
          <div className="mic-stage">{renderInputArea()}</div>
        </>
      )}

      {state.phase === 'listening' && (
        <div className="mic-stage">
          <div className="mic-wrap">
            <button className="mic-button listening" disabled aria-label="در حال گوش دادن">
              <MicIcon className="icon" />
            </button>
          </div>
          <div className="waveform">
            {Array.from({ length: 22 }).map((_, i) => (
              <span key={i} style={{ animationDelay: `${(i % 7) * 0.12}s` }} />
            ))}
          </div>
          <div className="card">
            <p className="transcript-live">{interimTranscript || '...'}</p>
          </div>
          <p className="hint">حرف بزن — وقتی تموم شد، خودکار متوقف می‌شه</p>
        </div>
      )}

      {state.phase === 'processing' && (
        <div className="mic-stage">
          <div className="spinner" />
          <p className="status-text">در حال پردازش...</p>
          <SlotChips slots={accumulatedSlots} />
        </div>
      )}

      {state.phase === 'clarifying' && (
        <>
          <p className="question">{state.question}</p>
          <SlotChips slots={accumulatedSlots} />
          <div className="mic-stage">{renderInputArea()}</div>
        </>
      )}

      {state.phase === 'confirming' && accumulatedSlots && (
        <>
          <div className="summary-card">
            <div className="summary-head">
              <SparkleIcon className="icon" />
              <span>{isHotel ? 'جستجوی هتل' : 'جستجوی پرواز'}</span>
            </div>
            <div className="summary-row">
              <span className="summary-label">
                <PinIcon className="icon" />
                {isHotel ? 'مقصد' : 'مسیر'}
              </span>
              <span className="summary-value">
                {isHotel
                  ? accumulatedSlots.destination.name
                  : `${accumulatedSlots.origin.name} ← ${accumulatedSlots.destination.name}`}
              </span>
            </div>
            <div className="summary-row">
              <span className="summary-label">
                <CalendarIcon className="icon" />
                تاریخ
              </span>
              <span className="summary-value">{formatJalali(accumulatedSlots.date)}</span>
            </div>
            <div className="summary-row">
              <span className="summary-label">
                <PassengersIcon className="icon" />
                مسافران
              </span>
              <span className="summary-value">
                {toPersianDigits(accumulatedSlots.adults ?? 1)} نفر
              </span>
            </div>
          </div>

          {searchUrl ? (
            <button className="btn-primary" onClick={handleSearch}>
              درسته، جستجو کن
            </button>
          ) : (
            <div className="notice">
              <SparkleIcon className="icon" />
              جستجوی هتل هنوز به ۷۸۰ وصل نشده — در قدم‌های بعدی اضافه می‌شود.
            </div>
          )}
          <button className="btn-secondary" onClick={handleCorrect}>
            اصلاح کن
          </button>
        </>
      )}

      {state.phase === 'redirecting' && (
        <div className="redirect-stage">
          <div className="check-badge">
            <CheckIcon className="icon" />
          </div>
          <p className="redirect-title">در حال انتقال به ۷۸۰</p>
          <p className="redirect-sub">
            نتایج جستجو در تب جدید باز شد.
            <br />
            اگه باز نشد، از لینک زیر استفاده کن.
          </p>
          {searchUrl && (
            <a className="url-chip" href={searchUrl} target="_blank" rel="noopener noreferrer">
              <ExternalLinkIcon className="icon" />
              780.ir
            </a>
          )}
          <button className="btn-secondary" onClick={handleNewSearch}>
            جستجوی جدید
          </button>
        </div>
      )}

      {state.phase === 'error' && (
        <div className="error-box">
          <p>{state.error}</p>
          <button className="btn-secondary" onClick={handleErrorRetry}>
            تلاش دوباره
          </button>
        </div>
      )}

      {(lastTranscript || accumulatedSlots) && (
        <details className="debug">
          <summary>جزئیات فنی</summary>
          {lastTranscript && <p className="debug-heard">متن تشخیص داده‌شده: «{lastTranscript}»</p>}
          <pre>{JSON.stringify(accumulatedSlots, null, 2)}</pre>
        </details>
      )}
    </div>
  )
}

export default App
