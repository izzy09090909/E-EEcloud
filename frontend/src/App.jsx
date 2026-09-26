import { useMemo, useState } from 'react'
import Header from './components/Header'
import Sidebar from './components/Sidebar'
import FileGrid from './components/FileVault/FileGrid'
import FileList from './components/FileVault/FileList'
import UploadModal from './components/Modals/UploadModal'
import DecryptModal from './components/Modals/DecryptModal'
import SettingsModal from './components/Modals/SettingsModal'
import VideoPlayer from './components/Player/VideoPlayer'
import { useVault } from './hooks/useVault'
import { useStream } from './hooks/useStream'

export default function App() {
  const { unlocked, files, error: vaultError, unlock, lock, refreshFiles, removeFile } = useVault()
  const stream = useStream()

  const [passphrase, setPassphrase] = useState('')
  const [unlocking, setUnlocking] = useState(false)
  const [viewMode, setViewMode] = useState('grid')
  const [search, setSearch] = useState('')
  const [targetFilter, setTargetFilter] = useState('all')
  const [activeNav, setActiveNav] = useState('home')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [decryptFile, setDecryptFile] = useState(null)
  const [streamFile, setStreamFile] = useState(null)

  const filteredFiles = useMemo(() => {
    return files.filter((f) => {
      const matchesSearch = f.original_name?.toLowerCase().includes(search.toLowerCase())
      const matchesTarget = targetFilter === 'all' || f.targets?.[targetFilter]
      return matchesSearch && matchesTarget
    })
  }, [files, search, targetFilter])

  const handleUnlock = async (e) => {
    e.preventDefault()
    setUnlocking(true)
    await unlock(passphrase)
    setUnlocking(false)
    setPassphrase('')
  }

  const handleStream = async (file) => {
    setStreamFile(file)
    await stream.play(file.id)
  }

  const closeStream = async () => {
    await stream.stop()
    setStreamFile(null)
  }

  const pageTitle = {
    home: 'Home',
    mydrive: 'My Drive',
    recent: 'Recent',
    trash: 'Trash',
  }[activeNav] || 'Home'

  if (!unlocked) {
    return (
      <div className="flex h-screen items-center justify-center bg-drive-light">
        <form
          onSubmit={handleUnlock}
          className="w-full max-w-sm rounded-2xl border border-drive-border bg-white p-8 shadow-card"
        >
          <div className="mb-6 text-center">
            <div className="mx-auto mb-3 flex h-14 w-14 items-center justify-center rounded-2xl bg-drive-accent">
              <svg width="28" height="28" viewBox="0 0 24 24" fill="none">
                <path d="M12 2L4 5v6.09c0 5.05 3.41 9.76 8 10.91 4.59-1.15 8-5.86 8-10.91V5l-8-3z" fill="white" opacity="0.95"/>
              </svg>
            </div>
            <h1 className="text-xl font-medium text-drive-text">CipherDrive</h1>
            <p className="mt-1 text-sm text-drive-muted">Enter your master passphrase to unlock the vault</p>
          </div>
          <input
            type="password"
            autoFocus
            value={passphrase}
            onChange={(e) => setPassphrase(e.target.value)}
            placeholder="Master passphrase"
            className="mb-4 w-full rounded-lg border border-drive-border bg-white px-3 py-2.5 text-sm text-drive-text placeholder-drive-muted focus:border-drive-accent focus:outline-none focus:ring-1 focus:ring-drive-accent"
          />
          {vaultError && <p className="mb-4 text-xs text-red-500">{vaultError}</p>}
          <button
            type="submit"
            disabled={unlocking || !passphrase}
            className="w-full rounded-full bg-drive-accent px-4 py-2.5 text-sm font-medium text-white transition hover:bg-drive-accentHover disabled:opacity-40"
          >
            {unlocking ? 'Unlocking...' : 'Unlock vault'}
          </button>
        </form>
      </div>
    )
  }

  return (
    <div className="flex h-screen flex-col bg-white">
      <Header
        search={search}
        onSearchChange={setSearch}
        onOpenSettings={() => setSettingsOpen(true)}
      />

      <div className="flex flex-1 overflow-hidden">
        <Sidebar
          activeNav={activeNav}
          onNavChange={setActiveNav}
          viewMode={viewMode}
          onViewModeChange={setViewMode}
          targetFilter={targetFilter}
          onTargetFilterChange={setTargetFilter}
          onUpload={() => setUploadOpen(true)}
          onLock={lock}
        />

        <main className="flex flex-1 flex-col overflow-hidden">
          <div className="flex items-center justify-between px-6 py-4">
            <h1 className="text-2xl font-normal text-drive-text">{pageTitle}</h1>
          </div>
          <div className="flex-1 overflow-y-auto px-6 pb-6">
            {viewMode === 'grid' ? (
              <FileGrid
                files={filteredFiles}
                onDecrypt={setDecryptFile}
                onDelete={(f) => removeFile(f.id)}
                onStream={handleStream}
              />
            ) : (
              <FileList
                files={filteredFiles}
                onDecrypt={setDecryptFile}
                onDelete={(f) => removeFile(f.id)}
                onStream={handleStream}
              />
            )}
          </div>
        </main>
      </div>

      <UploadModal open={uploadOpen} onClose={() => setUploadOpen(false)} onUploaded={refreshFiles} />
      <DecryptModal file={decryptFile} onClose={() => setDecryptFile(null)} />
      <SettingsModal open={settingsOpen} onClose={() => setSettingsOpen(false)} />
      <VideoPlayer
        file={streamFile}
        url={stream.url}
        loading={stream.loading}
        error={stream.error}
        onClose={closeStream}
      />
    </div>
  )
}
