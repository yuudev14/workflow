"use client"

import { CheckCircle2, Info, XCircle } from "lucide-react"
import { useToast } from "@/hooks/use-toast"
import {
  Toast,
  ToastClose,
  ToastDescription,
  ToastProvider,
  ToastTitle,
  ToastViewport,
} from "@/components/ui/toast"

const ICON = {
  success: CheckCircle2,
  destructive: XCircle,
  default: Info,
}

export function Toaster() {
  const { toasts } = useToast()

  return (
    <ToastProvider>
      {toasts.map(function ({ id, title, description, action, variant, ...props }) {
        const Icon = ICON[variant ?? "default"]
        return (
          <Toast key={id} variant={variant} {...props}>
            <div className="flex items-start gap-2.5">
              <Icon className="mt-0.5 size-[18px] shrink-0" />
              <div className="grid gap-0.5">
                {title && <ToastTitle className="text-[13.5px] font-semibold">{title}</ToastTitle>}
                {description && (
                  <ToastDescription className="text-[12.5px] opacity-90">
                    {description}
                  </ToastDescription>
                )}
              </div>
            </div>
            {action}
            <ToastClose />
          </Toast>
        )
      })}
      <ToastViewport />
    </ToastProvider>
  )
}
