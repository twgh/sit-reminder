// Bridge 通信层：前端与 Go 后端之间的 JS Bridge

export interface AppConfig {
  intervalMinutes: number
  ringtonePath: string
  activityRingtonePath: string
  activityMinutes: number
  volume: number
  autoHide: boolean
}

export type TimerStatus = "idle" | "running" | "paused" | "finished"
export type TimerFinishType = "sedentary" | "snooze" | "activity"
export type TimerType = "sedentary" | "snooze" | "activity"

declare global {
  interface Window {
    api: {
      // 窗口控制
      minimize: () => Promise<void>
      close: () => Promise<void>
      // 定时器
      startTimer: () => Promise<void>
      stopTimer: () => Promise<void>
      pauseTimer: () => Promise<void>
      resumeTimer: () => Promise<void>
      setInterval: (minutes: number) => Promise<void>
      snooze: (minutes: number) => Promise<void>
      startActivityTimer: () => Promise<void>
      startActivityTimerWithMinutes: (minutes: number) => Promise<void>
      // 提醒铃声
      selectRingtone: () => Promise<string>
      testRingtone: () => Promise<void>
      stopRingtone: () => Promise<void>
      // 活动铃声
      selectActivityRingtone: () => Promise<string>
      testActivityRingtone: () => Promise<void>
      // 配置
      getConfig: () => Promise<AppConfig>
      saveConfig: () => Promise<void>
      setActivityMinutes: (minutes: number) => Promise<void>
      setVolume: (volume: number) => Promise<void>
      setAutoHide: (enabled: boolean) => Promise<void>
      // 系统
      getVersion: () => Promise<string>
    }
    __timerTick: (remainingSeconds: number, totalSeconds: number) => void
    __timerFinished: (type: TimerFinishType) => void
    __statusChanged: (status: TimerStatus) => void
    __configLoaded: (config: AppConfig) => void
    __activityStarted: (minutes: number) => void
    __timerTypeChanged: (type: TimerType) => void
    __showSettings: () => void
  }
}

export function isBridgeAvailable(): boolean {
  return typeof window.api !== "undefined"
}

async function callApi<T>(method: string, ...args: unknown[]): Promise<T | null> {
  if (!isBridgeAvailable()) {
    console.warn(`[Bridge] API "${method}" not available (dev mode)`)
    return null
  }
  try {
    const fn = (window.api as unknown as Record<string, (...args: unknown[]) => Promise<T>>)[method]
    if (!fn) {
      console.warn(`[Bridge] Method "${method}" not found`)
      return null
    }
    return await fn(...args)
  } catch (err) {
    console.error(`[Bridge] Error calling "${method}":`, err)
    return null
  }
}

export const bridge = {
  minimize: () => callApi<void>("minimize"),
  close: () => callApi<void>("close"),
  startTimer: () => callApi<void>("startTimer"),
  stopTimer: () => callApi<void>("stopTimer"),
  pauseTimer: () => callApi<void>("pauseTimer"),
  resumeTimer: () => callApi<void>("resumeTimer"),
  setInterval: (minutes: number) => callApi<void>("setInterval", minutes),
  snooze: (minutes: number) => callApi<void>("snooze", minutes),
  startActivityTimer: () => callApi<void>("startActivityTimer"),
  startActivityTimerWithMinutes: (minutes: number) => callApi<void>("startActivityTimerWithMinutes", minutes),
  selectRingtone: () => callApi<string>("selectRingtone"),
  testRingtone: () => callApi<void>("testRingtone"),
  stopRingtone: () => callApi<void>("stopRingtone"),
  selectActivityRingtone: () => callApi<string>("selectActivityRingtone"),
  testActivityRingtone: () => callApi<void>("testActivityRingtone"),
  getConfig: () => callApi<AppConfig>("getConfig"),
  saveConfig: () => callApi<void>("saveConfig"),
  setActivityMinutes: (minutes: number) => callApi<void>("setActivityMinutes", minutes),
  setVolume: (volume: number) => callApi<void>("setVolume", volume),
  setAutoHide: (enabled: boolean) => callApi<void>("setAutoHide", enabled),
  getVersion: () => callApi<string>("getVersion"),
}
