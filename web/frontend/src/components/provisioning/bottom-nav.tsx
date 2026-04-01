import { Link } from "@tanstack/react-router"

export type NavStep = "connect" | "authorize" | "install" | "settings"

interface BottomNavProps {
  currentStep: NavStep
}

export function BottomNav({ currentStep }: BottomNavProps) {
  const steps: { key: NavStep; path: string; icon: string; label: string }[] = [
    { key: "connect", path: "/provisioning/", icon: "wifi", label: "连接" },
    { key: "authorize", path: "/provisioning/auth", icon: "vpn_key", label: "授权" },
    { key: "install", path: "/provisioning/install", icon: "install_mobile", label: "安装" },
    { key: "settings", path: "/provisioning/settings", icon: "settings", label: "设置" },
  ]

  return (
    <nav className="fixed bottom-0 left-0 w-full z-50 flex justify-around items-center px-8 pb-8 pt-4 bg-white/40 backdrop-blur-xl border-t border-white/20 shadow-[0_-8px_30px_rgb(0,0,0,0.04)]">
      {steps.map((step) => {
        const isActive = currentStep === step.key
        const isCompleted =
          (currentStep === "authorize" && step.key === "connect") ||
          (currentStep === "install" &&
            (step.key === "connect" || step.key === "authorize"))

        if (isActive) {
          return (
            <div
              key={step.key}
              className="flex flex-col items-center justify-center bg-gradient-to-br from-[#506070] to-[#708090] text-white rounded-full w-14 h-14 shadow-lg shadow-[#506070]/20"
            >
              <span
                className="material-symbols-outlined"
                style={{ fontVariationSettings: "'FILL' 1" }}
              >
                {step.icon}
              </span>
            </div>
          )
        }

        return (
          <Link
            key={step.key}
            to={step.path}
            className={`flex flex-col items-center justify-center transition-opacity ${
              isCompleted
                ? "text-[#506070] hover:opacity-80"
                : "text-[#abb3b7] hover:text-[#506070]"
            }`}
          >
            <span className="material-symbols-outlined">{step.icon}</span>
            <span className="text-[11px] tracking-[0.1em] uppercase mt-1">
              {step.label}
            </span>
          </Link>
        )
      })}
    </nav>
  )
}
