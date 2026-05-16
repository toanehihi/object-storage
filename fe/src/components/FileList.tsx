"use client";

import { useState } from "react";
import type { FileItem } from "@/lib/types";
import { fileApi } from "@/lib/api";
import {
  Download,
  Trash2,
  FileIcon,
  Film,
  Image as ImageIcon,
  FileText,
  Music,
  Archive,
  MoreVertical,
  Loader2,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";

function getFileIcon(contentType: string) {
  if (contentType.startsWith("video/")) return Film;
  if (contentType.startsWith("image/")) return ImageIcon;
  if (contentType.startsWith("audio/")) return Music;
  if (contentType.startsWith("text/") || contentType.includes("pdf"))
    return FileText;
  if (
    contentType.includes("zip") ||
    contentType.includes("tar") ||
    contentType.includes("rar")
  )
    return Archive;
  return FileIcon;
}

function getStatusStyle(status: string) {
  switch (status) {
    case "READY":
      return "bg-success/15 text-green-400";
    case "UPLOADING":
    case "UPLOADED":
      return "bg-accent/15 text-indigo-400";
    case "PROCESSING":
      return "bg-warning/15 text-amber-400";
    case "FAILED":
    case "INFECTED":
      return "bg-danger/15 text-red-400";
    case "DELETED":
    case "UNSCANNED":
      return "bg-zinc-500/15 text-zinc-400";
    default:
      return "bg-zinc-500/15 text-zinc-400";
  }
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}

function timeAgo(dateStr: string): string {
  const date = new Date(dateStr);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return "just now";
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  return `${Math.floor(seconds / 86400)}d ago`;
}

interface FileListProps {
  files: FileItem[];
  loading: boolean;
  page: number;
  totalPages: number;
  totalCount: number;
  onPageChange: (page: number) => void;
  onRefresh: () => void;
}

export default function FileList({
  files,
  loading,
  page,
  totalPages,
  totalCount,
  onPageChange,
  onRefresh,
}: FileListProps) {
  const [activeMenu, setActiveMenu] = useState<string | null>(null);
  const [downloading, setDownloading] = useState<string | null>(null);
  const [deleting, setDeleting] = useState<string | null>(null);

  const handleDownload = async (file: FileItem) => {
    setDownloading(file.id);
    setActiveMenu(null);
    try {
      const res = await fileApi.downloadURL(file.id);
      if (res.success && res.data.downloadUrl) {
        window.open(res.data.downloadUrl, "_blank");
      }
    } catch {
      // silently fail
    } finally {
      setDownloading(null);
    }
  };

  const handleDelete = async (file: FileItem) => {
    setDeleting(file.id);
    setActiveMenu(null);
    try {
      await fileApi.delete(file.id);
      onRefresh();
    } catch {
      // silently fail
    } finally {
      setDeleting(null);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-accent" />
      </div>
    );
  }

  if (files.length === 0 && page === 1) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-center">
        <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-surface">
          <FileIcon className="h-8 w-8 text-muted" />
        </div>
        <h3 className="text-lg font-medium text-foreground">No files yet</h3>
        <p className="mt-1 text-sm text-muted">
          Upload your first file to get started
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted">
          {totalCount} file{totalCount !== 1 ? "s" : ""}
        </p>
      </div>

      <div className="grid gap-3">
        {files.map((file, index) => {
          const Icon = getFileIcon(file.contentType);
          const isMenuOpen = activeMenu === file.id;

          return (
            <div
              key={file.id}
              className={`glass-card group flex items-center gap-4 p-4 transition-all hover:border-accent/30 animate-fade-in relative ${isMenuOpen ? "z-50" : "z-0"}`}
              style={{ animationDelay: `${index * 50}ms` }}
            >
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-accent/10">
                <Icon className="h-5 w-5 text-accent" />
              </div>

              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium text-foreground">
                  {file.filename}
                </p>
                <div className="mt-1 flex items-center gap-3 text-xs text-muted">
                  <span>{formatBytes(file.size)}</span>
                  <span>•</span>
                  <span>{timeAgo(file.createdAt)}</span>
                </div>
              </div>

              <span className={`badge ${getStatusStyle(file.status)}`}>
                {file.status === "PROCESSING" ? "SCANNING" : file.status}
              </span>

              <div className="relative">
                <button
                  onClick={() =>
                    setActiveMenu(isMenuOpen ? null : file.id)
                  }
                  className="flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-foreground"
                >
                  <MoreVertical className="h-4 w-4" />
                </button>

                {isMenuOpen && (
                  <>
                    <div
                      className="fixed inset-0 z-40"
                      onClick={() => setActiveMenu(null)}
                    />
                    <div className="absolute right-0 top-10 z-50 w-44 rounded-xl border border-border bg-surface p-1 shadow-xl animate-fade-in">
                      <button
                        onClick={() => handleDownload(file)}
                        disabled={
                          (file.status !== "READY" && file.status !== "UNSCANNED") || downloading === file.id
                        }
                        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-surface-hover disabled:opacity-40 disabled:cursor-not-allowed"
                      >
                        {downloading === file.id ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Download className="h-4 w-4" />
                        )}
                        Download
                      </button>
                      <button
                        onClick={() => handleDelete(file)}
                        disabled={deleting === file.id}
                        className="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-danger transition-colors hover:bg-danger/10 disabled:opacity-40"
                      >
                        {deleting === file.id ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Trash2 className="h-4 w-4" />
                        )}
                        Delete
                      </button>
                    </div>
                  </>
                )}
              </div>
            </div>
          );
        })}
      </div>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button
            onClick={() => onPageChange(page - 1)}
            disabled={page <= 1}
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-border text-muted transition-all hover:border-accent/50 hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronLeft className="h-4 w-4" />
          </button>

          {Array.from({ length: totalPages }, (_, i) => i + 1)
            .filter((p) => {
              // Show first, last, current, and neighbors
              if (p === 1 || p === totalPages) return true;
              if (Math.abs(p - page) <= 1) return true;
              return false;
            })
            .reduce<(number | "...")[]>((acc, p, idx, arr) => {
              if (idx > 0 && p - (arr[idx - 1]) > 1) {
                acc.push("...");
              }
              acc.push(p);
              return acc;
            }, [])
            .map((item, idx) =>
              item === "..." ? (
                <span key={`dots-${idx}`} className="px-1 text-muted">
                  …
                </span>
              ) : (
                <button
                  key={item}
                  onClick={() => onPageChange(item as number)}
                  className={`flex h-9 w-9 items-center justify-center rounded-lg text-sm font-medium transition-all ${
                    page === item
                      ? "gradient-accent text-white"
                      : "border border-border text-muted hover:border-accent/50 hover:text-foreground"
                  }`}
                >
                  {item}
                </button>
              )
            )}

          <button
            onClick={() => onPageChange(page + 1)}
            disabled={page >= totalPages}
            className="flex h-9 w-9 items-center justify-center rounded-lg border border-border text-muted transition-all hover:border-accent/50 hover:text-foreground disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        </div>
      )}
    </div>
  );
}
