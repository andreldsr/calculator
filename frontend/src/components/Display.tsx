interface DisplayProps {
  value: string
  expression: string
  error: string | null
  isLoading: boolean
}

export function Display({ value, expression, error, isLoading }: DisplayProps) {
  return (
    <section className="display" aria-label="Display" aria-busy={isLoading}>
      <p className="display__expression">{expression}</p>
      <output className="display__value" aria-live="polite" aria-atomic="true">
        {value}
      </output>
      {/* Always rendered so screen readers pick up the error when it appears. */}
      <p className="display__error" role="alert">
        {error}
      </p>
    </section>
  )
}
