import { useEffect, useState } from 'react'

type Status = 'loading' | 'ok' | 'down'

function App() {
  const [status, setStatus] = useState<Status>('loading')

  useEffect(() => {
    const controller = new AbortController()
    fetch('/api/health', { signal: controller.signal })
      .then((res) => setStatus(res.ok ? 'ok' : 'down'))
      .catch((err: unknown) => {
        if (!(err instanceof DOMException && err.name === 'AbortError')) {
          setStatus('down')
        }
      })
    return () => controller.abort()
  }, [])

  return (
    <main>
      <h1>Seller Panel</h1>
      <p>Backend: {status}</p>
    </main>
  )
}

export default App
