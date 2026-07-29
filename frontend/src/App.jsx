import { useEffect, useReducer, useState } from 'react'
import { parseText, sendEvent } from './api.js'
import { useSpeechRecognition, VOICE_UNAVAILABLE_CODES } from './useSpeechRecognition.js'
import { useSpeechSynthesis } from './useSpeechSynthesis.js'
import { addJalaliDays, buildHotelSearchUrl, lookupHotelCity } from './hotelCities.js'
import { buildTourSearchUrl, lookupTourDestination } from './tourDestinations.js'
import {
  buildBusSearchUrl,
  buildTrainSearchUrl,
  lookupBusCity,
  lookupTrainCity,
} from './transitCities.js'
import {
  BusIcon,
  CalendarIcon,
  CheckIcon,
  ExternalLinkIcon,
  GlobeIcon,
  MicIcon,
  PassengersIcon,
  PinIcon,
  SparkleIcon,
  SpeakerIcon,
  SpeakerOffIcon,
  TourIcon,
  TrainIcon,
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
    case 'HELP':
      return { ...initialState, phase: 'help' }
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
  // The backend asks for a return date only when the user actually requested a
  // round trip, so that request is what we carry forward — a later answer
  // can't re-signal it.
  const askedForReturn = !!result.returnDate || (result.missing ?? []).includes('returnDate')

  if (!prev) {
    return {
      intent: result.intent,
      origin: result.origin,
      destination: result.destination,
      date: result.date,
      returnDate: result.returnDate ?? null,
      wantsReturn: askedForReturn,
      nights: result.nights,
      adults: result.adults,
    }
  }

  // Each utterance is parsed on its own, so a bare "۲۲ مرداد" answering "when
  // do you come back?" arrives as a *departure* date. When the return leg is
  // the only date still missing, that is what it means.
  const answeringReturn = prev.wantsReturn && !prev.returnDate && !!prev.date
  const returnDate =
    prev.returnDate ?? result.returnDate ?? (answeringReturn ? result.date : null)

  return {
    intent: prev.intent,
    origin: prev.origin ?? result.origin,
    destination: prev.destination ?? result.destination,
    date: prev.date ?? result.date,
    returnDate,
    wantsReturn: prev.wantsReturn || askedForReturn,
    // nights is a number: 0 means "not given yet", so keep any positive value
    // we already have and otherwise take the new one.
    nights: prev.nights || result.nights,
    adults: prev.adults ?? result.adults,
  }
}

// Required fields per intent, computed over accumulated slots (mirrors the
// backend's per-parse rule: origin is required only for flights; hotels need
// a nights count).
// Route intents (flight/train/bus) go origin→destination; hotel is a stay.
const ROUTE_INTENTS = ['flight_search', 'international_flight_search', 'train_search', 'bus_search']

// A slot counts as filled when it holds something meaningful. nights is a
// number where 0 means "not given yet", so it can't share the null check.
function hasSlot(slots, field) {
  return field === 'nights' ? slots.nights > 0 : !!slots[field]
}

function computeMissing(slots) {
  const required = SERVICES[slots.intent]?.requires ?? []
  const missing = required.filter((field) => !hasSlot(slots, field))
  // Not in `requires` because it isn't a property of the service: the return
  // leg is only expected once this particular user has asked to come back.
  // Silence means one-way.
  if (slots.wantsReturn && !slots.returnDate) missing.push('returnDate')
  return missing
}

// Builds the 780.ir URL from accumulated slots (same schemes as the backend's
// internal/searchurl). Returns null when it can't be built yet — including a
// hotel in a city 780.ir doesn't cover.
function buildSearchUrl(slots) {
  if (!slots) return null
  if (slots.intent === 'hotel_search') return buildHotelSearchUrl(slots)
  if (slots.intent === 'tour_search') return buildTourSearchUrl(slots)
  if (slots.intent === 'train_search') return buildTrainSearchUrl(slots)
  if (slots.intent === 'bus_search') return buildBusSearchUrl(slots)
  if (slots.intent === 'international_flight_search') return buildIntlFlightSearchUrl(slots)
  if (slots.intent !== 'flight_search') return null
  if (!slots.origin || !slots.destination || !slots.date) return null
  // A city with no confirmed airport has iata "" — it isn't flight-able.
  if (!slots.origin.iata || !slots.destination.iata) return null
  return (
    `https://780.ir/tourism/flights/${slots.origin.iata}-${slots.destination.iata}` +
    `?adult=${slots.adults ?? 1}&child=0&infant=0&departureDate=${slots.date}&sort=lowPrice`
  )
}

