import { useState } from "react"
import { Trash2, AlertTriangle } from "lucide-react"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { factoryReset } from "@/api/provisioning"

interface FactoryResetDialogProps {
  onReset?: () => void
  trigger?: React.ReactNode
}

export function FactoryResetDialog({ onReset, trigger }: FactoryResetDialogProps) {
  const [isLoading, setIsLoading] = useState(false)
  const [open, setOpen] = useState(false)

  const handleReset = async () => {
    setIsLoading(true)
    try {
      await factoryReset()
      setOpen(false)
      onReset?.()
      // The device will reset and the page will likely become unavailable
      // as the hotspot is re-enabled
    } catch (error) {
      console.error("Factory reset failed:", error)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <AlertDialog open={open} onOpenChange={setOpen}>
      <AlertDialogTrigger asChild>
        {trigger || (
          <Button variant="destructive" size="sm">
            <Trash2 className="size-4" />
            恢复出厂设置
          </Button>
        )}
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <div className="flex items-center gap-3 mb-2">
            <div className="flex size-10 items-center justify-center rounded-full bg-destructive/10">
              <AlertTriangle className="size-5 text-destructive" />
            </div>
            <AlertDialogTitle>恢复出厂设置</AlertDialogTitle>
          </div>
          <AlertDialogDescription className="text-left">
            此操作将<strong>永久重置</strong>设备到初始状态：
          </AlertDialogDescription>
          <ul className="text-sm text-muted-foreground list-disc list-inside space-y-1 mt-2">
            <li>所有已保存的 WiFi 网络将被清除</li>
            <li>所有设备配置将被重置</li>
            <li>设备将返回配网模式</li>
            <li>您需要重新设置设备</li>
          </ul>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>取消</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleReset}
            disabled={isLoading}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {isLoading ? "重置中..." : "确认重置"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
