import { useCallback, useEffect, useState } from 'react'

function getSynth() {
  if (typeof window === 'undefined') return null
  return window.speechSynthesis || null
}

function findPersianVoice(voices) {
  // Match fa, fa-IR, fa_IR, ... but never a non-Persian voice: reading Persian
  // with an English voice sounds far worse than staying silent.
  return voices.find((v) => v.lang && v.lang.toLowerCase().replace('_', '-').startsWith('fa')) || null
}

export function useSpeechSynthesis({ rate = 0.95 } = {}) {
  const synth = getSynth()
  const isSupported = Boolean(synth)

  const [persianVoice, setPersianVoice] = useState(null)

  // getVoices() is empty on first call in most browsers; the list arrives
  // asynchronously via `voiceschanged`, so we read it both ways.
  useEffect(() => {
    if (!synth) return undefined

    function loadVoices() {
      const voice = findPersianVoice(synth.getVoices() || [])
      // Only set on change so we don't re-render on every voiceschanged.
      setPersianVoice((prev) => (prev?.voiceURI === voice?.voiceURI ? prev : voice))
    }

    loadVoices()
    synth.addEventListener('voiceschanged', loadVoices)
    return () => synth.removeEventListener('voiceschanged', loadVoices)
  }, [synth])

  // Never leave speech running after the component goes away.
  useEffect(() => {
    if (!synth) return undefined
    return () => synth.cancel()
  }, [synth])

  const cancel = useCallback(() => {
    if (synth) synth.cancel()
  }, [synth])

  const speak = useCallback(
    (text) => {
      // No Persian voice → stay silent rather than mangle the language.
      if (!synth || !persianVoice || !text) return
      synth.cancel()
      const utterance = new SpeechSynthesisUtterance(text)
      utterance.voice = persianVoice
      utterance.lang = persianVoice.lang || 'fa-IR'
      utterance.rate = rate
      synth.speak(utterance)
    },
    [synth, persianVoice, rate],
  )

  return {
    isSupported,
    hasPersianVoice: Boolean(persianVoice),
    speak,
    cancel,
  }
}
