import { useEffect, useId, useRef, useState, type FormEvent, type KeyboardEvent } from 'react'
import { MAX_STOCK, stockErrorMessage, updateStock, type StockLevel } from './api'
import './stock.css'

interface Props {
  productId: number
  stock: number
  reserved: number
  /** Archived products cannot change stock; the editor is shown read-only. */
  disabled?: boolean
  /** Called with the backend's new level after a successful save. */
  onSaved: (level: StockLevel) => void
}

type Draft = { ok: true; value: number } | { ok: false }

function parse(text: string): Draft {
  if (!/^\d+$/.test(text.trim())) return { ok: false }
  const value = Number(text.trim())
  return value <= MAX_STOCK ? { ok: true, value } : { ok: false }
}

/**
 * Inline stock editor for a product list row. The seller types a number or
 * uses −/+, then saves with Enter or the Save button; Escape cancels. Nothing
 * is sent until the seller saves, so clicking + five times is one request.
 */
export function StockEditor({ productId, stock, reserved, disabled = false, onSaved }: Props) {
  const [text, setText] = useState(String(stock))
  const [shownStock, setShownStock] = useState(stock)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const abortRef = useRef<AbortController | null>(null)
  const errorId = useId()

  // When the row gets a new stock from outside (list reload, order accepted),
  // replace the draft so the editor never shows a stale number.
  if (stock !== shownStock) {
    setShownStock(stock)
    setText(String(stock))
    setError(null)
  }

  useEffect(() => () => abortRef.current?.abort(), [])

  const draft = parse(text)
  const changed = draft.ok && draft.value !== stock
  const belowReserved = draft.ok && draft.value < reserved
  const canSave = changed && !belowReserved && !saving && !disabled

  function step(delta: number) {
    const base = draft.ok ? draft.value : stock
    const next = Math.min(MAX_STOCK, Math.max(reserved, base + delta))
    setText(String(next))
    setError(null)
  }

  function cancel() {
    abortRef.current?.abort()
    setText(String(stock))
    setError(null)
    setSaving(false)
  }

  async function save(e?: FormEvent) {
    e?.preventDefault()
    if (!canSave || !draft.ok) return
    abortRef.current?.abort()
    const controller = new AbortController()
    abortRef.current = controller
    setSaving(true)
    setError(null)
    try {
      const level = await updateStock(productId, draft.value, controller.signal)
      onSaved(level)
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return
      setError(stockErrorMessage(err))
    } finally {
      if (abortRef.current === controller) setSaving(false)
    }
  }

  function onKeyDown(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Escape') cancel()
  }

  let hint: string | null = null
  if (!draft.ok) hint = `Whole number, 0 to ${MAX_STOCK.toLocaleString()}`
  else if (belowReserved) hint = `At least ${reserved} (reserved by orders)`
  const message = error ?? hint

  return (
    <form className="stock-editor" onSubmit={save} aria-busy={saving}>
      <div className="stock-editor__row">
        <button
          type="button"
          className="stock-editor__step"
          onClick={() => step(-1)}
          disabled={disabled || saving || (draft.ok && draft.value <= reserved)}
          aria-label="Decrease stock by 1"
        >
          −
        </button>
        <input
          className="stock-editor__input"
          type="text"
          inputMode="numeric"
          value={text}
          onChange={(e) => {
            setText(e.target.value)
            setError(null)
          }}
          onKeyDown={onKeyDown}
          disabled={disabled || saving}
          aria-label="Stock quantity"
          aria-invalid={message !== null}
          aria-describedby={message ? errorId : undefined}
        />
        <button
          type="button"
          className="stock-editor__step"
          onClick={() => step(1)}
          disabled={disabled || saving || (draft.ok && draft.value >= MAX_STOCK)}
          aria-label="Increase stock by 1"
        >
          +
        </button>
        {changed && !disabled && (
          <>
            <button type="submit" className="stock-editor__save" disabled={!canSave}>
              {saving ? 'Saving…' : 'Save'}
            </button>
            <button type="button" className="stock-editor__cancel" onClick={cancel}>
              Cancel
            </button>
          </>
        )}
      </div>
      {reserved > 0 && <div className="stock-editor__meta">{reserved} reserved</div>}
      {message && (
        <div id={errorId} className="stock-editor__error" role={error ? 'alert' : undefined}>
          {message}
        </div>
      )}
    </form>
  )
}
