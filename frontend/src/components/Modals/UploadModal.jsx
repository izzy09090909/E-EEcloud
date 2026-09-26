import { useEffect, useState } from 'react'
import { SelectFileForUpload, EncryptAndUpload } from '../../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

export default function UploadModal({ open, onClose, onUploaded }) {
  const [srcPath, setSrcPath] = useState('')
  const [targets, setTargets] = useState({ gdrive: true, laptop: true })
  const [progress, setProgress] = useState(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  useEffect(() => {
    if (!open) return
    EventsOn('upload:progress', (evt) => setProgress(evt))
    return () => EventsOff('upload:progress')
  }, [open])

  if (!open) return null

  const pickFile = async () => {
    const path = await SelectFileForUpload()
    if (path) setSrcPath(path)
  }

  const handleClose = () => {
    setSrcPath('')
    setProgress(null)
    setBusy(false)
    setError(null)
    onClose()
  }

  const startUpload = async () => {
    if (!srcPath) return
    setBusy(true)
    setError(null)
    try {
      await EncryptAndUpload(srcPath, targets)
      onUploaded()
      handleClose()
    } catch (err) {
      setError(String(err))
      setBusy(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="w-full max-w-md rounded-2xl border border-drive-border bg-white p-6 shadow-card">
        <h2 className="mb-4 text-lg font-medium text-drive-text">Encrypt & upload</h2>

        <button
          onClick={pickFile}
          disabled={busy}
          className="mb-4 w-full rounded-xl border border-dashed border-drive-border px-4 py-8 text-sm text-drive-muted transition hover:border-drive-accent hover:bg-drive-light"
        >
          {srcPath ? srcPath.split(/[\\/]/).pop() : 'Choose a file...'}
        </button>

        <div className="mb-4 flex gap-6 text-sm text-drive-text">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={targets.gdrive}
              onChange={(e) => setTargets((t) => ({ ...t, gdrive: e.target.checked }))}
              className="h-4 w-4 rounded border-drive-border text-drive-accent focus:ring-drive-accent"
            />
            Google Drive
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={targets.laptop}
              onChange={(e) => setTargets((t) => ({ ...t, laptop: e.target.checked }))}
              className="h-4 w-4 rounded border-drive-border text-drive-accent focus:ring-drive-accent"
            />
            Laptop node
          </label>
        </div>

        {progress && (
          <div className="mb-4">
            <div className="mb-1 flex justify-between text-xs text-drive-muted">
              <span>{progress.stage}{progress.target ? ` · ${progress.target}` : ''}</span>
              <span>{Math.round(progress.percent || 0)}%</span>
            </div>
            <div className="h-1.5 overflow-hidden rounded-full bg-drive-light">
              <div
                className="h-full rounded-full bg-drive-accent transition-all"
                style={{ width: `${progress.percent || 0}%` }}
              />
            </div>
            {progress.message && <p className="mt-1 text-xs text-drive-muted">{progress.message}</p>}
          </div>
        )}

        {error && <p className="mb-4 text-xs text-red-500">{error}</p>}

        <div className="flex justify-end gap-2">
          <button
            onClick={handleClose}
            disabled={busy}
            className="rounded-full px-4 py-2 text-sm font-medium text-drive-muted hover:bg-drive-hover"
          >
            Cancel
          </button>
          <button
            onClick={startUpload}
            disabled={busy || !srcPath || (!targets.gdrive && !targets.laptop)}
            className="rounded-full bg-drive-accent px-5 py-2 text-sm font-medium text-white hover:bg-drive-accentHover disabled:opacity-40"
          >
            {busy ? 'Encrypting...' : 'Upload'}
          </button>
        </div>
      </div>
    </div>
  )
}
