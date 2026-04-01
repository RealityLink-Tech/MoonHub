import { Outlet, createRootRoute, redirect, useNavigate } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools"

function NotFound() {
  const navigate = useNavigate()
  return (
    <div className="min-h-screen flex flex-col items-center justify-center bg-[#f8f9fa]">
      <h1 className="text-2xl font-light text-[#506070] mb-4">404</h1>
      <p className="text-[#586064] text-sm mb-8">页面未找到</p>
      <button
        onClick={() => navigate({ to: "/provisioning/" })}
        className="px-6 py-3 rounded-full bg-[#506070] text-white text-sm hover:opacity-90 transition-opacity"
      >
        返回首页
      </button>
    </div>
  )
}

const RootLayout = () => {
  return (
    <>
      <Outlet />
      <TanStackRouterDevtools />
    </>
  )
}

export const Route = createRootRoute({
  component: RootLayout,
  notFoundComponent: NotFound,
  beforeLoad: ({ location }) => {
    if (location.pathname === "/") {
      throw redirect({ to: "/provisioning/" })
    }
  },
})
