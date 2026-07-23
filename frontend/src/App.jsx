import { useEffect, useReducer, useState } from 'react'
import { parseText } from './api.js'
import { useSpeechRecognition, VOICE_UNAVAILABLE_CODES } from './useSpeechRecognition.js'
import { useSpeechSynthesis } from './useSpeechSynthesis.js'
import { addJalaliDays, buildHotelSearchUrl, lookupHotelCity } from './hotelCities.js'
import {
  CalendarIcon,
  CheckIcon,
  ExternalLinkIcon,
  MicIcon,
  PassengersIcon,
  PinIcon,
  SparkleIcon,
  SpeakerIcon,
  SpeakerOffIcon,
} from './icons.jsx'
import './App.css'

// Pacing. Speech recognition and the local NLU are both near-instant, which
// reads as the app cutting the user off rather than listening to them. These
// two floors trade a little latency for a calmer, more attentive feel.
// GRACE_MS: how long the final heard text stays on screen before we submit,
// so the user gets to see that we caught the whole sentence.
const GRACE_MS = 1300
// MIN_THINKING_MS: floor on how long the spinner shows. It runs alongside the
// request, so a slow API costs nothing extra — it only stops a fast one from
// flashing past.
const MIN_THINKING_MS = 1000

// Phases: idle → listening → settling → processing → (clarifying ↔ listening)
//         → confirming → redirecting, plus error.
const initialState = { phase: 'idle', question: '', error: null, notice: '', heardText: '' }

