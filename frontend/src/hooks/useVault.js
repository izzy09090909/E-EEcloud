import { useCallback, useEffect, useState } from 'react'
import {
  UnlockVault,
  LockVault,
  IsUnlocked,
  ListVaultFiles,
  DeleteVaultFile,
} from '../../wailsjs/go/main/App'

// useVault centralizes vault lock state and the file list so any component can trigger a
// refresh (e.g. after an upload completes) without prop-drilling callbacks everywhere.
export function useVault() {
  const [unlocked, setUnlocked] = useState(false)
  const [files, setFiles] = useState([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(null)

  const refreshFiles = useCallback(async () => {
    setLoading(true)
    try {
      const list = await ListVaultFiles()
      setFiles(list || [])
      setError(null)
    } catch (err) {
      setError(String(err))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    IsUnlocked()
      .then(setUnlocked)
      .catch(() => setUnlocked(false))
  }, [])

  useEffect(() => {
    if (unlocked) refreshFiles()
  }, [unlocked, refreshFiles])

  const unlock = useCallback(async (passphrase) => {
    setError(null)
    try {
      await UnlockVault(passphrase)
      setUnlocked(true)
      return true
    } catch (err) {
      setError(String(err))
      return false
    }
  }, [])

  const lock = useCallback(async () => {
    await LockVault()
    setUnlocked(false)
    setFiles([])
  }, [])

  const removeFile = useCallback(
    async (fileId) => {
      await DeleteVaultFile(fileId)
      await refreshFiles()
    },
    [refreshFiles]
  )

  return { unlocked, files, loading, error, unlock, lock, refreshFiles, removeFile }
}
