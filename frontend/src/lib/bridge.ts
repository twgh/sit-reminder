// Bridge 通信层：前端与 Go 后端之间的 JS Bridge

export interface AppConfig {
  intervalMinutes: number
  ringtonePath: string
  activityRingtonePath: string
}

export type TimerStatus = "idle" | "running" | "paused" | "finished"
export type TimerFinishType = "regular" | "snooze" | "activity"

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
    }
    __timerTick: (remainingSeconds: number, totalSeconds: number) => void
    __timerFinished: (type: TimerFinishType) => void
    __statusChanged: (status: TimerStatus) => void
    __configLoaded: (config: AppConfig) => void
    __activityStarted: () => void
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
    const fn = (window.api as Record<string, (...args: unknown[]) => Promise<T>>)[method]
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
  selectRingtone: () => callApi<string>("selectRingtone"),
  testRingtone: () => callApi<void>("testRingtone"),
  stopRingtone: () => callApi<void>("stopRingtone"),
  selectActivityRingtone: () => callApi<string>("selectActivityRingtone"),
  testActivityRingtone: () => callApi<void>("testActivityRingtone"),
  getConfig: () => callApi<AppConfig>("getConfig"),
  saveConfig: () => callApi<void>("saveConfig"),
}
