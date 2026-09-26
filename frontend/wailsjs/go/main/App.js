// Hand-authored placeholder mirroring the Go App struct's exported methods in app.go 1:1.
// Running `wails dev` or `wails build` regenerates this file automatically to match app.go
// exactly -- keep app.go as the source of truth if the two ever drift.

export function UnlockVault(passphrase) {
  return window.go.main.App.UnlockVault(passphrase)
}

export function LockVault() {
  return window.go.main.App.LockVault()
}

export function IsUnlocked() {
  return window.go.main.App.IsUnlocked()
}

export function ListVaultFiles() {
  return window.go.main.App.ListVaultFiles()
}

export function SelectFileForUpload() {
  return window.go.main.App.SelectFileForUpload()
}

export function EncryptAndUpload(srcPath, targets) {
  return window.go.main.App.EncryptAndUpload(srcPath, targets)
}

export function DecryptAndExport(fileId) {
  return window.go.main.App.DecryptAndExport(fileId)
}

export function DeleteVaultFile(fileId) {
  return window.go.main.App.DeleteVaultFile(fileId)
}

export function StartStream(fileId) {
  return window.go.main.App.StartStream(fileId)
}

export function StopStream(sessionId) {
  return window.go.main.App.StopStream(sessionId)
}

export function GetSettings() {
  return window.go.main.App.GetSettings()
}

export function SaveLaptopSettings(cfg, enabled) {
  return window.go.main.App.SaveLaptopSettings(cfg, enabled)
}

export function CheckLaptopHealth() {
  return window.go.main.App.CheckLaptopHealth()
}

export function StartGoogleOAuth(clientId, clientSecret) {
  return window.go.main.App.StartGoogleOAuth(clientId, clientSecret)
}

export function SaveGDriveTargetFolder(folderId) {
  return window.go.main.App.SaveGDriveTargetFolder(folderId)
}

export function CheckGDriveHealth() {
  return window.go.main.App.CheckGDriveHealth()
}

export function HardenVaultKeyStorage() {
  return window.go.main.App.HardenVaultKeyStorage()
}

export function IsOSKeyProtectionAvailable() {
  return window.go.main.App.IsOSKeyProtectionAvailable()
}
