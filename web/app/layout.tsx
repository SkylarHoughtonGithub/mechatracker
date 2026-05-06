import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { ClerkProvider } from '@clerk/nextjs';
import "./globals.css";
import { Providers } from './providers'
import { AppHeader } from './components/app-header'
import { Sidebar } from './components/sidebar'

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export const metadata: Metadata = {
  title: "mechatracker",
  description: "Repair tracking for mechanics, shops, and fleets.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <ClerkProvider>
      <html
        lang="en"
        className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      >
        <body className="min-h-full flex flex-col bg-black text-zinc-100">
          <Providers>
            <AppHeader />
            <div className="flex min-h-[calc(100vh-3rem)]">
              <Sidebar />
              <main className="flex-1 min-w-0">
                {children}
              </main>
            </div>
          </Providers>
        </body>
      </html>
    </ClerkProvider>
  );
}