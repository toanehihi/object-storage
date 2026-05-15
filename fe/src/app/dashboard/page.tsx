"use client";

import { useState, useEffect, useCallback } from "react";
import { fileApi } from "@/lib/api";
import type { FileItem } from "@/lib/types";
import FileUploader from "@/components/FileUploader";
import FileList from "@/components/FileList";
import { FolderOpen, RefreshCw } from "lucide-react";

export default function DashboardPage() {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchFiles = useCallback(async () => {
    setLoading(true);
    try {
      const res = await fileApi.list(50, 0);
      if (res.success) {
        setFiles(res.data.files || []);
      }
    } catch {
      // silently fail
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchFiles();
  }, [fetchFiles]);

  return (
    <div className="space-y-8 animate-slide-up">
      {/* Title */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <FolderOpen className="h-6 w-6 text-accent" />
          <h1 className="text-2xl font-bold text-foreground">My Files</h1>
        </div>
        <button
          onClick={fetchFiles}
          disabled={loading}
          className="flex items-center gap-2 rounded-xl border border-border px-4 py-2 text-sm font-medium text-muted transition-all hover:border-accent/50 hover:text-foreground disabled:opacity-50"
        >
          <RefreshCw
            className={`h-4 w-4 ${loading ? "animate-spin" : ""}`}
          />
          Refresh
        </button>
      </div>

      {/* Upload Section */}
      <section>
        <FileUploader onUploadComplete={fetchFiles} />
      </section>

      {/* File List */}
      <section>
        <FileList files={files} loading={loading} onRefresh={fetchFiles} />
      </section>
    </div>
  );
}
