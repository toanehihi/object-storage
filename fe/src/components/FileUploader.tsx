"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { uploadApi } from "@/lib/api";
import {
  Upload,
  X,
  CheckCircle2,
  AlertCircle,
  FileIcon,
  Loader2,
  RotateCcw,
  Clock,
  ChevronUp,
  ChevronDown,
} from "lucide-react";

const CHUNK_SIZE = 5 * 1024 * 1024; // 5MB
const MAX_CONCURRENT_FILES = 2;
const MAX_CONCURRENT_CHUNKS = 3;

type UploadStatus =
  | "queued"
  | "uploading"
  | "completing"
  | "success"
  | "failed";

interface UploadingFile {
  id: string;
  file: File;
  fileId?: string;
  totalChunks: number;
  completedChunks: number;
  status: UploadStatus;
  error?: string;
}

let uploadIdCounter = 0;

export default function FileUploader({
  onUploadComplete,
}: {
  onUploadComplete: () => void;
}) {
  const [uploads, setUploads] = useState<UploadingFile[]>([]);
  const [isDragOver, setIsDragOver] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const startedIdsRef = useRef<Set<string>>(new Set());

  const updateUpload = useCallback(
    (id: string, update: Partial<UploadingFile>) => {
      setUploads((prev) =>
        prev.map((u) => (u.id === id ? { ...u, ...update } : u))
      );
    },
    []
  );

  const uploadSingleFile = useCallback(
    async (upload: UploadingFile) => {
      const { id, file } = upload;

      try {
        updateUpload(id, { status: "uploading" });

        // 1. Init upload session
        const initRes = await uploadApi.init({
          filename: file.name,
          size: file.size,
          contentType: file.type || "application/octet-stream",
          chunkSize: CHUNK_SIZE,
        });

        const { fileId, totalChunks } = initRes.data;
        updateUpload(id, { fileId, totalChunks });

        // 2. Upload chunks with concurrency limit
        let completedCount = 0;
        const chunkIndices = Array.from({ length: totalChunks }, (_, i) => i);
        let activeCount = 0;
        let nextIdx = 0;

        await new Promise<void>((resolve, reject) => {
          const launchNext = () => {
            while (
              activeCount < MAX_CONCURRENT_CHUNKS &&
              nextIdx < chunkIndices.length
            ) {
              const ci = chunkIndices[nextIdx++];
              activeCount++;

              (async () => {
                // Get presigned URL
                const urlRes = await uploadApi.getChunkURL(fileId, ci);
                const presignedUrl = urlRes.data.url;

                // Slice chunk
                const start = ci * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const blob = file.slice(start, end);

                // PUT to MinIO
                const putRes = await fetch(presignedUrl, {
                  method: "PUT",
                  body: blob,
                  headers: { "Content-Type": "application/octet-stream" },
                });

                const etag =
                  putRes.headers.get("ETag") || `"chunk-${ci}"`;

                // Mark complete
                await uploadApi.markChunkComplete(fileId, ci, etag);

                completedCount++;
                updateUpload(id, { completedChunks: completedCount });
              })()
                .then(() => {
                  activeCount--;
                  if (completedCount === totalChunks) {
                    resolve();
                  } else {
                    launchNext();
                  }
                })
                .catch((err) => {
                  reject(err);
                });
            }
          };

          launchNext();
        });

        // 3. Complete
        updateUpload(id, { status: "completing" });
        await uploadApi.complete(fileId);
        updateUpload(id, { status: "success" });
        startedIdsRef.current.delete(id);
        onUploadComplete();
      } catch (err) {
        updateUpload(id, {
          status: "failed",
          error: err instanceof Error ? err.message : "Upload failed",
        });
        startedIdsRef.current.delete(id);
      }
    },
    [updateUpload, onUploadComplete]
  );

  // Queue processor — picks queued files and uploads them
  useEffect(() => {
    const activeCount = uploads.filter(
      (u) => u.status === "uploading" || u.status === "completing"
    ).length;
    const slots = MAX_CONCURRENT_FILES - activeCount;

    if (slots <= 0) return;

    const queued = uploads.filter(
      (u) => u.status === "queued" && !startedIdsRef.current.has(u.id)
    );
    const toStart = queued.slice(0, slots);

    for (const upload of toStart) {
      startedIdsRef.current.add(upload.id);
      uploadSingleFile(upload);
    }
  }, [uploads, uploadSingleFile]);

  const handleFiles = useCallback((files: FileList) => {
    const newUploads: UploadingFile[] = Array.from(files).map((file) => ({
      id: `upload-${++uploadIdCounter}`,
      file,
      totalChunks: Math.ceil(file.size / CHUNK_SIZE),
      completedChunks: 0,
      status: "queued" as const,
    }));

    setUploads((prev) => [...prev, ...newUploads]);
  }, []);

  const retryUpload = useCallback(
    (id: string) => {
      setUploads((prev) =>
        prev.map((u) =>
          u.id === id
            ? { ...u, status: "queued" as const, completedChunks: 0, error: undefined }
            : u
        )
      );
      // processQueue will pick it up via useEffect
    },
    []
  );

  const removeUpload = (id: string) => {
    setUploads((prev) => prev.filter((u) => u.id !== id));
  };

  const clearCompleted = () => {
    setUploads((prev) => prev.filter((u) => u.status !== "success"));
  };

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      setIsDragOver(false);
      if (e.dataTransfer.files.length > 0) {
        handleFiles(e.dataTransfer.files);
      }
    },
    [handleFiles]
  );

  const counts = {
    queued: uploads.filter((u) => u.status === "queued").length,
    uploading: uploads.filter(
      (u) => u.status === "uploading" || u.status === "completing"
    ).length,
    success: uploads.filter((u) => u.status === "success").length,
    failed: uploads.filter((u) => u.status === "failed").length,
  };

  return (
    <div className="space-y-4">
      {/* Drop Zone */}
      <div
        className={`drop-zone rounded-2xl p-8 text-center transition-all cursor-pointer ${
          isDragOver ? "active" : ""
        }`}
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragOver(true);
        }}
        onDragLeave={() => setIsDragOver(false)}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
      >
        <input
          ref={fileInputRef}
          type="file"
          multiple
          className="hidden"
          onChange={(e) => {
            if (e.target.files) handleFiles(e.target.files);
            e.target.value = "";
          }}
        />
        <div className="flex flex-col items-center gap-3">
          <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-accent/10">
            <Upload className="h-6 w-6 text-accent" />
          </div>
          <div>
            <p className="text-sm font-medium text-foreground">
              Drop files here or <span className="text-accent">browse</span>
            </p>
            <p className="mt-1 text-xs text-muted">
              Files are split into 5MB chunks • {MAX_CONCURRENT_FILES} files upload concurrently
            </p>
          </div>
        </div>
      </div>

      {/* Upload Queue */}
      {uploads.length > 0 && (
        <div className="glass-card overflow-hidden">
          {/* Queue Header */}
          <button
            onClick={() => setCollapsed(!collapsed)}
            className="flex w-full items-center justify-between px-4 py-3 text-sm font-medium text-foreground transition-colors hover:bg-surface-hover"
          >
            <div className="flex items-center gap-3">
              <span>Upload Queue</span>
              <div className="flex items-center gap-2">
                {counts.queued > 0 && (
                  <span className="badge bg-zinc-500/15 text-zinc-400">
                    <Clock className="h-3 w-3" />
                    {counts.queued}
                  </span>
                )}
                {counts.uploading > 0 && (
                  <span className="badge bg-accent/15 text-indigo-400">
                    <Loader2 className="h-3 w-3 animate-spin" />
                    {counts.uploading}
                  </span>
                )}
                {counts.success > 0 && (
                  <span className="badge bg-success/15 text-green-400">
                    <CheckCircle2 className="h-3 w-3" />
                    {counts.success}
                  </span>
                )}
                {counts.failed > 0 && (
                  <span className="badge bg-danger/15 text-red-400">
                    <AlertCircle className="h-3 w-3" />
                    {counts.failed}
                  </span>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {counts.success > 0 && (
                <span
                  onClick={(e) => {
                    e.stopPropagation();
                    clearCompleted();
                  }}
                  className="text-xs text-muted hover:text-foreground cursor-pointer"
                >
                  Clear done
                </span>
              )}
              {collapsed ? (
                <ChevronDown className="h-4 w-4 text-muted" />
              ) : (
                <ChevronUp className="h-4 w-4 text-muted" />
              )}
            </div>
          </button>

          {/* Queue Items */}
          {!collapsed && (
            <div className="divide-y divide-border/50">
              {uploads.map((upload) => {
                const progress =
                  upload.totalChunks > 0
                    ? Math.round(
                        (upload.completedChunks / upload.totalChunks) * 100
                      )
                    : 0;

                return (
                  <div
                    key={upload.id}
                    className="flex items-center gap-3 px-4 py-3 animate-fade-in"
                  >
                    {/* Status Icon */}
                    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-surface">
                      {upload.status === "queued" && (
                        <Clock className="h-4 w-4 text-muted" />
                      )}
                      {(upload.status === "uploading" ||
                        upload.status === "completing") && (
                        <Loader2 className="h-4 w-4 animate-spin text-accent" />
                      )}
                      {upload.status === "success" && (
                        <CheckCircle2 className="h-4 w-4 text-success" />
                      )}
                      {upload.status === "failed" && (
                        <AlertCircle className="h-4 w-4 text-danger" />
                      )}
                    </div>

                    {/* File Info */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center justify-between">
                        <p className="truncate text-sm font-medium">
                          {upload.file.name}
                        </p>
                        <span className="ml-2 shrink-0 text-xs text-muted">
                          {formatBytes(upload.file.size)}
                        </span>
                      </div>

                      {upload.status === "queued" && (
                        <p className="mt-0.5 text-xs text-muted">
                          Waiting in queue…
                        </p>
                      )}

                      {upload.status === "uploading" && (
                        <div className="mt-1.5">
                          <div className="progress-bar">
                            <div
                              className="progress-bar-fill"
                              style={{ width: `${progress}%` }}
                            />
                          </div>
                          <p className="mt-1 text-xs text-muted">
                            {upload.completedChunks}/{upload.totalChunks} chunks
                            • {progress}%
                          </p>
                        </div>
                      )}

                      {upload.status === "completing" && (
                        <p className="mt-0.5 text-xs text-warning">
                          Finalizing upload…
                        </p>
                      )}

                      {upload.status === "success" && (
                        <p className="mt-0.5 text-xs text-success">
                          Upload complete
                        </p>
                      )}

                      {upload.status === "failed" && (
                        <p className="mt-0.5 text-xs text-danger">
                          {upload.error || "Upload failed"}
                        </p>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="flex shrink-0 items-center gap-1">
                      {upload.status === "failed" && (
                        <button
                          onClick={() => retryUpload(upload.id)}
                          className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-accent"
                          title="Retry"
                        >
                          <RotateCcw className="h-3.5 w-3.5" />
                        </button>
                      )}
                      {(upload.status === "success" ||
                        upload.status === "failed" ||
                        upload.status === "queued") && (
                        <button
                          onClick={() => removeUpload(upload.id)}
                          className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-foreground"
                          title="Remove"
                        >
                          <X className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}
