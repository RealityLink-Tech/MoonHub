import { createFileRoute } from "@tanstack/react-router"
import { useEffect, useState } from "react"

import type { WifiNetwork } from "@/api/provisioning"
import { scanNetworks, connectWifi } from "@/api/provisioning"
import { WifiSelector } from "@/components/provisioning/wifi-selector"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Switch } from "@/components/ui/switch"
import { Label } from "@/components/ui/label"

export const Route = createFileRoute("/provisioning/")({
  component: ProvisioningIndex,
})

function ProvisioningIndex() {
  const [networks, setNetworks] = useState<WifiNetwork[]>([])
  const [selectedSsid, setSelectedSsid] = useState<string | null>(null)
  const [password, setPassword] = useState("")
  const [manualSsid, setManualSsid] = useState("")
  const [isManualMode, setIsManualMode] = useState(false)
  const [isScanning, setIsScanning] = useState(true)
  const [isConnecting, setIsConnecting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Initial scan
  useEffect(() => {
    handleScan()
  }, [])

  const handleSelectNetwork = (network: WifiNetwork) => {
    setSelectedSsid(network.ssid)
  }

  const handleScan = async () => {
    setIsScanning(true)
    setError(null)
    try {
      const result = await scanNetworks()
      setNetworks(result.networks)
      if (result.networks.length > 0 && !selectedSsid) {
        setSelectedSsid(result.networks[0].ssid)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "扫描网络失败")
    } finally {
      setIsScanning(false)
    }
  }

  const handleConnect = async () => {
    const ssid = isManualMode ? manualSsid : selectedSsid
    if (!ssid) {
      setError("请选择或输入网络名称")
      return
    }

    setIsConnecting(true)
    setError(null)

    try {
      await connectWifi(ssid, password, false)
      // Navigate to auth page on success
      window.location.href = "/provisioning/auth"
    } catch (err) {
      setError(err instanceof Error ? err.message : "连接失败")
    } finally {
      setIsConnecting(false)
    }
  }

  const selectedNetwork = networks.find((n) => n.ssid === selectedSsid)

  return (
    <main className="max-w-md mx-auto px-6 pt-8 pb-32 flex flex-col items-center">
      {/* Center Icon with Lunar Halo */}
      <div className="relative mb-12">
        <div className="absolute inset-0 bg-[#d4e4f7]/30 rounded-full blur-3xl scale-150" />
        <div className="relative w-32 h-32 rounded-full bg-[#ffffff] shadow-[0_0_60px_10px_rgba(212,228,247,0.4)] flex items-center justify-center">
          <span className="material-symbols-outlined text-[#506070] text-5xl">wifi</span>
        </div>
      </div>

      {/* WiFi Selection Section */}
      <div className="w-full space-y-8">
        <WifiSelector
          networks={networks}
          selectedSsid={selectedSsid}
          onSelect={handleSelectNetwork}
          onRefresh={handleScan}
          isLoading={isScanning}
        />

        {/* Manual SSID Toggle */}
        <div className="flex items-center justify-between px-1 pt-2">
          <span className="text-sm text-[#586064] font-light">手动输入 SSID</span>
          <div className="flex items-center gap-2">
            <Switch
              id="manual-mode"
              checked={isManualMode}
              onCheckedChange={setIsManualMode}
            />
            <Label htmlFor="manual-mode" className="sr-only">
              手动输入 SSID
            </Label>
          </div>
        </div>

        {/* Manual SSID Input */}
        {isManualMode && (
          <div className="space-y-2">
            <Label className="px-1 text-[11px] uppercase tracking-widest text-[#737c7f]">
              网络名称
            </Label>
            <Input
              value={manualSsid}
              onChange={(e) => setManualSsid(e.target.value)}
              placeholder="输入 WiFi 名称 (SSID)"
              className="bg-[#ffffff] border-0 border-b border-[#abb3b7]/30 focus:border-[#506070] focus:ring-0 py-4 px-1"
            />
          </div>
        )}

        {/* Password Input */}
        {((!isManualMode && selectedNetwork) || (isManualMode && manualSsid)) && (
          <div className="space-y-2">
            <Label className="px-1 text-[11px] uppercase tracking-widest text-[#737c7f]">
              访问密码
            </Label>
            <div className="relative">
              <Input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="请输入密码"
                className="bg-[#ffffff] border-0 border-b border-[#abb3b7]/30 focus:border-[#506070] focus:ring-0 py-4 px-1 pr-10"
              />
              <button
                type="button"
                className="absolute right-2 top-1/2 -translate-y-1/2 text-[#abb3b7] hover:text-[#506070] transition-colors"
              >
                <span className="material-symbols-outlined">visibility_off</span>
              </button>
            </div>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="px-4 py-3 rounded-lg bg-[#fe8983]/20 text-[#9f403d] text-sm">
            {error}
          </div>
        )}
      </div>

      {/* Primary Action Button */}
      <div className="fixed bottom-24 left-0 w-full px-6 flex justify-center">
        <Button
          onClick={handleConnect}
          disabled={(!isManualMode && !selectedSsid) || (isManualMode && !manualSsid) || isConnecting}
          className="w-full max-w-[320px] h-12 bg-gradient-to-r from-[#506070] to-[#455463] text-[#f4f8ff] rounded-full font-medium text-sm tracking-widest shadow-lg shadow-[#506070]/20 active:scale-95 transition-all duration-300 flex items-center justify-center gap-2"
        >
          {isConnecting ? (
            <>
              <span className="material-symbols-outlined animate-spin">sync</span>
              <span>连接中...</span>
            </>
          ) : (
            <>
              <span className="leading-none">开始配网</span>
              <span className="material-symbols-outlined text-lg">bolt</span>
            </>
          )}
        </Button>
      </div>

      {/* Ambient Background */}
      <div className="fixed top-20 -left-20 w-64 h-64 bg-[#d4e4f7]/10 rounded-full blur-[80px] pointer-events-none" />
      <div className="fixed bottom-40 -right-20 w-80 h-80 bg-[#d1dce0]/20 rounded-full blur-[100px] pointer-events-none" />
    </main>
  )
}