// International flights use a different path and parameter set from the
// domestic one. Two details that are easy to get wrong, both verified against
// 780's own bundle: the dates stay Jalali even though the trip leaves Iran,
// and the return parameter is spelled `returningDate` (`returnDate` is only
// 780's internal field name and silently drops the return leg).
//
// The iata values come straight from the backend response, which already
// resolves Tehran to IKA (Imam Khomeini) rather than THR (Mehrabad) for this
// intent — so no airport knowledge is needed here.
function buildIntlFlightSearchUrl(slots) {
  if (!slots.origin || !slots.destination || !slots.date) return null
  if (!slots.origin.iata || !slots.destination.iata) return null
  const tripMode = slots.returnDate ? 2 : 1
  const returning = slots.returnDate ? `&returningDate=${slots.returnDate}` : ''
  return (
    `https://780.ir/tourism/international/${slots.origin.iata}-${slots.destination.iata}` +
    `?departureDate=${slots.date}${returning}` +
    `&cabinType=CABIN_TYPE_ECONOMY&adult=${slots.adults ?? 1}&child=0&infant=0` +
    `&originType=0&destinationType=0&tripMode=${tripMode}&sort=fast`
  )
}

// Warm, PRD-toned Persian question covering all missing fields at once.
function buildQuestion(missing, slots) {
  // Destination is the only slot a tour can be missing, so one line covers it.
  if (slots.intent === 'tour_search') return 'به کجا می‌خواید تور برید؟'
  return slots.intent === 'hotel_search'
    ? buildHotelQuestion(missing, slots)
    : buildRouteQuestion(missing, slots)
}

// Neutral wording (برید/حرکت) so it fits flight, train and bus alike.
function buildRouteQuestion(missing, slots) {
  const dest = slots.destination?.name
  const has = (f) => missing.includes(f)

  // Asked last, once the outbound trip is settled, so the question is only
  // ever about the leg home.
  if (has('returnDate') && missing.length === 1) {
    return dest ? `چه تاریخی از ${dest} برمی‌گردید؟` : 'چه تاریخی برمی‌گردید؟'
  }

  if (has('origin') && has('destination') && has('date')) {
    return 'خیلی خوب! از کجا، به کجا و چه تاریخی می‌خواید سفر کنید؟'
  }
  if (has('origin') && has('destination')) {
    return 'از کدوم شهر و به کجا می‌خواید برید؟'
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
    return dest ? `عالیه! از کدوم شهر می‌خواید به ${dest} برید؟` : 'از کدوم شهر حرکت می‌کنید؟'
  }
  if (has('destination')) {
    return 'به کجا می‌خواید برید؟'
  }
  if (has('date')) {
    return dest ? `چه تاریخی می‌خواید به ${dest} برید؟` : 'چه تاریخی می‌خواید سفر کنید؟'
  }
  return 'چه کمکی می‌تونم بکنم؟'
}

