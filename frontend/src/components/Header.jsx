export default function Header({ search, onSearchChange, onOpenSettings }) {
  return (
    <header className="flex h-16 shrink-0 items-center gap-4 border-b border-drive-border bg-white px-4">
      {/* Logo */}
      <div className="flex items-center gap-2 min-w-[200px]">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-drive-accent">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L4 5v6.09c0 5.05 3.41 9.76 8 10.91 4.59-1.15 8-5.86 8-10.91V5l-8-3z" fill="white" opacity="0.9"/>
            <path d="M12 12.5c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0-5c-1.66 0-3 1.34-3 3s1.34 3 3 3 3-1.34 3-3-1.34-3-3-3z" fill="white"/>
          </svg>
        </div>
        <span className="text-xl font-medium text-drive-text tracking-tight">CipherDrive</span>
      </div>

      {/* Search bar */}
      <div className="flex flex-1 justify-center max-w-2xl mx-auto">
        <div className="relative w-full">
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-4">
            <svg className="h-5 w-5 text-drive-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
              <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
            </svg>
          </div>
          <input
            type="text"
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search in Drive"
            className="w-full rounded-full border-0 bg-drive-light py-2.5 pl-12 pr-4 text-sm text-drive-text placeholder-drive-muted shadow-search focus:bg-white focus:outline-none focus:ring-1 focus:ring-drive-accent"
          />
        </div>
      </div>

      {/* Right actions */}
      <div className="flex items-center gap-1 min-w-[120px] justify-end">
        <button
          onClick={onOpenSettings}
          className="flex h-10 w-10 items-center justify-center rounded-full text-drive-muted hover:bg-drive-hover"
          title="Settings"
        >
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2">
            <path strokeLinecap="round" strokeLinejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </button>
        <button
          className="flex h-10 w-10 items-center justify-center rounded-full text-drive-muted hover:bg-drive-hover"
          title="View options"
        >
          <svg className="h-5 w-5" fill="currentColor" viewBox="0 0 24 24">
            <path d="M3 3h8v8H3V3zm10 0h8v8h-8V3zM3 13h8v8H3v-8zm10 0h8v8h-8v-8z"/>
          </svg>
        </button>
        <div className="ml-1 flex h-9 w-9 items-center justify-center rounded-full bg-drive-accent text-sm font-medium text-white">
          S
        </div>
      </div>
    </header>
  )
}
