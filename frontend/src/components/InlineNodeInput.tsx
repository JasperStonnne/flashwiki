import { useEffect, useRef, useState, type KeyboardEvent } from 'react'

interface InlineNodeInputProps {
  defaultValue: string
  onConfirm: (value: string) => void
  onCancel: () => void
}

export function InlineNodeInput({ defaultValue, onCancel, onConfirm }: InlineNodeInputProps) {
  const [value, setValue] = useState(defaultValue)
  const inputRef = useRef<HTMLInputElement>(null)
  const finishedRef = useRef(false)

  useEffect(() => {
    inputRef.current?.focus()
    inputRef.current?.select()
  }, [])

  function finish(nextValue: string) {
    if (finishedRef.current) {
      return
    }

    finishedRef.current = true
    const trimmed = nextValue.trim()
    if (trimmed === '') {
      onCancel()
      return
    }

    onConfirm(trimmed)
  }

  function cancel() {
    if (finishedRef.current) {
      return
    }

    finishedRef.current = true
    onCancel()
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    event.stopPropagation()
    if (event.key === 'Enter') {
      event.preventDefault()
      finish(value)
      return
    }

    if (event.key === 'Escape') {
      event.preventDefault()
      cancel()
    }
  }

  return (
    <input
      autoFocus
      className="inline-node-input"
      onBlur={() => finish(value)}
      onChange={(event) => setValue(event.target.value)}
      onClick={(event) => event.stopPropagation()}
      onContextMenu={(event) => event.stopPropagation()}
      onKeyDown={handleKeyDown}
      ref={inputRef}
      value={value}
    />
  )
}
