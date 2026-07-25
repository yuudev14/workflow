"use client"
import localFont from "next/font/local";
import "./globals.css";


import dynamic from 'next/dynamic'
import { Toaster } from "@/components/ui/toaster";
import AuthProvider from "@/components/provider/auth-provider";
import AppShell from "@/components/provider/app-shell";
const Providers = dynamic(() => import("../components/provider/main-provider"), { ssr: false })


const geistSans = localFont({
  src: "./fonts/GeistVF.woff",
  variable: "--font-geist-sans",
  weight: "100 900",
});
const geistMono = localFont({
  src: "./fonts/GeistMonoVF.woff",
  variable: "--font-geist-mono",
  weight: "100 900",
});



export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`} suppressHydrationWarning
      >
        {/* AuthProvider resolves the session before anything protected
            renders; AppShell then decides whether the page gets the sidebar
            chrome, so /login can render bare. */}
        <Providers>
          <AuthProvider>
            <AppShell>{children}</AppShell>
          </AuthProvider>
        </Providers>
        <Toaster />

      </body>
    </html>


  );
}
