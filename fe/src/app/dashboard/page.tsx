"use client";

import { useState, useEffect, useCallback } from "react";
import { fileApi } from "@/lib/api";
import type { FileItem, PageMeta } from "@/lib/types";
import FileUploader from "@/components/FileUploader";
import FileList from "@/components/FileList";
import { FolderOpen, RefreshCw } from "lucide-react";

const PAGE_SIZE = 10;

export default function DashboardPage() {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [pagination, setPagination] = useState<PageMeta>({
    page: 1,
    pageSize: PAGE_SIZE,
    total: 0,
    totalPages: 0,
  });
  const [loading, setLoading] = useState(true);

  const fetchFiles = useCallback(async (page = 1) => {
    setLoading(true);
    try {
      const res = await fileApi.list(page, PAGE_SIZE);
      if (res.success && res.data) {
        setFiles(res.data.data ?? []);
        if (res.data.pagination) {
          setPagination(res.data.pagination);
        }
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

  const handlePageChange = (page: number) => {
    fetchFiles(page);
  };

  return (
    <div className="space-y-8 animate-slide-up">
      {/* Title */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <FolderOpen className="h-6 w-6 text-accent" />
          <h1 className="text-2xl font-bold text-foreground">My Files</h1>
        </div>
        <button
          onClick={() => fetchFiles(pagination.page)}
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
        <FileUploader onUploadComplete={() => fetchFiles(1)} />
      </section>

      {/* File List */}
      <section>
        <FileList
          files={files}
          loading={loading}
          page={pagination.page}
          totalPages={pagination.totalPages}
          totalCount={pagination.total}
          onPageChange={handlePageChange}
          onRefresh={() => fetchFiles(pagination.page)}
        />
      </section>
    </div>
  );
}
