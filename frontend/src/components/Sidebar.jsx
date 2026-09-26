export default function Sidebar({
  activeNav,
  onNavChange,
  viewMode,
  onViewModeChange,
  targetFilter,
  onTargetFilterChange,
  onUpload,
  onLock,
}) {
  const navItems = [
    { id: 'home', label: 'Home', icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6" />
      </svg>
    )},
    { id: 'mydrive', label: 'My Drive', icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8">
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
      </svg>
    )},
    { id: 'recent', label: 'Recent', icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8">
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    )},
    { id: 'trash', label: 'Trash', icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8">
        <path strokeLinecap="round" strokeLinejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
      </svg>
    )},
  ]

  return (
    <aside className="flex w-60 shrink-0 flex-col border-r border-drive-border bg-white">
      {/* New button */}
      <div className="px-3 pt-3 pb-2">
        <button
          onClick={onUpload}
          className="flex w-full items-center gap-3 rounded-2xl border border-drive-border bg-white px-4 py-3 text-sm font-medium text-drive-text shadow-card transition hover:bg-drive-hover hover:shadow-md"
        >
          <svg className="h-5 w-5 text-drive-accent" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="2.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          New
        </button>
      </div>

      {/* Navigation */}
      <nav className="flex-1 px-2 py-2">
        <ul className="space-y-0.5">
          {navItems.map((item) => (
            <li key={item.id}>
              <button
                onClick={() => onNavChange(item.id)}
                className={`flex w-full items-center gap-3 rounded-full px-4 py-2.5 text-sm font-medium transition ${
                  activeNav === item.id
                    ? 'bg-drive-active text-drive-accent'
                    : 'text-drive-text hover:bg-drive-hover'
                }`}
              >
                <span className={activeNav === item.id ? 'text-drive-accent' : 'text-drive-muted'}>
                  {item.icon}
                </span>
                {item.label}
              </button>
            </li>
          ))}
        </ul>

        {/* Storage filter (kept for functionality) */}
        <div className="mt-6 px-2">
          <p className="mb-2 px-2 text-xs font-medium uppercase tracking-wide text-drive-muted">Storage</p>
          <div className="space-y-0.5">
            {[
              { id: 'all', label: 'All targets' },
              { id: 'gdrive', label: 'Google Drive' },
              { id: 'laptop', label: 'Laptop node' },
            ].map((t) => (
              <button
                key={t.id}
                onClick={() => onTargetFilterChange(t.id)}
                className={`w-full rounded-full px-4 py-2 text-left text-sm transition ${
                  targetFilter === t.id
                    ? 'bg-drive-active text-drive-accent font-medium'
                    : 'text-drive-muted hover:bg-drive-hover'
                }`}
              >
                {t.label}
              </button>
            ))}
          </div>
        </div>

        {/* View mode */}
        <div className="mt-4 px-2">
          <p className="mb-2 px-2 text-xs font-medium uppercase tracking-wide text-drive-muted">View</p>
          <div className="flex gap-1 px-1">
            <button
              onClick={() => onViewModeChange('grid')}
              className={`flex-1 rounded-lg px-2 py-1.5 text-xs font-medium ${
                viewMode === 'grid' ? 'bg-drive-active text-drive-accent' : 'text-drive-muted hover:bg-drive-hover'
              }`}
            >
              Grid
            </button>
            <button
              onClick={() => onViewModeChange('list')}
              className={`flex-1 rounded-lg px-2 py-1.5 text-xs font-medium ${
                viewMode === 'list' ? 'bg-drive-active text-drive-accent' : 'text-drive-muted hover:bg-drive-hover'
              }`}
            >
              List
            </button>
          </div>
        </div>
      </nav>

      {/* Lock vault at bottom */}
      <div className="border-t border-drive-border p-3">
        <button
          onClick={onLock}
          className="flex w-full items-center gap-3 rounded-full px-4 py-2.5 text-sm font-medium text-drive-muted transition hover:bg-drive-hover hover:text-drive-text"
        >
          <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth="1.8">
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          Lock vault
        </button>
      </div>
    </aside>
  )
}