function reducer(state, action) {
  switch (action.type) {
    case 'LISTENING':
      return { ...state, phase: 'listening', error: null, notice: '', heardText: '' }
    case 'SETTLE':
      return { ...state, phase: 'settling', heardText: action.text, error: null }
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

// Temporary default until the "how many nights?" question lands.
const HOTEL_NIGHTS = 1

// Builds the 780.ir URL from accumulated slots (same schemes as the backend's
// internal/searchurl). Returns null when it can't be built yet — including a
// hotel in a city 780.ir doesn't cover.
function buildSearchUrl(slots) {
  if (!slots) return null
  if (slots.intent === 'hotel_search') return buildHotelSearchUrl(slots, HOTEL_NIGHTS)
  if (slots.intent !== 'flight_search') return null
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
  settling: 'شنیدم',
  processing: 'در حال پردازش',
  clarifying: 'یک سوال کوچیک',
  confirming: 'این درسته؟',
  error: 'مشکلی پیش اومد',
}

const STEP_BY_PHASE = { listening: 0, settling: 0, processing: 0, clarifying: 1, confirming: 2 }

// A short spoken summary for the confirm step — one natural sentence, not a
// readout of every field.
function buildConfirmSpeech(slots) {
  const date = formatJalali(slots.date)
  if (slots.intent === 'hotel_search') {
    const nights = toPersianDigits(HOTEL_NIGHTS)
    return `هتل در ${slots.destination.name}، ورود ${date}، ${nights} شب. درسته؟`
  }
  const people = toPersianDigits(slots.adults ?? 1)
  return `پرواز از ${slots.origin.name} به ${slots.destination.name} در تاریخ ${date} برای ${people} نفر. درسته؟`
}

function TtsToggle({ enabled, onToggle }) {
  return (
    <button
      className="tts-toggle"
      onClick={onToggle}
      aria-label={enabled ? 'خاموش کردن صدای دستیار' : 'روشن کردن صدای دستیار'}
      title={enabled ? 'صدای دستیار روشن است' : 'صدای دستیار خاموش است'}
    >
      {enabled ? <SpeakerIcon className="icon" /> : <SpeakerOffIcon className="icon" />}
    </button>
  )
}

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
    errorCode: speechErrorCode,
    startListening,
    resetTranscript,
  } = useSpeechRecognition({ lang: 'fa-IR' })

  const { hasPersianVoice, speak, cancel: cancelSpeech } = useSpeechSynthesis()
  const [ttsEnabled, setTtsEnabled] = useState(true)

  const [useTextMode, setUseTextMode] = useState(!isSupported)
  const [textValue, setTextValue] = useState('')
  // Accumulated conversation context. null = no conversation in progress.
  const [accumulatedSlots, setAccumulatedSlots] = useState(null)
  // Exact text sent to the NLU (for the debug display).
  const [lastTranscript, setLastTranscript] = useState('')
  // Shown above the text box after we auto-switch away from voice because the
  // device can't do speech recognition.
  const [voiceNotice, setVoiceNotice] = useState('')
  // Raw speech error code, surfaced in the debug section for on-device
  // diagnosis (e.g. reading it off an iPhone).
  const [lastErrorCode, setLastErrorCode] = useState('')

  async function submit(rawText) {
    const trimmed = rawText.trim()
    if (!trimmed) return
    // Covers the typed and suggestion-chip paths the mic handler doesn't.
    cancelSpeech()
    setVoiceNotice('')
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
      // Run the request and the minimum-spinner delay together so the floor
      // never stacks on top of a slow response.
      const [data] = await Promise.all([
        parseText(textToParse),
        new Promise((resolve) => setTimeout(resolve, MIN_THINKING_MS)),
      ])

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

  // A final voice transcript arrived → hold it on screen (settling) instead of
  // submitting straight away, then clear so this effect doesn't re-fire on the
  // emptied value. `settling` is a phase of its own rather than a flag on
  // `listening`, so the silence-recovery effect below can't cut it short.
  useEffect(() => {
    if (transcript) {
      dispatch({ type: 'SETTLE', text: transcript })
      resetTranscript()
    }
  }, [transcript]) // eslint-disable-line react-hooks/exhaustive-deps

  // Grace period: after the pause, submit what we heard. Leaving the phase for
  // any reason (new search, correction, error) cancels the pending submit.
  useEffect(() => {
    if (state.phase !== 'settling') return
    const timer = setTimeout(() => submit(state.heardText), GRACE_MS)
    return () => clearTimeout(timer)
  }, [state.phase, state.heardText]) // eslint-disable-line react-hooks/exhaustive-deps

  // Speech recognition raised an error (mic denied, network, no-speech, ...).
  useEffect(() => {
    if (!speechError) return
    setLastErrorCode(speechErrorCode || '')

    // Device can't do speech at all → don't dead-end on an error screen.
    // Switch to typing, keep any conversation context, and explain why.
    if (VOICE_UNAVAILABLE_CODES.has(speechErrorCode)) {
      setUseTextMode(true)
      setVoiceNotice(speechError)
      dispatch(accumulatedSlots ? { type: 'RESUME_CLARIFY' } : { type: 'RESET' })
      return
    }

    dispatch({ type: 'ERROR', error: speechError })
  }, [speechError]) // eslint-disable-line react-hooks/exhaustive-deps

  // Recognition ended while listening with no transcript: user said nothing.
  // Resume the pending clarification if mid-conversation, else back to idle.
  // Skip when an error is present — that's handled by the error effect above,
  // and onend can fire in the same tick as onerror.
  useEffect(() => {
    if (!speechError && !isListening && state.phase === 'listening' && !transcript) {
      dispatch(accumulatedSlots ? { type: 'RESUME_CLARIFY' } : { type: 'RESET' })
    }
  }, [isListening, state.phase, transcript, accumulatedSlots, speechError])

  // Read out whatever the app is telling the user. Keyed on the message
  // itself, so re-entering a phase with the same text doesn't repeat it, but a
  // new question or error does get spoken.
  useEffect(() => {
    if (!ttsEnabled || !hasPersianVoice) return
    if (state.phase === 'clarifying' && state.question) {
      speak(state.question)
    } else if (state.phase === 'confirming' && accumulatedSlots) {
      speak(buildConfirmSpeech(accumulatedSlots))
    } else if (state.phase === 'error' && state.error) {
      speak(state.error)
    } else if (state.phase === 'idle' && state.notice) {
      speak(state.notice)
    }
  }, [state.phase, state.question, state.error, state.notice, ttsEnabled, hasPersianVoice]) // eslint-disable-line react-hooks/exhaustive-deps

  // Stop speaking before the mic opens, otherwise recognition picks up the
  // assistant's own voice.
  function handleMicClick() {
    cancelSpeech()
    dispatch({ type: 'LISTENING' })
    startListening()
  }

  function toggleTts() {
    const next = !ttsEnabled
    setTtsEnabled(next)
    if (!next) cancelSpeech()
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
    setVoiceNotice('')
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
  // A city we understood, but one 780.ir has no hotel coverage for.
  const hotelCityUnsupported =
    isHotel && !!accumulatedSlots?.destination && !lookupHotelCity(accumulatedSlots.destination.iata)
  const title = TITLE_BY_PHASE[state.phase]
  const step = STEP_BY_PHASE[state.phase]

  function renderInputArea() {
    if (showTextInput) {
      return (
        <>
          {voiceNotice && (
            <div className="notice notice-voice">
              <SparkleIcon className="icon" />
              {voiceNotice}
            </div>
          )}
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
      <div className={`header${state.phase === 'idle' ? ' header-idle' : ''}`}>
        {state.phase === 'idle' ? (
          <span className="brand">دستیار سفر هفت‌هشتاد</span>
        ) : (
          <span className="header-title">{title}</span>
        )}
        {step !== undefined && (
          <span className="dots">
            {[0, 1, 2].map((i) => (
              <span key={i} className={`dot${i === step ? ' active' : ''}`} />
            ))}
          </span>
        )}
        {/* Hidden entirely when no Persian voice exists — the control would
            do nothing. */}
        {hasPersianVoice && <TtsToggle enabled={ttsEnabled} onToggle={toggleTts} />}
      </div>

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

      {state.phase === 'settling' && (
        <div className="mic-stage">
          <div className="mic-wrap">
            <button className="mic-button listening settled" disabled aria-label="شنیده شد">
              <MicIcon className="icon" />
            </button>
          </div>
          <p className="heard-flag">
            <CheckIcon className="icon" />
            شنیدم
          </p>
          <div className="card">
            <p className="transcript-live">{state.heardText}</p>
          </div>
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
                {isHotel ? 'ورود' : 'تاریخ'}
              </span>
              <span className="summary-value">{formatJalali(accumulatedSlots.date)}</span>
            </div>
            {isHotel && (
              <div className="summary-row">
                <span className="summary-label">
                  <CalendarIcon className="icon" />
                  خروج
                </span>
                <span className="summary-value">
                  {formatJalali(addJalaliDays(accumulatedSlots.date, HOTEL_NIGHTS))}
                  <span className="summary-note"> ({toPersianDigits(HOTEL_NIGHTS)} شب)</span>
                </span>
              </div>
            )}
            {!isHotel && (
              <div className="summary-row">
                <span className="summary-label">
                  <PassengersIcon className="icon" />
                  مسافران
                </span>
                <span className="summary-value">
                  {toPersianDigits(accumulatedSlots.adults ?? 1)} نفر
                </span>
              </div>
            )}
          </div>

          {searchUrl ? (
            <button className="btn-primary" onClick={handleSearch}>
              درسته، جستجو کن
            </button>
          ) : (
            <div className="notice">
              <SparkleIcon className="icon" />
              {hotelCityUnsupported
                ? `فعلاً برای هتلِ ${accumulatedSlots.destination.name} پشتیبانی نداریم، ولی به‌زودی اضافه می‌شود.`
                : 'فعلاً امکان جستجو برای این درخواست وجود ندارد.'}
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

      {(lastTranscript || accumulatedSlots || lastErrorCode) && (
        <details className="debug">
          <summary>جزئیات فنی</summary>
          {lastTranscript && <p className="debug-heard">متن تشخیص داده‌شده: «{lastTranscript}»</p>}
          {lastErrorCode && (
            <p className="debug-heard">کد خطای تشخیص گفتار: {lastErrorCode}</p>
          )}
          {accumulatedSlots && <pre>{JSON.stringify(accumulatedSlots, null, 2)}</pre>}
        </details>
      )}
    </div>
  )
}

export default App
