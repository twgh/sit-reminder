import { useCallback, useEffect, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Progress, ProgressValue, ProgressTrack, ProgressIndicator } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { TooltipProvider } from "@/components/ui/tooltip"
import {
  PlayIcon,
  SquareIcon,
  PauseIcon,
  SkipForwardIcon,
  MusicIcon,
  Volume2Icon,
  BellIcon,
  ClockIcon,
  TimerIcon,
  MinusIcon,
  Maximize2Icon,
  XIcon,
  DumbbellIcon,
} from "lucide-react"
import { bridge, type AppConfig, type TimerStatus, type TimerFinishType } from "@/lib/bridge"

const DEFAULT_RINGTONE = "未设置"

function formatTime(totalSeconds: number): string {
  const m = Math.floor(totalSeconds / 60)
  const s = totalSeconds % 60
  return `${m.toString().padStart(2, "0")}:${s.toString().padStart(2, "0")}`
}

function StatusBadge({ status }: { status: TimerStatus }) {
  const config: Record<TimerStatus, { label: string; variant: "default" | "secondary" | "destructive" | "outline" }> = {
    idle: { label: "未开始", variant: "secondary" },
    running: { label: "运行中", variant: "default" },
    paused: { label: "已暂停", variant: "outline" },
    finished: { label: "提醒中", variant: "destructive" },
  }
  const { label, variant } = config[status]
  return <Badge variant={variant}>{label}</Badge>
}

// 提醒通知浮层 — 久坐时间到了
function NotificationOverlay({
  visible,
  onStopAndReset,
  onStartActivity,
  onSnooze,
}: {
  visible: boolean
  onStopAndReset: () => void
  onStartActivity: () => void
  onSnooze: (minutes: number) => void
}) {
  if (!visible) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <Card className="mx-4 w-full max-w-sm ring-2 ring-destructive/50">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex size-16 items-center justify-center rounded-full bg-destructive/10">
            <BellIcon className="size-8 text-destructive" />
          </div>
          <CardTitle className="text-xl">该起来活动一下了！</CardTitle>
          <CardDescription>久坐对健康不利，站起来走走吧！</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button size="lg" className="w-full" onClick={onStopAndReset}>
            <SquareIcon data-icon="inline-start" />
            停止并返回
          </Button>
          <Button variant="secondary" size="lg" className="w-full" onClick={onStartActivity}>
            <DumbbellIcon data-icon="inline-start" />
            开始5分钟活动倒计时
          </Button>
          <Separator />
          <div className="flex flex-col gap-2">
            <p className="text-center text-xs text-muted-foreground">稍后提醒</p>
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(5)}>
                5 分钟
              </Button>
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(10)}>
                10 分钟
              </Button>
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(15)}>
                15 分钟
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

