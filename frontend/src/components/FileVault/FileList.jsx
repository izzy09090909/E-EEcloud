function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)} GB`
}

export default function FileList({ files, onDecrypt, onDelete, onStream }) {
  if (!files || files.length === 0) {
    return (
      <div className="flex h-full min-h-[400px] flex-col items-center justify-center text-center">
        <div className="mb-4 flex h-20 w-20 items-center justify-center rounded-2xl bg-drive-light">
          <svg className="h-10 w-10 text-drive-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
        </div>
        <h2 className="text-xl font-medium text-drive-text">No files yet</h2>
        <p className="mt-2 max-w-sm text-sm text-drive-muted">
          Click New to encrypt and upload a file from your computer
        </p>
      </div>
    )
  }

  return (
    <div className="overflow-hidden rounded-xl border border-drive-border">
      <table className="w-full text-left text-sm">
        <thead className="bg-drive-light text-xs font-medium uppercase tracking-wide text-drive-muted">
          <tr>
            <th className="px-4 py-3">Name</th>
            <th className="px-4 py-3">Size</th>
            <th className="px-4 py-3">Targets</th>
            <th className="px-4 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-drive-border">
          {files.map((f) => {
            const name = f.original_name || f.id || 'Untitled'
            const isVideo = ['mp4', 'mkv', 'webm', 'avi', 'mov'].includes(name.split('.').pop()?.toLowerCase() || '')
            return (
              <tr key={f.id} className="hover:bg-drive-hover">
                <td className="px-4 py-3 font-medium text-drive-text">{name}</td>
                <td className="px-4 py-3 text-drive-muted">{formatSize(f.size)}</td>
                <td className="px-4 py-3">
                  <div className="flex gap-1">
                    {f.targets?.gdrive && <span className="rounded-full bg-green-50 px-2 py-0.5 text-[10px] font-medium text-green-700">GDrive</span>}
                    {f.targets?.laptop && <span className="rounded-full bg-blue-50 px-2 py-0.5 text-[10px] font-medium text-blue-700">Laptop</span>}
                  </div>
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="inline-flex gap-1">
                    {isVideo && (
                      <button onClick={() => onStream(f)} className="rounded-full p-1.5 text-drive-muted hover:bg-drive-light hover:text-drive-accent" title="Stream">
                        <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
                      </button>
                    )}
                    <button onClick={() => onDecrypt(f)} className="rounded-full p-1.5 text-drive-muted hover:bg-drive-light hover:text-drive-accent" title="Decrypt">
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"/></svg>
                    </button>
                    <button onClick={() => onDelete(f)} className="rounded-full p-1.5 text-drive-muted hover:bg-red-50 hover:text-red-500" title="Delete">
                      <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2"><path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
                    </button>
                  </div>
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