function buildHotelQuestion(missing, slots) {
  const dest = slots.destination?.name
  const has = (f) => missing.includes(f)
  const inDest = dest ? ` در ${dest}` : ''

  if (has('destination') && has('date') && has('nights')) {
    return 'به کدوم شهر، چه تاریخی و چند شب می‌خواید برید؟'
  }
  if (has('destination') && has('date')) {
    return 'به کدوم شهر و چه تاریخی می‌خواید برید؟'
  }
  if (has('destination') && has('nights')) {
    return 'به کدوم شهر و چند شب می‌خواید بمونید؟'
  }
  if (has('date') && has('nights')) {
    return `چه تاریخی و چند شب می‌خواید${inDest} بمونید؟`
  }
  if (has('destination')) {
    return 'به کدوم شهر می‌خواید برید؟'
  }
  if (has('date')) {
    return dest ? `چه تاریخی می‌خواید برید ${dest}؟` : 'چه تاریخی می‌خواید سفر کنید؟'
  }
  if (has('nights')) {
    return `چند شب می‌خواید${inDest} بمونید؟`
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
  { label: 'پرواز خارجی', text: 'پرواز خارجی می‌خوام' },
  { label: 'بلیط قطار', text: 'بلیط قطار می‌خوام' },
  { label: 'بلیط اتوبوس', text: 'بلیط اتوبوس می‌خوام' },
  { label: 'رزرو هتل', text: 'رزرو هتل' },
  { label: 'تور', text: 'تور می‌خوام' },
]

// Shown (and spoken) on the help screen — a greeting or "what can you do?".
const HELP_TEXT =
  'من دستیار سفر هفت‌هشتادم. می‌تونم کمکت کنم بلیط پرواز داخلی و خارجی، قطار، اتوبوس، هتل یا تور پیدا کنی — فقط کافیه بگی کجا و کِی. مثلاً بگو: بلیط تهران به مشهد برای فردا، پرواز تهران به استانبول، یا تور کیش.'

// Per-service labels/icons for the confirm card, spoken summary, and the
// keyword we prefix onto clarification answers so bare replies still parse.
//
// `requires` lists the slots a service needs before it can search. It is data
// rather than a chain of per-intent conditionals because the services genuinely
// disagree: hotels have no origin but need nights, tours need neither an origin
// nor a date. Three exceptions had already accumulated as `if`s in
// computeMissing before this became a field.
const SERVICES = {
  flight_search: {
    title: 'جستجوی پرواز',
    speak: 'پرواز',
    keyword: 'بلیط',
    label: 'پرواز',
    requires: ['origin', 'destination', 'date'],
  },
  international_flight_search: {
    title: 'جستجوی پرواز خارجی',
    speak: 'پرواز خارجی',
    keyword: 'پرواز خارجی',
    label: 'پرواز خارجی',
    requires: ['origin', 'destination', 'date'],
  },
  train_search: {
    title: 'جستجوی قطار',
    speak: 'قطار',
    keyword: 'قطار',
    label: 'قطار',
    requires: ['origin', 'destination', 'date'],
  },
  bus_search: {
    title: 'جستجوی اتوبوس',
    speak: 'اتوبوس',
    keyword: 'اتوبوس',
    label: 'اتوبوس',
    requires: ['origin', 'destination', 'date'],
  },
  hotel_search: {
    title: 'جستجوی هتل',
    speak: 'هتل',
    keyword: 'هتل',
    label: 'هتل',
    requires: ['destination', 'date', 'nights'],
  },
  tour_search: {
    title: 'جستجوی تور',
    speak: 'تور',
    keyword: 'تور',
    label: 'تور',
    requires: ['destination'],
  },
}

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
    const nights = toPersianDigits(slots.nights)
    return `هتل در ${slots.destination.name}، ورود ${date}، ${nights} شب. درسته؟`
  }
  // No date to read out: 780 filters tours by month on the results page.
  if (slots.intent === 'tour_search') {
    return `تور ${slots.destination.name}. درسته؟`
  }
  const service = SERVICES[slots.intent]?.speak ?? 'سفر'
  const people = toPersianDigits(slots.adults ?? 1)
  const back = slots.returnDate ? ` و برگشت ${formatJalali(slots.returnDate)}` : ''
  return `${service} از ${slots.origin.name} به ${slots.destination.name} در تاریخ ${date}${back} برای ${people} نفر. درسته؟`
}

