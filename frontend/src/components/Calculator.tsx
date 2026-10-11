import { useEffect } from 'react'
import { useCalculator } from '../hooks/useCalculator'
import { keyToCalculatorKey } from '../hooks/keyboard'
import { Display } from './Display'
import { History } from './History'
import { Keypad } from './Keypad'

// Calculator wires the hook and the keyboard to the presentational components.
export function Calculator() {
  const { display, expression, error, isLoading, pendingOperator, history, press } = useCalculator()

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      // Enter on a focused button activates that button, as browsers normally do.
      if (event.key === 'Enter' && event.target instanceof HTMLButtonElement) return
      const key = keyToCalculatorKey(event)
      if (!key) return
      event.preventDefault()
      press(key)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [press])

  return (
    <div className="calculator">
      <div className="calculator__main">
        <Display value={display} expression={expression} error={error} isLoading={isLoading} />
        <Keypad onPress={press} activeOperator={pendingOperator} disabled={isLoading} />
      </div>
      <History entries={history} />
    </div>
  )
}
