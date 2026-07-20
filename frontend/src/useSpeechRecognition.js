import { useCallback, useEffect, useRef, useState } from 'react'

const ERROR_MESSAGES = {
  'no-speech': 'صدایی شنیده نشد. لطفاً دوباره تلاش کنید.',
  'not-allowed': 'دسترسی به میکروفون رد شد. لطفاً از تنظیمات مرورگر اجازه دسترسی بدهید.',
  'audio-capture': 'میکروفونی پیدا نشد. اتصال میکروفون را بررسی کنید.',
  network: 'خطای شبکه در تشخیص گفتار. اتصال اینترنت را بررسی کنید.',
}

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
    setIsListening(true)
    try {
      recognitionRef.current.start()
    } catch {
      setIsListening(false)
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
    startListening,
    stopListening,
    resetTranscript,
  }
}
