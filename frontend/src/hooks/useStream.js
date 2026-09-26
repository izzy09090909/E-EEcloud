import { useCallback, useRef, useState } from 'react'
import { StartStream, StopStream } from '../../wailsjs/go/main/App'

// useStream manages a single active loopback streaming session at a time. It always stops the
// previous session before starting a new one, so the Go side never leaks a decrypted session
// (and its FEK held in RAM) when the user switches between videos.
export function useStream() {
  const [url, setUrl] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)
  const sessionRef = useRef(null)

  const play = useCallback(async (fileId) => {
    setLoading(true)
    setError(null)
    try {
      if (sessionRef.current) {
        await StopStream(sessionRef.current)
        sessionRef.current = null
      }
      const handle = await StartStream(fileId)
      sessionRef.current = handle.sessionId
      setUrl(handle.url)
    } catch (err) {
      setError(String(err))
      setUrl(null)
    } finally {
      setLoading(false)
    }
  }, [])

  const stop = useCallback(async () => {
    if (sessionRef.current) {
      await StopStream(sessionRef.current)
      sessionRef.current = null
    }
    setUrl(null)
  }, [])

  return { url, loading, error, play, stop }
}
