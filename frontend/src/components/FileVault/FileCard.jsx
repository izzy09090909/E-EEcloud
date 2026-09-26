function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

function fileIcon(name = '') {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const video = ['mp4', 'mkv', 'webm', 'avi', 'mov']
  const image = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg']
  const doc = ['pdf', 'doc', 'docx', 'txt', 'md']
  if (video.includes(ext)) {
    return (
      <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-red-50 text-red-500">
        <svg className="h-6 w-6" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
      </div>
    )
  }
  if (image.includes(ext)) {
    return (
      <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-purple-50 text-purple-500">
        <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8"><path strokeLinecap="round" strokeLinejoin="round" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>
      </div>
    )
  }
  if (doc.includes(ext)) {
    return (
      <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-50 text-blue-500">
        <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8"><path strokeLinecap="round" strokeLinejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
      </div>
    )
  }
  return (
    <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-drive-light text-drive-muted">
      <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8"><path strokeLinecap="round" strokeLinejoin="round" d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"/></svg>
    </div>
  )
}

export default function FileCard({ file, onDecrypt, onDelete, onStream }) {
  const name = file.original_name || file.id || 'Untitled'
  const isVideo = ['mp4', 'mkv', 'webm', 'avi', 'mov'].includes(name.split('.').pop()?.toLowerCase() || '')

  return (
    <div className="group relative flex flex-col rounded-xl border border-drive-border bg-white p-4 transition hover:border-drive-accent/40 hover:shadow-card">
      <div className="mb-3 flex items-start justify-between">
        {fileIcon(name)}
        <div className="flex gap-1 opacity-0 transition group-hover:opacity-100">
          {isVideo && (
            <button
              onClick={() => onStream(file)}
              className="rounded-full p-1.5 text-drive-muted hover:bg-drive-hover hover:text-drive-accent"
              title="Stream"
            >
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
            </button>
          )}
          <button
            onClick={() => onDecrypt(file)}
            className="rounded-full p-1.5 text-drive-muted hover:bg-drive-hover hover:text-drive-accent"
            title="Decrypt & download"
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
          </button>
          <button
            onClick={() => onDelete(file)}
            className="rounded-full p-1.5 text-drive-muted hover:bg-red-50 hover:text-red-500"
            title="Delete"
          >
            <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
          </button>
        </div>
      </div>
      <p className="truncate text-sm font-medium text-drive-text" title={name}>{name}</p>
      <p className="mt-1 text-xs text-drive-muted">{formatSize(file.size)}</p>
      {file.targets && (
        <div className="mt-2 flex flex-wrap gap-1">
          {file.targets.gdrive && (
            <span className="rounded-full bg-green-50 px-2 py-0.5 text-[10px] font-medium text-green-700">GDrive</span>
          )}
          {file.targets.laptop && (
            <span className="rounded-full bg-blue-50 px-2 py-0.5 text-[10px] font-medium text-blue-700">Laptop</span>
          )}
        </div>
      )}
    </div>
  )
}
