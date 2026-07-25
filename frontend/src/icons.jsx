// Icons extracted from the brand mockups in src/assets/*.svg.
// The path geometry is lifted verbatim from those files (translated to a
// local viewBox), so the shapes stay exactly on-brand. Every icon paints
// with currentColor so the theme decides the color, not the icon.

export function MicIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 36 40" fill="none" aria-hidden="true">
      <rect x="11.7" y="1.8" width="12.6" height="21.6" rx="6.3" fill="currentColor" />
      <g stroke="currentColor" strokeWidth="3.6" strokeLinecap="round">
        <path d="M 3.6 16.2 A 14.4 14.4 0 0 0 32.4 16.2" />
        <line x1="18" y1="30.6" x2="18" y2="37.8" />
        <line x1="10.8" y1="37.8" x2="25.2" y2="37.8" />
      </g>
    </svg>
  )
}

export function PinIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 16 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8">
        <path d="M 8 2 C 12 2 14 5 14 8 C 14 12 8 18 8 18 C 8 18 2 12 2 8 C 2 5 4 2 8 2 Z" />
        <circle cx="8" cy="8" r="2" />
      </g>
    </svg>
  )
}

export function CalendarIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 16 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8">
        <rect x="1" y="5" width="14" height="13" rx="2" />
        <line x1="1" y1="9" x2="15" y2="9" />
        <line x1="4.5" y1="2" x2="4.5" y2="6" />
        <line x1="11.5" y1="2" x2="11.5" y2="6" />
      </g>
    </svg>
  )
}

export function PassengersIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8">
        <circle cx="8" cy="6" r="3" />
        <path d="M 2 16 C 2 11 5 9 8 9 C 11 9 14 11 14 16" />
        <circle cx="16" cy="5" r="2.4" />
        <path d="M 13 16 C 13 12 15 10 17 9.5" />
      </g>
    </svg>
  )
}

export function CheckIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 28 28" fill="none" aria-hidden="true">
      <path
        d="M 7.4 14 L 12.35 19.5 L 20.6 7.4"
        stroke="currentColor"
        strokeWidth="2.64"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </svg>
  )
}

export function SparkleIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 22" fill="none" aria-hidden="true">
      <path
        d="M 10 0.2 L 12.2 6.8 L 19.9 11.2 L 19.9 13.4 L 12.2 11.2 L 11.1 16.7 L 14.4 18.9 L 14.4 20.55 L 10 18.9 L 5.6 20.55 L 5.6 18.9 L 8.9 16.7 L 7.8 11.2 L 0.1 13.4 L 0.1 11.2 L 7.8 6.8 Z"
        fill="currentColor"
      />
    </svg>
  )
}

// Not present in the mockups — drawn to match the line style of the icons
// above (1.8 stroke, currentColor, rounded joins).
export function SpeakerIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M 3 7.5 L 6 7.5 L 10 4 L 10 16 L 6 12.5 L 3 12.5 Z" />
        <path d="M 12.8 7.6 A 3.4 3.4 0 0 1 12.8 12.4" />
        <path d="M 15.2 5.2 A 6.6 6.6 0 0 1 15.2 14.8" />
      </g>
    </svg>
  )
}

export function SpeakerOffIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M 3 7.5 L 6 7.5 L 10 4 L 10 16 L 6 12.5 L 3 12.5 Z" />
        <line x1="13.2" y1="7.6" x2="17.6" y2="12.4" />
        <line x1="17.6" y1="7.6" x2="13.2" y2="12.4" />
      </g>
    </svg>
  )
}

// Line icons in the same style as the rest (1.8 stroke, currentColor).
export function TrainIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <rect x="4.5" y="2.5" width="11" height="12" rx="3" />
        <line x1="4.5" y1="8.5" x2="15.5" y2="8.5" />
        <circle cx="7.5" cy="11.4" r="0.6" fill="currentColor" stroke="none" />
        <circle cx="12.5" cy="11.4" r="0.6" fill="currentColor" stroke="none" />
        <line x1="6.5" y1="14.5" x2="4.5" y2="17.5" />
        <line x1="13.5" y1="14.5" x2="15.5" y2="17.5" />
      </g>
    </svg>
  )
}

export function BusIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 20 20" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <rect x="3" y="3.5" width="14" height="10" rx="2.5" />
        <line x1="3" y1="9" x2="17" y2="9" />
        <circle cx="6.5" cy="15.5" r="1.5" />
        <circle cx="13.5" cy="15.5" r="1.5" />
        <line x1="3" y1="14.5" x2="3" y2="11" />
        <line x1="17" y1="14.5" x2="17" y2="11" />
      </g>
    </svg>
  )
}

export function ExternalLinkIcon({ className }) {
  return (
    <svg className={className} viewBox="0 0 16 18" fill="none" aria-hidden="true">
      <g stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
        <path d="M 6 2 L 13 2 L 13 9" />
        <line x1="13" y1="2" x2="5" y2="10" />
        <path d="M 9 5 L 1 5 L 1 14 L 10 14 L 10 7" />
      </g>
    </svg>
  )
}
