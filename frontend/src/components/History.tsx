import { describeCalculation, formatResult } from '../utils/format'
import type { HistoryEntry } from '../types/calculator'

interface HistoryProps {
  // Newest first; the hook caps the list.
  entries: readonly HistoryEntry[]
}

export function History({ entries }: HistoryProps) {
  return (
    <section className="history" aria-labelledby="history-heading">
      <h2 id="history-heading">History</h2>
      {entries.length === 0 ? (
        <p className="history__empty">No calculations yet</p>
      ) : (
        <ol className="history__list">
          {entries.map(({ id, operation, a, b, result }) => (
            <li key={id}>
              {describeCalculation(operation, a, b)} = <strong>{formatResult(result)}</strong>
            </li>
          ))}
        </ol>
      )}
    </section>
  )
}
