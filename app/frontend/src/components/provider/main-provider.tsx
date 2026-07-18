'use client'

import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ThemeProvider } from "./theme-provider"
import PlaybookStatusProvider from "./playbook-status-provider"

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      refetchOnWindowFocus: false,
      refetchOnMount: false,
      refetchOnReconnect: false,
    }
  }
})

export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider
        attribute='class'
        defaultTheme='light'
        enableSystem
        disableTransitionOnChange
      >
        <PlaybookStatusProvider>{children}</PlaybookStatusProvider>
      </ThemeProvider>

    </QueryClientProvider>
  )
}