// Returns the name of a city we understood but 780.ir doesn't cover on the
// requested service, or null when everything is supported. For route services
// either endpoint can be the culprit.
function findUnsupportedCity(slots) {
  if (!slots) return null
  const { intent, origin, destination } = slots
  if (intent === 'hotel_search') {
    return destination && !lookupHotelCity(destination.name) ? destination.name : null
  }
  if (intent === 'tour_search') {
    return destination && !lookupTourDestination(destination.name) ? destination.name : null
  }
  // Route services: a city is supported if it has this service's slug (train/
  // bus) or a confirmed airport (flight — cities with no airport have iata "").
  const supported =
    intent === 'train_search'
      ? (c) => !!lookupTrainCity(c.name)
      : intent === 'bus_search'
        ? (c) => !!lookupBusCity(c.name)
        : intent === 'flight_search' || intent === 'international_flight_search'
          ? (c) => !!c.iata
          : null
  if (!supported) return null
  if (origin && !supported(origin)) return origin.name
  if (destination && !supported(destination)) return destination.name
  return null
}

function ServiceIcon({ intent, className }) {
  if (intent === 'international_flight_search') return <GlobeIcon className={className} />
  if (intent === 'train_search') return <TrainIcon className={className} />
  if (intent === 'bus_search') return <BusIcon className={className} />
  if (intent === 'tour_search') return <TourIcon className={className} />
  return <SparkleIcon className={className} />
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
    const keyword = SERVICES[accumulatedSlots?.intent]?.keyword ?? 'بلیط'
    const textToParse = inConversation ? `${keyword} ${trimmed}` : trimmed

    try {
      // Run the request and the minimum-spinner delay together so the floor
      // never stacks on top of a slow response.
      const [data] = await Promise.all([
        parseText(textToParse),
        new Promise((resolve) => setTimeout(resolve, MIN_THINKING_MS)),
      ])

      // An answer to a clarification question (vs. a fresh first utterance).
      if (inConversation) sendEvent('clarification_answered')

      // Greeting or "what can you do?" → friendly help screen, not a search.
      if (data.intent === 'help') {
        setAccumulatedSlots(null)
        sendEvent('help_shown')
        dispatch({ type: 'HELP' })
        return
      }

      // First utterance we can't understand at all → don't enter the loop.
      if (!inConversation && data.intent === 'unknown') {
        sendEvent('error_shown', 'unknown_intent')
        dispatch({
          type: 'ERROR',
          error:
            'متوجه منظورتون نشدم. من می‌تونم بلیط پرواز، قطار، اتوبوس یا هتل پیدا کنم — مثلاً بگید: «بلیط تهران به مشهد برای فردا».',
        })
        return
      }

      const merged = mergeSlots(accumulatedSlots, data)
      setAccumulatedSlots(merged)

      const missing = computeMissing(merged)
      if (missing.length === 0) {
        sendEvent('confirm_shown', merged.intent)
        dispatch({ type: 'CONFIRM' })
      } else {
        sendEvent('clarification_shown', missing.join(','))
        dispatch({ type: 'CLARIFY', question: buildQuestion(missing, merged) })
      }
    } catch (err) {
      sendEvent('error_shown', 'network')
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
    // Raw Web Speech code — the key signal for diagnosing the iOS issue.
    sendEvent('voice_error', speechErrorCode || 'unknown')

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
    } else if (state.phase === 'help') {
      speak(HELP_TEXT)
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
    sendEvent('new_search')
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
    // The key success event: the user actually went to 780.
    sendEvent('search_clicked', accumulatedSlots?.intent ?? '')
    if (searchUrl) window.open(searchUrl, '_blank', 'noopener')
    dispatch({ type: 'REDIRECT' })
  }

  // Partial slot editing needs machinery we don't have yet, so correcting
  // simply starts the conversation over.
  function handleCorrect() {
    sendEvent('correction_clicked', accumulatedSlots?.intent ?? '')
    setAccumulatedSlots(null)
    setLastTranscript('')
    setTextValue('')
    resetTranscript()
    dispatch({ type: 'RESTART', notice: 'باشه، از نو بگید.' })
  }

  const showTextInput = !isSupported || useTextMode
  const searchUrl = buildSearchUrl(accumulatedSlots)
  const isHotel = accumulatedSlots?.intent === 'hotel_search'
  const isTour = accumulatedSlots?.intent === 'tour_search'
  // Only route services have an origin to show alongside the destination.
  const isRoute = ROUTE_INTENTS.includes(accumulatedSlots?.intent)
  const service = SERVICES[accumulatedSlots?.intent]
  // A city we understood, but one 780.ir doesn't cover on this service (its
  // name, for the friendly message). null when everything is supported.
  const unsupportedCityName = findUnsupportedCity(accumulatedSlots)
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
      <div className={`header${state.phase === 'idle' || state.phase === 'help' ? ' header-idle' : ''}`}>
        {state.phase === 'idle' || state.phase === 'help' ? (
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
          <p className="privacy-note">
            نسخه‌ی آزمایشی — برای بهتر شدن دستیار، تعامل‌ها به‌صورت ناشناس ثبت می‌شود.
          </p>
        </>
      )}

      {state.phase === 'help' && (
        <>
          <p className="help-text">{HELP_TEXT}</p>
          <div className="suggestions">
            {SUGGESTIONS.map((s) => (
              <button key={s.label} className="suggestion" onClick={() => submit(s.text)}>
                {s.label}
              </button>
            ))}
          </div>
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
              <ServiceIcon intent={accumulatedSlots.intent} className="icon" />
              <span>{service?.title ?? 'جستجو'}</span>
            </div>
            <div className="summary-row">
              <span className="summary-label">
                <PinIcon className="icon" />
                {isRoute ? 'مسیر' : 'مقصد'}
              </span>
              <span className="summary-value">
                {isRoute
                  ? `${accumulatedSlots.origin.name} ← ${accumulatedSlots.destination.name}`
                  : accumulatedSlots.destination.name}
              </span>
            </div>
            {/* Tours carry no departure date — 780 filters them by month on
                the results page — so the date row is skipped entirely. */}
            {!isTour && (
              <div className="summary-row">
                <span className="summary-label">
                  <CalendarIcon className="icon" />
                  {isHotel ? 'ورود' : 'تاریخ'}
                </span>
                <span className="summary-value">{formatJalali(accumulatedSlots.date)}</span>
              </div>
            )}
            {isHotel && (
              <div className="summary-row">
                <span className="summary-label">
                  <CalendarIcon className="icon" />
                  خروج
                </span>
                <span className="summary-value">
                  {formatJalali(addJalaliDays(accumulatedSlots.date, accumulatedSlots.nights))}
                  <span className="summary-note"> ({toPersianDigits(accumulatedSlots.nights)} شب)</span>
                </span>
              </div>
            )}
            {accumulatedSlots.returnDate && (
              <div className="summary-row">
                <span className="summary-label">
                  <CalendarIcon className="icon" />
                  برگشت
                </span>
                <span className="summary-value">
                  {formatJalali(accumulatedSlots.returnDate)}
                </span>
              </div>
            )}
            {!isHotel && !isTour && (
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
            <>
              {/* TEMPORARY (see CLAUDE.md): 780's hotel results page ignores
                  URL dates, so we open the city landing page and ask the user
                  to pick the dates — which we already know — there. */}
              {isHotel && (
                <div className="notice">
                  <SparkleIcon className="icon" />
                  {`مقصد رو توی ۷۸۰ برات باز می‌کنم — فقط تاریخ ورود «${formatJalali(
                    accumulatedSlots.date,
                  )}» تا «${formatJalali(
                    addJalaliDays(accumulatedSlots.date, accumulatedSlots.nights),
                  )}» رو همون‌جا انتخاب کن.`}
                </div>
              )}
              <button className="btn-primary" onClick={handleSearch}>
                {isHotel ? 'باشه، مقصد رو باز کن' : 'درسته، جستجو کن'}
              </button>
            </>
          ) : (
            <div className="notice">
              <SparkleIcon className="icon" />
              {unsupportedCityName
                ? `فعلاً برای ${service?.label ?? 'این سرویس'}ِ ${unsupportedCityName} پشتیبانی نداریم، ولی به‌زودی اضافه می‌شود.`
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