// 活动结束通知浮层
function ActivityDoneOverlay({
  visible,
  onStopAndReset,
}: {
  visible: boolean
  onStopAndReset: () => void
}) {
  if (!visible) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <Card className="mx-4 w-full max-w-sm ring-2 ring-primary/50">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex size-16 items-center justify-center rounded-full bg-primary/10">
            <DumbbellIcon className="size-8 text-primary" />
          </div>
          <CardTitle className="text-xl">运动完成！</CardTitle>
          <CardDescription>活动时间结束，该回去继续工作啦~</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button size="lg" className="w-full" onClick={onStopAndReset}>
            <SquareIcon data-icon="inline-start" />
            停止并启动下一次提醒
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

export function App() {
  const [status, setStatus] = useState<TimerStatus>("idle")
  const [remainingSeconds, setRemainingSeconds] = useState(0)
  const [totalSeconds, setTotalSeconds] = useState(2400)
  const [intervalMinutes, setIntervalMinutes] = useState(40)
  const [ringtonePath, setRingtonePath] = useState(DEFAULT_RINGTONE)
  const [activityRingtonePath, setActivityRingtonePath] = useState(DEFAULT_RINGTONE)
  const [finishType, setFinishType] = useState<TimerFinishType>("regular")
  const [notificationVisible, setNotificationVisible] = useState(false)

  // 初始化
  useEffect(() => {
    window.__timerTick = (remaining: number, total: number) => {
      setRemainingSeconds(remaining)
      setTotalSeconds(total)
    }

    window.__timerFinished = (type: TimerFinishType) => {
      setFinishType(type)
      setStatus("finished")
      setRemainingSeconds(0)
      setNotificationVisible(true)
    }

    window.__statusChanged = (s: TimerStatus) => {
      setStatus(s)
      if (s !== "finished") {
        setNotificationVisible(false)
      }
    }

    window.__configLoaded = (config: AppConfig) => {
      setIntervalMinutes(config.intervalMinutes)
      setTotalSeconds(config.intervalMinutes * 60)
      setRingtonePath(config.ringtonePath || DEFAULT_RINGTONE)
      setActivityRingtonePath(config.activityRingtonePath || DEFAULT_RINGTONE)
    }

    window.__activityStarted = () => {
      setNotificationVisible(false)
      setStatus("running")
    }

    bridge.getConfig().then((config) => {
      if (config) {
        setIntervalMinutes(config.intervalMinutes)
        setTotalSeconds(config.intervalMinutes * 60)
        setRingtonePath(config.ringtonePath || DEFAULT_RINGTONE)
        setActivityRingtonePath(config.activityRingtonePath || DEFAULT_RINGTONE)
      }
    })
  }, [])

  // 启动常规计时
  const handleStart = useCallback(async () => {
    await bridge.startTimer()
  }, [])

  // 停止并重置
  const handleStopAndReset = useCallback(async () => {
    await bridge.stopTimer()
    setStatus("idle")
    setRemainingSeconds(intervalMinutes * 60)
    setTotalSeconds(intervalMinutes * 60)
    setNotificationVisible(false)
    setFinishType("regular")
  }, [intervalMinutes])

  // 暂停
  const handlePause = useCallback(async () => {
    await bridge.pauseTimer()
  }, [])

  // 继续
  const handleResume = useCallback(async () => {
    await bridge.resumeTimer()
  }, [])

  // 停止（运行中时）
  const handleStop = useCallback(async () => {
    await bridge.stopTimer()
    setStatus("idle")
    setRemainingSeconds(intervalMinutes * 60)
    setTotalSeconds(intervalMinutes * 60)
    setNotificationVisible(false)
    setFinishType("regular")
  }, [intervalMinutes])

  // 修改间隔时间
  const handleIntervalChange = useCallback(async (value: number) => {
    const clamped = Math.max(1, Math.min(999, value))
    setIntervalMinutes(clamped)
    setTotalSeconds(clamped * 60)
    await bridge.setInterval(clamped)
    await bridge.saveConfig()
  }, [])

  // 步进
  const stepInterval = useCallback(
    async (delta: number) => {
      const newVal = Math.max(1, Math.min(999, intervalMinutes + delta))
      setIntervalMinutes(newVal)
      setTotalSeconds(newVal * 60)
      await bridge.setInterval(newVal)
      await bridge.saveConfig()
    },
    [intervalMinutes]
  )

  // 选择提醒铃声
  const handleSelectRingtone = useCallback(async () => {
    const path = await bridge.selectRingtone()
    if (path) {
      setRingtonePath(path)
      await bridge.saveConfig()
    }
  }, [])

  const handleTestRingtone = useCallback(async () => {
    await bridge.testRingtone()
  }, [])

  // 选择活动铃声
  const handleSelectActivityRingtone = useCallback(async () => {
    const path = await bridge.selectActivityRingtone()
    if (path) {
      setActivityRingtonePath(path)
      await bridge.saveConfig()
    }
  }, [])

  const handleTestActivityRingtone = useCallback(async () => {
    await bridge.testActivityRingtone()
  }, [])

  // 开始活动倒计时
  const handleStartActivity = useCallback(async () => {
    await bridge.startActivityTimer()
  }, [])

  // 稍后提醒
  const handleSnooze = useCallback(async (minutes: number) => {
    await bridge.snooze(minutes)
    setNotificationVisible(false)
    setFinishType("regular")
  }, [])

  // 进度百分比: elapsed / total
  const progressPercent =
    totalSeconds > 0 ? Math.max(0, ((totalSeconds - remainingSeconds) / totalSeconds) * 100) : 0

  const stepperButtons = [
    { label: "-10", delta: -10 },
    { label: "-5", delta: -5 },
    { label: "-1", delta: -1 },
    { label: "+1", delta: 1 },
    { label: "+5", delta: 5 },
    { label: "+10", delta: 10 },
  ]

  return (
    <TooltipProvider>
      <div className="flex min-h-svh flex-col">
        {/* 可拖动的标题栏 */}
        <div
          className="flex h-9 shrink-0 items-center justify-between border-b bg-muted/30 px-2"
          style={{ WebkitAppRegion: "drag" } as React.CSSProperties}
        >
          <span className="select-none px-2 text-xs text-muted-foreground">久坐提醒助手</span>
          <div className="flex items-center" style={{ WebkitAppRegion: "no-drag" } as React.CSSProperties}>
            <Button variant="ghost" size="icon" className="size-8 rounded-none" onClick={() => bridge.minimize()}>
              <MinusIcon className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon" className="size-8 rounded-none" onClick={() => bridge.toggleMaximize()}>
              <Maximize2Icon className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon" className="size-8 rounded-none hover:bg-destructive hover:text-destructive-foreground" onClick={() => bridge.close()}>
              <XIcon className="size-3.5" />
            </Button>
          </div>
        </div>

        {/* 主内容 */}
        <div className="flex flex-1 items-start justify-center p-4">
          <div className="flex w-full max-w-md flex-col gap-4">
            {/* 标题 */}
            <div className="text-center">
              <h1 className="flex items-center justify-center gap-2 font-heading text-xl font-semibold">
                <ClockIcon className="size-5 text-primary" />
                久坐提醒助手
              </h1>
              <p className="mt-1 text-sm text-muted-foreground">定时提醒，守护健康</p>
            </div>

            {/* 状态显示 */}
            <div className="flex items-center justify-between">
              <StatusBadge status={status} />
              {status === "running" && (
                <span className="text-sm tabular-nums text-muted-foreground">
                  下次提醒: {formatTime(remainingSeconds)}
                </span>
              )}
            </div>

            {/* 进度条 (单条，Progress组件自带Track) */}
            <Progress value={status === "idle" ? 0 : progressPercent}>
              <ProgressValue className="tabular-nums" />
            </Progress>

            {/* 时间设置 */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">时间设置</CardTitle>
                <CardDescription>设置提醒间隔时间</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <div className="flex flex-1 items-center gap-1">
                  {stepperButtons.map((b) => (
                    <Button
                      key={b.label}
                      variant="outline"
                      size="sm"
                      className="h-7 flex-1 px-0 text-xs"
                      disabled={status === "running"}
                      onClick={() => stepInterval(b.delta)}
                    >
                      {b.label}
                    </Button>
                  ))}
                </div>
                <div className="flex items-center gap-2">
                  <Input
                    type="number"
                    min={1}
                    max={999}
                    value={intervalMinutes}
                    disabled={status === "running"}
                    onChange={(e) => {
                      const v = parseInt(e.target.value) || 0
                      setIntervalMinutes(v)
                    }}
                    onBlur={() => handleIntervalChange(intervalMinutes)}
                    className="w-24 text-center"
                  />
                  <span className="text-sm text-muted-foreground">分钟</span>
                  {status === "idle" && (
                    <span className="ml-auto text-xs text-muted-foreground">失焦自动保存</span>
                  )}
                </div>
              </CardContent>
            </Card>

            {/* 控制按钮 */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">控制</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-wrap gap-2">
                {status === "idle" && (
                  <Button className="flex-1" onClick={handleStart}>
                    <PlayIcon data-icon="inline-start" />
                    启动
                  </Button>
                )}
                {status === "running" && (
                  <>
                    <Button variant="secondary" className="flex-1" onClick={handlePause}>
                      <PauseIcon data-icon="inline-start" />
                      暂停
                    </Button>
                    <Button variant="destructive" className="flex-1" onClick={handleStop}>
                      <SquareIcon data-icon="inline-start" />
                      停止
                    </Button>
                  </>
                )}
                {status === "paused" && (
                  <>
                    <Button className="flex-1" onClick={handleResume}>
                      <PlayIcon data-icon="inline-start" />
                      继续
                    </Button>
                    <Button variant="destructive" className="flex-1" onClick={handleStop}>
                      <SquareIcon data-icon="inline-start" />
                      停止
                    </Button>
                  </>
                )}
                {status === "finished" && (
                  <Button className="flex-1" onClick={handleStopAndReset}>
                    <SkipForwardIcon data-icon="inline-start" />
                    停止并启动下一次
                  </Button>
                )}
              </CardContent>
            </Card>

            {/* 提醒铃声设置 */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">提醒铃声</CardTitle>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <MusicIcon className="size-4 shrink-0" />
                  <span className="flex-1 truncate">{ringtonePath}</span>
                </div>
                <Separator />
                <div className="flex gap-2">
                  <Button variant="outline" className="flex-1" size="sm" onClick={handleSelectRingtone}>
                    <MusicIcon data-icon="inline-start" />
                    选择铃声
                  </Button>
                  <Button variant="secondary" className="flex-1" size="sm" onClick={handleTestRingtone}>
                    <Volume2Icon data-icon="inline-start" />
                    测试播放
                  </Button>
                </div>
              </CardContent>
            </Card>

            {/* 活动结束铃声设置 */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">活动结束铃声</CardTitle>
                <CardDescription>5分钟活动结束后播放</CardDescription>
              </CardHeader>
              <CardContent className="flex flex-col gap-3">
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <MusicIcon className="size-4 shrink-0" />
                  <span className="flex-1 truncate">{activityRingtonePath}</span>
                </div>
                <Separator />
                <div className="flex gap-2">
                  <Button variant="outline" className="flex-1" size="sm" onClick={handleSelectActivityRingtone}>
                    <MusicIcon data-icon="inline-start" />
                    选择铃声
                  </Button>
                  <Button variant="secondary" className="flex-1" size="sm" onClick={handleTestActivityRingtone}>
                    <Volume2Icon data-icon="inline-start" />
                    测试播放
                  </Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </div>

        {/* 通知浮层 */}
        {finishType === "activity" ? (
          <ActivityDoneOverlay
            visible={notificationVisible}
            onStopAndReset={handleStopAndReset}
          />
        ) : (
          <NotificationOverlay
            visible={notificationVisible}
            onStopAndReset={handleStopAndReset}
            onStartActivity={handleStartActivity}
            onSnooze={handleSnooze}
          />
        )}
      </div>
    </TooltipProvider>
  )
}

export default App
