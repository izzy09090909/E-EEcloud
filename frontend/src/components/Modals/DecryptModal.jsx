import { useState } from 'react'
import { DecryptAndExport } from '../../../wailsjs/go/main/App'

export default function DecryptModal({ file, onClose }) {
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)
  const [done, setDone] = useState(false)

  if (!file) return null

  const handleDecrypt = async () => {
    setBusy(true)
    setError(null)
    try {
      await DecryptAndExport(file.id)
      setDone(true)
    } catch (err) {
      setError(String(err))
    } finally {
      setBusy(false)
    }
  }

  const handleClose = () => {
    setBusy(false)
    setError(null)
    setDone(false)
    onClose()
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-2xl border border-drive-border bg-white p-6 shadow-card">
        <h2 className="mb-2 text-lg font-medium text-drive-text">Decrypt & export</h2>
        <p className="mb-4 text-sm text-drive-muted truncate" title={file.original_name}>
          {file.original_name || file.id}
        </p>

        {done ? (
          <p className="mb-4 text-sm text-green-600">File decrypted and saved successfully.</p>
        ) : (
          <>
            {error && <p className="mb-4 text-xs text-red-500">{error}</p>}
            <p className="mb-4 text-xs text-drive-muted">
              The file will be decrypted client-side and saved via the native save dialog.
            </p>
          </>
        )}

        <div className="flex justify-end gap-2">
          <button
            onClick={handleClose}
            className="rounded-full px-4 py-2 text-sm font-medium text-drive-muted hover:bg-drive-hover"
          >
            {done ? 'Close' : 'Cancel'}
          </button>
          {!done && (
            <button
              onClick={handleDecrypt}
              disabled={busy}
              className="rounded-full bg-drive-accent px-5 py-2 text-sm font-medium text-white hover:bg-drive-accentHover disabled:opacity-40"
            >
              {busy ? 'Decrypting...' : 'Decrypt'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
