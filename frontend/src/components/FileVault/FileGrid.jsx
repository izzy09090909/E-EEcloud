import FileCard from './FileCard'

export default function FileGrid({ files, onDecrypt, onDelete, onStream }) {
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
    <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
      {files.map((f) => (
        <FileCard key={f.id} file={f} onDecrypt={onDecrypt} onDelete={onDelete} onStream={onStream} />
      ))}
    </div>
  )
}
