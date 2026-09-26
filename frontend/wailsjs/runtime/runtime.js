// Hand-authored placeholder re-exporting window.runtime.* (injected by the Wails webview at
// runtime). Regenerated automatically by `wails dev` / `wails build`.

export function EventsOn(eventName, callback) {
  return window.runtime.EventsOn(eventName, callback)
}

export function EventsOff(eventName) {
  return window.runtime.EventsOff(eventName)
}

export function EventsEmit(eventName, ...data) {
  return window.runtime.EventsEmit(eventName, ...data)
}

export function BrowserOpenURL(url) {
  return window.runtime.BrowserOpenURL(url)
}

export function LogInfo(message) {
  return window.runtime.LogInfo(message)
}
