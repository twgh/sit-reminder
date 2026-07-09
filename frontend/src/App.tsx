import { useCallback, useEffect, useRef, useState } from "react"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Progress, ProgressValue } from "@/components/ui/progress"
import { Separator } from "@/components/ui/separator"
import { Slider } from "@/components/ui/slider"
import { TooltipProvider } from "@/components/ui/tooltip"
import {
  PlayIcon,
  SquareIcon,
  PauseIcon,
  SkipForwardIcon,
  MusicIcon,
  Volume2Icon,
  VolumeXIcon,
  BellIcon,
  ClockIcon,
  MinusIcon,
  XIcon,
  SettingsIcon,
  ArrowLeftIcon,
  DumbbellIcon,
} from "lucide-react"
import { bridge, type AppConfig, type TimerStatus, type TimerFinishType, type TimerType } from "@/lib/bridge"

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

// 久坐提醒通知浮层
function NotificationOverlay({
  visible,
  onStopAndReset,
  onStartActivity,
  onSnooze,
  activityMinutes,
  setActivityMinutes,
}: {
  visible: boolean
  onStopAndReset: () => void
  onStartActivity: () => void
  onSnooze: (minutes: number) => void
  activityMinutes: number
  setActivityMinutes: (v: number) => void
}) {
  if (!visible) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <Card className="mx-4 w-full max-w-sm ring-2 ring-destructive/50">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex size-16 items-center justify-center rounded-full bg-destructive/10">
            <BellIcon className="size-8 text-destructive" />
          </div>
          <CardTitle className="text-xl select-none">该起来活动一下了！</CardTitle>
          <CardDescription className="select-none">久坐对健康不利，站起来走走吧！</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button size="lg" className="w-full" onClick={onStartActivity}>
            <DumbbellIcon data-icon="inline-start" />
            开始{activityMinutes}分钟活动倒计时
          </Button>
          {/* 活动时间设置 */}
          <div className="flex items-center justify-center gap-2">
            <span className="text-sm text-muted-foreground">活动时间:</span>
            <Input
              type="number"
              min={1}
              max={99}
              value={activityMinutes}
              onChange={(e) => setActivityMinutes(Math.max(1, Math.min(99, parseInt(e.target.value) || 1)))}
              className="w-20 text-center"
            />
            <span className="text-sm text-muted-foreground">分钟</span>
          </div>
          <Button variant="secondary" size="lg" className="w-full" onClick={onStopAndReset}>
            <SquareIcon data-icon="inline-start" />
            停止并返回
          </Button>
          <Separator />
          <div className="flex flex-col gap-2">
            <p className="text-center text-xs text-muted-foreground select-none">稍后提醒</p>
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(3)}>3 分钟</Button>
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(5)}>5 分钟</Button>
              <Button variant="outline" className="flex-1" size="sm" onClick={() => onSnooze(10)}>10 分钟</Button>
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
  onStopOnly,
  onStopAndStartNext,
}: {
  visible: boolean
  onStopOnly: () => void
  onStopAndStartNext: () => void
}) {
  if (!visible) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm">
      <Card className="mx-4 w-full max-w-sm ring-2 ring-primary/50">
        <CardHeader className="text-center">
          <div className="mx-auto mb-3 flex size-16 items-center justify-center rounded-full bg-primary/10">
            <DumbbellIcon className="size-8 text-primary" />
          </div>
          <CardTitle className="text-xl select-none">运动完成！</CardTitle>
          <CardDescription className="select-none">活动时间结束，该回去继续工作啦~</CardDescription>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <Button size="lg" className="w-full" onClick={onStopAndStartNext}>
            <SkipForwardIcon data-icon="inline-start" />
            开始下一次久坐提醒
          </Button>
          <Button variant="secondary" size="lg" className="w-full" onClick={onStopOnly}>
            <SquareIcon data-icon="inline-start" />
            停止
          </Button>
        </CardContent>
      </Card>
    </div>
  )
}

