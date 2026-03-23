import { createFileRoute } from "@tanstack/react-router"
import { useEffect, useState } from "react"

import { getAuthCode, regenerateAuthCode } from "@/api/provisioning"
import { AuthCode } from "@/components/provisioning/auth-code"
import { Button } from "@/components/ui/button"

export const Route = createFileRoute("/provisioning/auth")({
  component: ProvisioningAuth,
})

function ProvisioningAuth() {
  const [authCode, setAuthCode] = useState<string | null>(null)
  const [deviceId, setDeviceId] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [isRegenerating, setIsRegenerating] = useState(false)
  const [copied, setCopied] = useState(false)

  // Fetch auth code on mount
  useEffect(() => {
    const fetchAuthCode = async () => {
      try {
        const result = await getAuthCode()
        setAuthCode(result.code)
        setDeviceId(result.deviceId)
      } catch {
        // Generate a random code if API fails (for demo)
        const code = Math.floor(100000 + Math.random() * 900000)
          .toString()
          .replace(/(\d{3})(\d{3})/, "$1 $2")
        setAuthCode(code)
        setDeviceId("YSHU-" + new Date().getFullYear() + "-" + Math.random().toString(36).slice(0, 3).toUpperCase())
      } finally {
        setIsLoading(false)
      }
    }
    fetchAuthCode()
  }, [])

  const handleCopy = async () => {
    if (authCode) {
      await navigator.clipboard.writeText(authCode.replace(/\s/g, ""))
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  const handleRegenerate = async () => {
    setIsRegenerating(true)
    try {
      const result = await regenerateAuthCode()
      setAuthCode(result.code)
      setDeviceId(result.deviceId)
    } catch {
      // Generate a random code if API fails (for demo)
      const code = Math.floor(100000 + Math.random() * 900000)
        .toString()
        .replace(/(\d{3})(\d{3})/, "$1 $2")
      setAuthCode(code)
    } finally {
      setIsRegenerating(false)
    }
  }

  const handleNext = () => {
    window.location.href = "/provisioning/install"
  }

  return (
    <main className="min-h-[calc(100vh-64px)] flex flex-col items-center justify-center px-6 pb-24 relative overflow-hidden">
      {/* Abstract Background Light Elements */}
      <div className="absolute top-1/4 left-1/2 -translate-x-1/2 w-[600px] h-[600px] bg-[#d4e4f7]/20 rounded-full blur-[120px] -z-10" />

      {/* Success Celebration Icon & Halo */}
      <div className="relative flex items-center justify-center mb-12">
        <div className="absolute w-48 h-48 rounded-full border border-[#506070]/5 animate-[pulse-ring_4s_cubic-bezier(0.4,0,0.6,1)_infinite]" />
        <div className="absolute w-32 h-32 rounded-full border border-[#506070]/5 animate-[pulse-ring_4s_cubic-bezier(0.4,0,0.6,1)_infinite]" style={{ animationDelay: "1s" }} />
        <div className="w-20 h-20 bg-[#ffffff] rounded-full flex items-center justify-center shadow-sm relative z-10">
          <span className="material-symbols-outlined text-[#506070] text-4xl" style={{ fontVariationSettings: "'FILL' 1" }}>check_circle</span>
        </div>
      </div>

      {/* Headline & Code Section */}
      <div className="text-center space-y-8 max-w-md w-full">
        <div className="space-y-2">
          <h2 className="text-[#586064] font-light tracking-[0.2em] text-sm uppercase">Connection Successful</h2>
          <p className="text-[#506070] font-medium text-lg">配网成功</p>
        </div>

        {/* 6-Digit Auth Code */}
        {isLoading ? (
          <div className="py-10 bg-[#ffffff]/40 backdrop-blur-md rounded-[2.5rem] border border-white/40 shadow-sm">
            <div className="flex items-center justify-center">
              <span className="material-symbols-outlined animate-spin text-[#506070] text-4xl">sync</span>
            </div>
          </div>
        ) : (
          <AuthCode code={authCode || "------"} />
        )}

        <div className="space-y-6">
          <p className="text-[#586064] font-light text-sm leading-relaxed px-8">
            请妥善保存此授权码，用于连接 PWA
          </p>

          {/* Action Buttons */}
          <div className="flex items-center justify-center gap-3">
            <button
              onClick={handleCopy}
              className="inline-flex items-center space-x-2 px-6 py-3 rounded-full bg-[#e3e9ec] text-[#506070] text-sm font-medium hover:bg-[#dbe4e7] transition-colors active:scale-95 duration-200"
            >
              <span className="material-symbols-outlined text-sm">
                {copied ? "check" : "content_copy"}
              </span>
              <span>{copied ? "已复制" : "复制授权码"}</span>
            </button>

            <button
              onClick={handleRegenerate}
              disabled={isRegenerating}
              className="inline-flex items-center space-x-2 px-4 py-3 rounded-full border border-[#abb3b7] text-[#586064] text-sm hover:bg-[#f1f4f6] transition-colors active:scale-95 duration-200 disabled:opacity-50"
            >
              <span className={`material-symbols-outlined text-sm ${isRegenerating ? "animate-spin" : ""}`}>
                refresh
              </span>
            </button>
          </div>
        </div>
      </div>

      {/* Secondary Info & Main Action */}
      <div className="mt-20 w-full max-w-sm space-y-8">
        {deviceId && (
          <div className="flex flex-col items-center space-y-1">
            <span className="text-[10px] text-[#abb3b7] tracking-widest uppercase">Device Identity</span>
            <span className="text-[#586064] text-xs font-light">{deviceId}</span>
          </div>
        )}

        <Button
          onClick={handleNext}
          className="w-full py-5 rounded-full bg-gradient-to-br from-[#506070] to-[#455463] text-[#f4f8ff] font-medium tracking-wide shadow-lg shadow-[#506070]/20 hover:opacity-90 active:scale-[0.98] transition-all duration-300"
        >
          完成配网，安装应用
        </Button>
      </div>

      {/* CSS for pulse animation */}
      <style>{`
        @keyframes pulse-ring {
          0% { transform: scale(0.95); opacity: 0.8; }
          50% { transform: scale(1.1); opacity: 0.3; }
          100% { transform: scale(0.95); opacity: 0.8; }
        }
      `}</style>
    </main>
  )
}
