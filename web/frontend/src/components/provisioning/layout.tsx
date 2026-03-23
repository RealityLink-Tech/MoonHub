import { type ReactNode } from "react"
import { Link } from "@tanstack/react-router"
import { useCallback, useEffect, useState } from "react"

import {
  PROVISIONING_AUTH_REQUIRED_EVENT,
  PROVISIONING_TOKEN_STORAGE_KEY,
} from "@/api/provisioning"

import { BottomNav, type NavStep } from "./bottom-nav"

interface ProvisioningLayoutProps {
  children: ReactNode
  step: NavStep
  title?: string
  showBack?: boolean
  backTo?: string
}

export function ProvisioningLayout({
  children,
  step,
  title = "设备配对",
  showBack = false,
  backTo,
}: ProvisioningLayoutProps) {
  const [showTokenPrompt, setShowTokenPrompt] = useState(false)
  const [tokenInput, setTokenInput] = useState("")

  useEffect(() => {
    const onAuth = () => setShowTokenPrompt(true)
    window.addEventListener(PROVISIONING_AUTH_REQUIRED_EVENT, onAuth)
    return () => window.removeEventListener(PROVISIONING_AUTH_REQUIRED_EVENT, onAuth)
  }, [])

  const saveToken = useCallback(() => {
    const t = tokenInput.trim()
    if (t) {
      sessionStorage.setItem(PROVISIONING_TOKEN_STORAGE_KEY, t)
    } else {
      sessionStorage.removeItem(PROVISIONING_TOKEN_STORAGE_KEY)
    }
    window.location.reload()
  }, [tokenInput])

  return (
    <div className="min-h-screen bg-[#f8f9fa] text-[#2b3437] font-sans selection:bg-[#d4e4f7] selection:text-[#445362]">
      {showTokenPrompt ? (
        <div
          className="sticky top-0 z-[60] px-4 py-3 bg-[#2b3437] text-[#f8f9fa] text-sm shadow-md"
          role="status"
        >
          <p className="mb-2 font-medium">需要配网 API 令牌</p>
          <p className="mb-2 opacity-90 text-xs leading-relaxed">
            服务端已设置 <code className="text-[#d4e4f7]">MOONHUB_PROVISIONING_TOKEN</code>
            。请输入相同令牌（仅保存在本机浏览器 sessionStorage），保存后将刷新页面。
          </p>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <input
              type="password"
              autoComplete="off"
              value={tokenInput}
              onChange={(e) => setTokenInput(e.target.value)}
              placeholder="Provisioning token"
              className="flex-1 rounded-lg px-3 py-2 text-[#2b3437] bg-white border border-white/20"
            />
            <button
              type="button"
              onClick={saveToken}
              className="rounded-lg px-4 py-2 bg-[#d4e4f7] text-[#2b3437] font-medium hover:opacity-90"
            >
              保存并刷新
            </button>
          </div>
        </div>
      ) : null}
      {/* Top Navigation */}
      <header className="w-full sticky top-0 z-50 bg-gradient-to-b from-[#f8f9fa] to-transparent">
        <div className="flex items-center justify-between px-6 h-16 max-w-screen-xl mx-auto">
          {showBack ? (
            <Link
              to={backTo || "/provisioning"}
              className="w-10 h-10 flex items-center justify-center rounded-full hover:bg-[#d4e4f7]/30 transition-colors active:scale-95 duration-300"
            >
              <span className="material-symbols-outlined text-[#506070]">arrow_back</span>
            </Link>
          ) : (
            <div className="w-10" />
          )}
          <h1 className="text-lg tracking-wider font-light text-[#506070]">
            {title}
          </h1>
          <div className="w-10" />
        </div>
      </header>

      {/* Main Content */}
      <main className="pb-32">{children}</main>

      {/* Ambient Background Effects */}
      <div className="fixed top-20 -left-20 w-64 h-64 bg-[#d4e4f7]/10 rounded-full blur-[80px] pointer-events-none" />
      <div className="fixed bottom-40 -right-20 w-80 h-80 bg-[#d1dce0]/20 rounded-full blur-[100px] pointer-events-none" />

      {/* Bottom Navigation */}
      <BottomNav currentStep={step} />
    </div>
  )
}
