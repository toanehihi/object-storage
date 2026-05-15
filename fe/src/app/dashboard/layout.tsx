"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/lib/auth-context";
import {
  HardDrive,
  LogOut,
  Loader2,
} from "lucide-react";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
    }
  }, [user, loading, router]);

  if (loading) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-accent" />
      </div>
    );
  }

  if (!user) return null;

  return (
    <div className="flex min-h-screen flex-col">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-border bg-background/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
          <div className="flex items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl gradient-accent">
              <HardDrive className="h-4.5 w-4.5 text-white" />
            </div>
            <span className="text-lg font-bold gradient-text">ObjectVault</span>
          </div>

          <div className="flex items-center gap-4">
            <div className="hidden sm:flex flex-col items-end">
              <span className="text-sm font-medium text-foreground">
                {user.fullName || user.email}
              </span>
              {user.fullName && (
                <span className="text-xs text-muted">{user.email}</span>
              )}
            </div>

            <div className="h-9 w-9 rounded-full gradient-accent flex items-center justify-center text-sm font-bold text-white uppercase">
              {(user.fullName || user.email).charAt(0)}
            </div>

            <button
              id="logout-btn"
              onClick={() => {
                logout();
                router.push("/login");
              }}
              className="flex h-9 w-9 items-center justify-center rounded-xl border border-border text-muted transition-all hover:border-danger/50 hover:text-danger hover:bg-danger/5"
              title="Sign out"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </header>

      {/* Content */}
      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-8 sm:px-6">
        {children}
      </main>
    </div>
  );
}
