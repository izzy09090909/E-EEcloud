export default function VideoPlayer({ file, url, loading, error, onClose }) {
  if (!file) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70">
      <div className="relative w-full max-w-4xl rounded-2xl bg-black p-2 shadow-card">
        <div className="mb-2 flex items-center justify-between px-2">
          <p className="truncate text-sm text-white">{file.original_name || file.id}</p>
          <button
            onClick={onClose}
            className="rounded-full p-1.5 text-white/70 hover:bg-white/10 hover:text-white"
          >
            <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        {loading && (
          <div className="flex h-64 items-center justify-center text-sm text-white/70">Loading stream...</div>
        )}
        {error && (
          <div className="flex h-64 items-center justify-center text-sm text-red-400">{error}</div>
        )}
        {url && !loading && (
          <video src={url} controls autoPlay className="w-full rounded-lg max-h-[70vh]" />
        )}
      </div>
    </div>
  )
}