// 设置页面
function SettingsPage({
  ringtonePath,
  activityRingtonePath,
  isTestPlaying,
  isActivityTestPlaying,
  activityMinutes,
  volume,
  onSelectRingtone,
  onTestRingtone,
  onStopRingtone,
  onSelectActivityRingtone,
  onTestActivityRingtone,
  onStopActivityRingtone,
  onActivityMinutesChange,
  onVolumeChange,
  onBack,
}: {
  ringtonePath: string
  activityRingtonePath: string
  isTestPlaying: boolean
  isActivityTestPlaying: boolean
  activityMinutes: number
  volume: number
  onSelectRingtone: () => void
  onTestRingtone: () => void
  onStopRingtone: () => void
  onSelectActivityRingtone: () => void
  onTestActivityRingtone: () => void
  onStopActivityRingtone: () => void
  onActivityMinutesChange: (value: number) => void
  onVolumeChange: (value: number) => void
  onBack: () => void
}) {
  // 音量在 UI 上使用 0-100 的百分比，配置中存储 0-1000
  const volumePercent = Math.round(volume / 10)

  // 活动时间步进按钮
  const activitySteppers = [
    { label: "+1", delta: 1 },
    { label: "-1", delta: -1 },
    { label: "+3", delta: 3 },
    { label: "-3", delta: -3 },
  ]

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {/* 固定标题栏 */}
      <div className="flex shrink-0 items-center gap-3 border-b p-4 pb-3">
        <Button variant="ghost" size="icon" className="size-8" onClick={onBack}>
          <ArrowLeftIcon className="size-4" />
        </Button>
        <h2 className="flex items-center gap-2 font-heading text-lg font-semibold">
          <SettingsIcon className="size-5 text-primary" />
          设置
        </h2>
      </div>
      {/* 可滚动内容 */}
      <div className="flex min-h-0 flex-1 items-start justify-center overflow-y-auto p-4">
      <div className="flex w-full max-w-md flex-col gap-4">

        {/* 久坐提醒铃声设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">久坐提醒铃声</CardTitle>
            <CardDescription>久坐提醒时播放的铃声</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <MusicIcon className="size-4 shrink-0" />
              <span className="flex-1 truncate">{ringtonePath}</span>
            </div>
            <Separator />
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" size="sm" onClick={onSelectRingtone}>
                <MusicIcon data-icon="inline-start" />选择铃声
              </Button>
              {isTestPlaying ? (
                <Button variant="destructive" className="flex-1" size="sm" onClick={onStopRingtone}>
                  <VolumeXIcon data-icon="inline-start" />停止播放
                </Button>
              ) : (
                <Button variant="secondary" className="flex-1" size="sm" onClick={onTestRingtone}>
                  <Volume2Icon data-icon="inline-start" />测试播放
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* 活动结束铃声设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">活动结束铃声</CardTitle>
            <CardDescription>活动倒计时结束时播放</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <MusicIcon className="size-4 shrink-0" />
              <span className="flex-1 truncate">{activityRingtonePath}</span>
            </div>
            <Separator />
            <div className="flex gap-2">
              <Button variant="outline" className="flex-1" size="sm" onClick={onSelectActivityRingtone}>
                <MusicIcon data-icon="inline-start" />选择铃声
              </Button>
              {isActivityTestPlaying ? (
                <Button variant="destructive" className="flex-1" size="sm" onClick={onStopActivityRingtone}>
                  <VolumeXIcon data-icon="inline-start" />停止播放
                </Button>
              ) : (
                <Button variant="secondary" className="flex-1" size="sm" onClick={onTestActivityRingtone}>
                  <Volume2Icon data-icon="inline-start" />测试播放
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* 活动时间设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">活动时间</CardTitle>
            <CardDescription>久坐提醒后的活动时长</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex gap-1">
              {activitySteppers.map((b) => (
                <Button
                  key={b.label}
                  variant="outline"
                  size="sm"
                  className="h-7 flex-1 px-0 text-xs"
                  onClick={() => onActivityMinutesChange(activityMinutes + b.delta)}
                >
                  {b.label}
                </Button>
              ))}
            </div>
            <div className="flex items-center gap-2">
              <Input
                type="number"
                min={1}
                max={99}
                value={activityMinutes}
                onChange={(e) => onActivityMinutesChange(parseInt(e.target.value) || 1)}
                onBlur={() => onActivityMinutesChange(activityMinutes)}
                className="w-24 text-center"
              />
              <span className="text-sm text-muted-foreground">分钟</span>
            </div>
          </CardContent>
        </Card>

        {/* 音量设置 */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">音量</CardTitle>
            <CardDescription>铃声播放时使用的音量</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <div className="flex items-center gap-3">
              <VolumeXIcon className="size-4 shrink-0 text-muted-foreground" />
              <Slider
                value={[volumePercent]}
                min={0}
                max={100}
                step={1}
                onValueChange={(v) => onVolumeChange((Array.isArray(v) ? v[0] : v) * 10)}
                className="flex-1"
              />
              <Volume2Icon className="size-4 shrink-0 text-muted-foreground" />
              <span className="w-10 text-right text-sm tabular-nums text-muted-foreground">{volumePercent}%</span>
            </div>
          </CardContent>
        </Card>
      </div>
      </div>
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
  const [finishType, setFinishType] = useState<TimerFinishType>("sedentary")
  const [timerType, setTimerType] = useState<TimerType>("sedentary")
  const [notificationVisible, setNotificationVisible] = useState(false)
  const [showSettings, setShowSettings] = useState(false)
  const [isTestPlaying, setIsTestPlaying] = useState(false)
  const [isActivityTestPlaying, setIsActivityTestPlaying] = useState(false)
  const [activityMinutes, setActivityMinutes] = useState(5)
  const [tempActivityMinutes, setTempActivityMinutes] = useState(5)
  const [volume, setVolume] = useState(1000)

  // 跟踪最新的活动时长，供 __timerFinished 回调读取，避免闭包陈旧
  const activityMinutesRef = useRef(activityMinutes)
  activityMinutesRef.current = activityMinutes

  // 初始化全局回调
  useEffect(() => {
    window.__timerTick = (remaining: number, total: number) => {
      setRemainingSeconds(remaining)
      setTotalSeconds(total)
    }

    window.__timerFinished = (type: TimerFinishType) => {
      setFinishType(type)
      setStatus("finished")
      setRemainingSeconds(0)
      // 每次久坐提醒时，临时活动时长从配置值开始
      setTempActivityMinutes(activityMinutesRef.current)
      setNotificationVisible(true)
    }

    window.__statusChanged = (s: TimerStatus) => {
      setStatus(s)
      if (s !== "finished") setNotificationVisible(false)
    }

    window.__configLoaded = (config: AppConfig) => {
      setIntervalMinutes(config.intervalMinutes)
      setTotalSeconds(config.intervalMinutes * 60)
      setRingtonePath(config.ringtonePath || DEFAULT_RINGTONE)
      setActivityRingtonePath(config.activityRingtonePath || DEFAULT_RINGTONE)
      if (config.activityMinutes && config.activityMinutes > 0) {
        setActivityMinutes(config.activityMinutes)
        setTempActivityMinutes(config.activityMinutes)
      }
      if (typeof config.volume === "number") {
        setVolume(config.volume)
      }
    }

    window.__activityStarted = (_minutes: number) => {
      setNotificationVisible(false)
      setStatus("running")
    }

    window.__timerTypeChanged = (type: TimerType) => {
      setTimerType(type)
    }

    window.__showSettings = () => {
      setShowSettings(true)
    }

    bridge.getConfig().then((config) => {
      if (config) {
        setIntervalMinutes(config.intervalMinutes)
        setTotalSeconds(config.intervalMinutes * 60)
        setRingtonePath(config.ringtonePath || DEFAULT_RINGTONE)
        setActivityRingtonePath(config.activityRingtonePath || DEFAULT_RINGTONE)
        if (config.activityMinutes && config.activityMinutes > 0) {
          setActivityMinutes(config.activityMinutes)
          setTempActivityMinutes(config.activityMinutes)
        }
        if (typeof config.volume === "number") {
          setVolume(config.volume)
        }
      }
    })
  }, [])

  // 启动计时（预置剩余秒数防闪屏）
  const handleStart = useCallback(async () => {
    setRemainingSeconds(totalSeconds)
    setStatus("running")
    await bridge.startTimer()
  }, [totalSeconds])

  // 停止并重置到空闲态
  const handleStopAndReset = useCallback(async () => {
    await bridge.stopTimer()
    setStatus("idle")
    setRemainingSeconds(intervalMinutes * 60)
    setTotalSeconds(intervalMinutes * 60)
    setNotificationVisible(false)
    setFinishType("sedentary")
  }, [intervalMinutes])

  // 停止活动铃声但不开始下一次久坐提醒
  const handleStopOnly = useCallback(async () => {
    await bridge.stopTimer()
    setStatus("idle")
    setRemainingSeconds(intervalMinutes * 60)
    setTotalSeconds(intervalMinutes * 60)
    setNotificationVisible(false)
    setFinishType("sedentary")
  }, [intervalMinutes])

  // 停止活动铃声并开始下一次久坐提醒
  const handleStopAndStartNext = useCallback(async () => {
    await bridge.stopTimer()
    setNotificationVisible(false)
    setFinishType("sedentary")
    // 立即启动下一次久坐提醒
    await bridge.startTimer()
    setStatus("running")
  }, [])

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
    setFinishType("sedentary")
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

  // 选择久坐提醒铃声
  const handleSelectRingtone = useCallback(async () => {
    const path = await bridge.selectRingtone()
    if (path) {
      setRingtonePath(path)
      await bridge.saveConfig()
    }
  }, [])

  // 测试久坐提醒铃声
  const handleTestRingtone = useCallback(async () => {
    setIsTestPlaying(true)
    await bridge.testRingtone()
  }, [])

  // 停止久坐提醒铃声测试
  const handleStopRingtone = useCallback(async () => {
    await bridge.stopRingtone()
    setIsTestPlaying(false)
  }, [])

  // 选择活动铃声
  const handleSelectActivityRingtone = useCallback(async () => {
    const path = await bridge.selectActivityRingtone()
    if (path) {
      setActivityRingtonePath(path)
      await bridge.saveConfig()
    }
  }, [])

  // 测试活动铃声
  const handleTestActivityRingtone = useCallback(async () => {
    setIsActivityTestPlaying(true)
    await bridge.testActivityRingtone()
  }, [])

  // 停止活动铃声测试
  const handleStopActivityRingtone = useCallback(async () => {
    await bridge.stopRingtone()
    setIsActivityTestPlaying(false)
  }, [])

  // 修改活动时间（设置页面）
  const handleActivityMinutesChange = useCallback(async (value: number) => {
    const clamped = Math.max(1, Math.min(99, value))
    setActivityMinutes(clamped)
    await bridge.setActivityMinutes(clamped)
    await bridge.saveConfig()
  }, [])

  // 修改音量（设置页面），UI 百分比 0-100 映射到配置 0-1000
  const handleVolumeChange = useCallback(async (value: number) => {
    const clamped = Math.max(0, Math.min(1000, value))
    setVolume(clamped)
    await bridge.setVolume(clamped)
    await bridge.saveConfig()
  }, [])

  // 开始活动倒计时（使用自定义分钟数）
  const handleStartActivity = useCallback(async () => {
    await bridge.startActivityTimerWithMinutes(tempActivityMinutes)
  }, [tempActivityMinutes])

  // 稍后提醒
  const handleSnooze = useCallback(async (minutes: number) => {
    await bridge.snooze(minutes)
    setNotificationVisible(false)
    setFinishType("sedentary")
  }, [])

  // 进度百分比: elapsed / total
  const progressPercent =
    totalSeconds > 0 ? Math.max(0, ((totalSeconds - remainingSeconds) / totalSeconds) * 100) : 0

  const stepperButtons = [
    { label: "30", value: 30 },
    { label: "40", value: 40 },
    { label: "+5", delta: 5 },
    { label: "-5", delta: -5 },
    { label: "+10", delta: 10 },
    { label: "-10", delta: -10 },
  ]

  return (
    <TooltipProvider>
      <div className="flex h-svh flex-col overflow-hidden select-none">
        {/* 可拖动标题栏 */}
        <div
          className="flex h-9 shrink-0 items-center justify-between border-b bg-muted/30 px-2"
          style={{ WebkitAppRegion: "drag" } as React.CSSProperties}
        >
          <span className="px-2 text-xs text-muted-foreground">久坐提醒助手</span>
          <div className="flex items-center" style={{ WebkitAppRegion: "no-drag" } as React.CSSProperties}>
            {/* 设置按钮 */}
            <Button
              variant="ghost"
              size="icon"
              className="size-8 rounded-none"
              onClick={() => setShowSettings((s) => !s)}
            >
              <SettingsIcon className="size-3.5" />
            </Button>
            <Button variant="ghost" size="icon" className="size-8 rounded-none" onClick={() => bridge.minimize()}>
              <MinusIcon className="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              className="size-8 rounded-none hover:bg-destructive hover:text-destructive-foreground"
              onClick={() => bridge.close()}
            >
              <XIcon className="size-3.5" />
            </Button>
          </div>
        </div>

        {showSettings ? (
          <SettingsPage
            ringtonePath={ringtonePath}
            activityRingtonePath={activityRingtonePath}
            isTestPlaying={isTestPlaying}
            isActivityTestPlaying={isActivityTestPlaying}
            activityMinutes={activityMinutes}
            volume={volume}
            onSelectRingtone={handleSelectRingtone}
            onTestRingtone={handleTestRingtone}
            onStopRingtone={handleStopRingtone}
            onSelectActivityRingtone={handleSelectActivityRingtone}
            onTestActivityRingtone={handleTestActivityRingtone}
            onStopActivityRingtone={handleStopActivityRingtone}
            onActivityMinutesChange={handleActivityMinutesChange}
            onVolumeChange={handleVolumeChange}
            onBack={() => setShowSettings(false)}
          />
        ) : (
          /* 主内容 */
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
                <div className="flex items-center gap-2">
                  <StatusBadge status={status} />
                  {status === "running" && (
                    <span className="text-xs text-muted-foreground">
                      {timerType === "sedentary" ? "久坐提醒" : timerType === "activity" ? "活动倒计时" : "稍后提醒"}
                    </span>
                  )}
                </div>
                {status === "running" && (
                  <span className="text-sm tabular-nums text-muted-foreground">
                    {timerType === "activity" ? "活动结束" : "下次提醒"}: {formatTime(remainingSeconds)}
                  </span>
                )}
              </div>

              {/* 进度条 */}
              <Progress value={status === "idle" ? 0 : progressPercent}>
                <ProgressValue className="tabular-nums" />
              </Progress>

              {/* 时间设置 */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">时间设置</CardTitle>
                  <CardDescription>设置久坐提醒间隔时间</CardDescription>
                </CardHeader>
                <CardContent className="flex flex-col gap-3">
                  <div className="flex gap-1">
                    {stepperButtons.map((b) => (
                      <Button
                        key={b.label}
                        variant="outline"
                        size="sm"
                        className="h-7 flex-1 px-0 text-xs"
                        disabled={status === "running"}
                        onClick={() => b.value !== undefined ? handleIntervalChange(b.value) : stepInterval(b.delta)}
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
                      onChange={(e) => setIntervalMinutes(parseInt(e.target.value) || 0)}
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
            </div>
          </div>
        )}

        {/* 通知浮层 */}
        {finishType === "activity" ? (
          <ActivityDoneOverlay
            visible={notificationVisible}
            onStopOnly={handleStopOnly}
            onStopAndStartNext={handleStopAndStartNext}
          />
        ) : (
          <NotificationOverlay
            visible={notificationVisible}
            onStopAndReset={handleStopAndReset}
            onStartActivity={handleStartActivity}
            onSnooze={handleSnooze}
            activityMinutes={tempActivityMinutes}
            setActivityMinutes={setTempActivityMinutes}
          />
        )}
      </div>
    </TooltipProvider>
  )
}

export default App
