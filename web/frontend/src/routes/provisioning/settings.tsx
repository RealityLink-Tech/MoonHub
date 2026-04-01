import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { useState } from "react"

import { getStatus, triggerRecovery } from "@/api/provisioning"
import { FactoryResetDialog } from "@/components/provisioning/factory-reset-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export const Route = createFileRoute("/provisioning/settings")({
  component: ProvisioningSettings,
})

function ProvisioningSettings() {
  const [recoveryLoading, setRecoveryLoading] = useState(false)

  const { data: statusData, isLoading, refetch } = useQuery({
    queryKey: ["provisioning", "status"],
    queryFn: getStatus,
    refetchInterval: (query) => query.state.error ? false : 5000,
  })

  const status = statusData?.status

  const handleRecovery = async () => {
    setRecoveryLoading(true)
    try {
      await triggerRecovery()
      await refetch()
    } catch (err) {
      console.error("Recovery failed:", err)
    } finally {
      setRecoveryLoading(false)
    }
  }

  return (
    <main className="max-w-md mx-auto px-6 pt-8 pb-32 flex flex-col items-center">
      {/* Header */}
      <div className="w-full mb-8">
        <h1 className="text-2xl font-light text-[#3a4144]">设备设置</h1>
        <p className="text-sm text-[#737c7f] mt-1">管理设备网络和系统设置</p>
      </div>

      {/* Device Status Card */}
      <Card className="w-full mb-6 bg-white/80 backdrop-blur-sm border-0 shadow-sm">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-[#3a4144]">设备状态</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {isLoading ? (
            <div className="text-sm text-[#737c7f]">加载中...</div>
          ) : (
            <>
              <div className="flex justify-between text-sm">
                <span className="text-[#737c7f]">模式</span>
                <span className="text-[#3a4144] font-medium">{status?.mode || "-"}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#737c7f]">运行状态</span>
                <span className="text-[#3a4144] font-medium">{status?.runtime?.phase || "-"}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#737c7f]">网络连接</span>
                <span className={`font-medium ${status?.network?.connected ? "text-green-600" : "text-red-500"}`}>
                  {status?.network?.connected ? "已连接" : "未连接"}
                </span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#737c7f]">当前网络</span>
                <span className="text-[#3a4144] font-medium">{status?.network?.activeConnection || "-"}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-[#737c7f]">热点状态</span>
                <span className={`font-medium ${status?.network?.apEnabled ? "text-green-600" : "text-[#737c7f]"}`}>
                  {status?.network?.apEnabled ? "已启用" : "未启用"}
                </span>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Network Recovery Card */}
      <Card className="w-full mb-6 bg-white/80 backdrop-blur-sm border-0 shadow-sm">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-[#3a4144]">网络恢复</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-[#737c7f]">
            如果网络连接出现问题，可以尝试触发网络恢复。系统将尝试重新连接到上次保存的WiFi网络，或启用热点作为后备。
          </p>
          <div className="flex justify-between text-sm">
            <span className="text-[#737c7f]">恢复功能</span>
            <span className={`font-medium ${status?.recovery?.enabled ? "text-green-600" : "text-[#737c7f]"}`}>
              {status?.recovery?.enabled ? "已启用" : "未启用"}
            </span>
          </div>
          {status?.recovery?.inCooldown && (
            <div className="text-xs text-yellow-600 bg-yellow-50 px-3 py-2 rounded">
              恢复冷却中，请稍后再试
            </div>
          )}
          <Button
            variant="outline"
            onClick={handleRecovery}
            disabled={recoveryLoading || status?.recovery?.inCooldown}
            className="w-full"
          >
            {recoveryLoading ? "恢复中..." : "触发网络恢复"}
          </Button>
        </CardContent>
      </Card>

      {/* Danger Zone */}
      <Card className="w-full bg-red-50/50 border border-red-200/50">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium text-red-600">危险操作</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <p className="text-sm text-[#737c7f]">
            恢复出厂设置将清除所有WiFi凭据和设备配置，设备将返回初始配网状态。
          </p>
          <FactoryResetDialog onReset={() => refetch()} />
        </CardContent>
      </Card>

      {/* Ambient Background */}
      <div className="fixed top-20 -left-20 w-64 h-64 bg-[#d4e4f7]/10 rounded-full blur-[80px] pointer-events-none" />
      <div className="fixed bottom-40 -right-20 w-80 h-80 bg-[#d1dce0]/20 rounded-full blur-[100px] pointer-events-none" />
    </main>
  )
}
