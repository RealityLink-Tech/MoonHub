import {
  Navigate,
  Outlet,
  createFileRoute,
  useRouterState,
} from "@tanstack/react-router"

import { ProvisioningLayout } from "@/components/provisioning/layout"

export const Route = createFileRoute("/provisioning")({
  component: ProvisioningRouteLayout,
})

function ProvisioningRouteLayout() {
  const pathname = useRouterState({
    select: (state) => state.location.pathname,
  })

  // Determine current step based on pathname
  const getStep = (): "connect" | "authorize" | "install" | "settings" => {
    if (pathname === "/provisioning/auth") return "authorize"
    if (pathname === "/provisioning/install") return "install"
    if (pathname === "/provisioning/settings") return "settings"
    return "connect"
  }

  // Get title based on current step
  const getTitle = (): string => {
    const step = getStep()
    switch (step) {
      case "authorize":
        return "设备授权"
      case "install":
        return "应用安装"
      case "settings":
        return "设备设置"
      default:
        return "设备配对"
    }
  }

  // Redirect to WiFi page if accessing /provisioning directly
  if (pathname === "/provisioning") {
    return <Navigate to="/provisioning" />
  }

  return (
    <ProvisioningLayout step={getStep()} title={getTitle()}>
      <Outlet />
    </ProvisioningLayout>
  )
}
