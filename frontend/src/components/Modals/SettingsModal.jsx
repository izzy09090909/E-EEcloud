import { useEffect, useState } from 'react'
import {
  GetSettings,
  SaveLaptopSettings,
  CheckLaptopHealth,
  StartGoogleOAuth,
  SaveGDriveTargetFolder,
  CheckGDriveHealth,
  HardenVaultKeyStorage,
  IsOSKeyProtectionAvailable,
} from '../../../wailsjs/go/main/App'

export default function SettingsModal({ open, onClose }) {
  const [laptopUrl, setLaptopUrl] = useState('')
  const [laptopToken, setLaptopToken] = useState('')
  const [skipTlsVerify, setSkipTlsVerify] = useState(false)
  const [gClientId, setGClientId] = useState('')
  const [gClientSecret, setGClientSecret] = useState('')
  const [gFolderId, setGFolderId] = useState('')
  const [status, setStatus] = useState(null)
  const [osProtection, setOsProtection] = useState(false)

  useEffect(() => {
    if (!open) return
    GetSettings().then((s) => {
      setLaptopUrl(s.laptop?.base_url || '')
      setLaptopToken(s.laptop?.auth_token || '')
      setSkipTlsVerify(!!s.laptop?.skip_tls_verify)
      setGClientId(s.gdrive?.client_id || '')
      setGClientSecret(s.gdrive?.client_secret || '')
      setGFolderId(s.gdrive?.target_folder_id || '')
    })
    IsOSKeyProtectionAvailable().then(setOsProtection)
  }, [open])

  if (!open) return null

  const saveLaptop = async () => {
    setStatus('Saving laptop node...')
    try {
      await SaveLaptopSettings(
        { base_url: laptopUrl, auth_token: laptopToken, skip_tls_verify: skipTlsVerify },
        true
      )
      setStatus('Laptop node saved.')
    } catch (err) {
      setStatus('Error: ' + err)
    }
  }

  const testLaptop = async () => {
    setStatus('Checking laptop node...')
    try {
      await CheckLaptopHealth()
      setStatus('Laptop node reachable.')
    } catch (err) {
      setStatus('Laptop node error: ' + err)
    }
  }

  const connectGoogle = async () => {
    setStatus('Opening browser for Google sign-in...')
    try {
      await StartGoogleOAuth(gClientId, gClientSecret)
      setStatus('Google Drive connected.')
    } catch (err) {
      setStatus('Google OAuth error: ' + err)
    }
  }

  const saveFolder = async () => {
    try {
      await SaveGDriveTargetFolder(gFolderId)
      setStatus('Drive folder updated.')
    } catch (err) {
      setStatus('Error: ' + err)
    }
  }

  const testGoogle = async () => {
    setStatus('Checking Google Drive...')
    try {
      await CheckGDriveHealth()
      setStatus('Google Drive reachable.')
    } catch (err) {
      setStatus('Google Drive error: ' + err)
    }
  }

  const harden = async () => {
    try {
      await HardenVaultKeyStorage()
      setStatus('OS-level key protection enabled.')
    } catch (err) {
      setStatus('Error: ' + err)
    }
  }

  const inputCls =
    'mb-2 w-full rounded-lg border border-drive-border bg-white px-3 py-2 text-sm text-drive-text placeholder-drive-muted focus:border-drive-accent focus:outline-none focus:ring-1 focus:ring-drive-accent'
  const btnSecondary =
    'rounded-full border border-drive-border px-3 py-1.5 text-xs font-medium text-drive-text hover:bg-drive-hover'
  const btnPrimary =
    'rounded-full bg-drive-accent px-3 py-1.5 text-xs font-medium text-white hover:bg-drive-accentHover'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="max-h-[90vh] w-full max-w-lg overflow-y-auto rounded-2xl border border-drive-border bg-white p-6 shadow-card">
        <h2 className="mb-6 text-lg font-medium text-drive-text">Settings</h2>

        <section className="mb-6">
          <h3 className="mb-2 text-sm font-medium text-drive-text">Laptop node</h3>
          <input
            value={laptopUrl}
            onChange={(e) => setLaptopUrl(e.target.value)}
            placeholder="Base URL (e.g. https://192.168.1.10:8443)"
            className={inputCls}
          />
          <input
            value={laptopToken}
            onChange={(e) => setLaptopToken(e.target.value)}
            placeholder="Auth token"
            type="password"
            className={inputCls}
          />
          <label className="mb-3 flex items-center gap-2 text-sm text-drive-muted">
            <input
              type="checkbox"
              checked={skipTlsVerify}
              onChange={(e) => setSkipTlsVerify(e.target.checked)}
              className="h-4 w-4 rounded border-drive-border text-drive-accent"
            />
            Skip TLS verify (dev only)
          </label>
          <div className="flex gap-2">
            <button onClick={saveLaptop} className={btnPrimary}>
              Save
            </button>
            <button onClick={testLaptop} className={btnSecondary}>
              Test connection
            </button>
          </div>
        </section>

        <section className="mb-6">
          <h3 className="mb-2 text-sm font-medium text-drive-text">Google Drive</h3>
          <input
            value={gClientId}
            onChange={(e) => setGClientId(e.target.value)}
            placeholder="OAuth Client ID"
            className={inputCls}
          />
          <input
            value={gClientSecret}
            onChange={(e) => setGClientSecret(e.target.value)}
            placeholder="OAuth Client Secret"
            type="password"
            className={inputCls}
          />
          <button onClick={connectGoogle} className={`${btnPrimary} mb-2 w-full`}>
            Connect Google Drive
          </button>
          <input
            value={gFolderId}
            onChange={(e) => setGFolderId(e.target.value)}
            placeholder="Destination folder ID (optional)"
            className={inputCls}
          />
          <div className="flex gap-2">
            <button onClick={saveFolder} className={btnSecondary}>
              Save folder
            </button>
            <button onClick={testGoogle} className={btnSecondary}>
              Test connection
            </button>
          </div>
        </section>

        <section className="mb-6">
          <h3 className="mb-2 text-sm font-medium text-drive-text">Key protection</h3>
          <p className="mb-2 text-xs text-drive-muted">
            {osProtection
              ? 'Store an extra OS-protected backup of your vault salt (Windows DPAPI / macOS Keychain). Optional — your passphrase alone always unlocks the vault.'
              : 'Not available on this platform (Linux); the vault relies on your passphrase only.'}
          </p>
          <button onClick={harden} disabled={!osProtection} className={`${btnSecondary} disabled:opacity-40`}>
            Enable OS-level protection
          </button>
        </section>

        {status && <p className="mb-4 text-xs text-drive-muted">{status}</p>}

        <div className="flex justify-end">
          <button
            onClick={onClose}
            className="rounded-full border border-drive-border px-4 py-2 text-sm font-medium text-drive-text hover:bg-drive-hover"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  )
}
