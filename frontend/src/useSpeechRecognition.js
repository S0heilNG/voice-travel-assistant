import { useCallback, useEffect, useRef, useState } from 'react'

const ERROR_MESSAGES = {
  'no-speech': 'صدایی شنیده نشد. لطفاً دوباره تلاش کنید.',
  'not-allowed':
    'دسترسی به میکروفون داده نشد. ممکن است لازم باشد در مرورگر اجازه‌ی دسترسی به میکروفون را بدهید، یا به‌جای صدا تایپ کنید.',
  // On iOS this usually means Dictation is turned off in device settings.
  'service-not-allowed':
    'تشخیص گفتار روی این دستگاه فعال نیست. در تنظیمات آیفون، Dictation را روشن کنید یا از ورودی متنی استفاده کنید.',
  'language-not-supported': 'زبان فارسی روی این دستگاه پشتیبانی نمی‌شود. لطفاً تایپ کنید.',
  'audio-capture': 'میکروفونی پیدا نشد. اتصال میکروفون را بررسی کنید.',
  network: 'خطای شبکه در تشخیص گفتار. اتصال اینترنت را بررسی کنید.',
}

// Error codes that mean speech won't work on this device at all (as opposed to
// a transient/retryable problem). When one of these fires, the UI should fall
// back to text input rather than leaving the user stuck on an error screen.
export const VOICE_UNAVAILABLE_CODES = new Set(['service-not-allowed', 'language-not-supported'])

function getSpeechRecognitionCtor() {
  if (typeof window === 'undefined') return null
  return window.SpeechRecognition || window.webkitSpeechRecognition || null
}

export function useSpeechRecognition({ lang = 'fa-IR' } = {}) {
  const SpeechRecognitionCtor = getSpeechRecognitionCtor()
  const isSupported = Boolean(SpeechRecognitionCtor)

  const recognitionRef = useRef(null)
  const [isListening, setIsListening] = useState(false)
  const [transcript, setTranscript] = useState('')
  const [interimTranscript, setInterimTranscript] = useState('')
  const [error, setError] = useState(null)
  // Raw event.error from the Web Speech API, kept for on-device debugging and
  // so callers can branch on the specific failure.
  const [errorCode, setErrorCode] = useState(null)

  useEffect(() => {
    if (!isSupported) return undefined

    const recognition = new SpeechRecognitionCtor()
    recognition.lang = lang
    recognition.continuous = false
    recognition.interimResults = true

    recognition.onresult = (event) => {
      let finalText = ''
      let interimText = ''
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const chunk = event.results[i][0].transcript
        if (event.results[i].isFinal) {
          finalText += chunk
        } else {
          interimText += chunk
        }
      }
      if (interimText) setInterimTranscript(interimText)
      if (finalText) {
        setInterimTranscript('')
        setTranscript(finalText)
      }
    }

    recognition.onerror = (event) => {
      if (event.error === 'aborted') return
      setErrorCode(event.error || 'unknown')
      setError(ERROR_MESSAGES[event.error] || 'خطای ناشناخته در تشخیص گفتار.')
    }

    recognition.onend = () => {
      setIsListening(false)
    }

    recognitionRef.current = recognition

    // onend still fires after stop(); detach handlers first so it can't
    // touch state of a hook instance that's already being torn down.
    return () => {
      recognition.onresult = null
      recognition.onerror = null
      recognition.onend = null
      recognition.stop()
      recognitionRef.current = null
    }
  }, [isSupported, lang])

  const startListening = useCallback(() => {
    if (!recognitionRef.current || isListening) return
    setTranscript('')
    setInterimTranscript('')
    setError(null)
    setErrorCode(null)
    setIsListening(true)
    try {
      recognitionRef.current.start()
    } catch {
      setIsListening(false)
      setErrorCode('start-failed')
      setError('شروع تشخیص گفتار با خطا مواجه شد.')
    }
  }, [isListening])

  const stopListening = useCallback(() => {
    recognitionRef.current?.stop()
  }, [])

  const resetTranscript = useCallback(() => {
    setTranscript('')
    setInterimTranscript('')
  }, [])

  return {
    isSupported,
    isListening,
    transcript,
    interimTranscript,
    error,
    errorCode,
    startListening,
    stopListening,
    resetTranscript,
  }
}
