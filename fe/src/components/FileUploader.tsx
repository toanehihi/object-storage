"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import { uploadApi } from "@/lib/api";
import {
  Upload,
  X,
  CheckCircle2,
  AlertCircle,
  Loader2,
  RotateCcw,
  Clock,
  ChevronUp,
  ChevronDown,
  Minus,
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
  const [minimized, setMinimized] = useState(false);
  const [expanded, setExpanded] = useState(true);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const processingRef = useRef(false);

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

        const initRes = await uploadApi.init({
          filename: file.name,
          size: file.size,
          contentType: file.type || "application/octet-stream",
          chunkSize: CHUNK_SIZE,
        });

        const { fileId, totalChunks } = initRes.data;
        updateUpload(id, { fileId, totalChunks });

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
                const urlRes = await uploadApi.getChunkURL(fileId, ci);
                const presignedUrl = urlRes.data.url;

                const start = ci * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const blob = file.slice(start, end);

                const putRes = await fetch(presignedUrl, {
                  method: "PUT",
                  body: blob,
                  headers: { "Content-Type": "application/octet-stream" },
                });

                const etag =
                  putRes.headers.get("ETag") || `"chunk-${ci}"`;

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

        updateUpload(id, { status: "completing" });
        await uploadApi.complete(fileId);
        updateUpload(id, { status: "success" });
        onUploadComplete();
      } catch (err) {
        updateUpload(id, {
          status: "failed",
          error: err instanceof Error ? err.message : "Upload failed",
        });
      }
    },
    [updateUpload, onUploadComplete]
  );

  const startedIdsRef = useRef(new Set<string>());

  const processQueue = useCallback(() => {
    if (processingRef.current) return;
    processingRef.current = true;

    setUploads((prev) => {
      const activeCount = prev.filter(
        (u) => u.status === "uploading" || u.status === "completing"
      ).length;
      const slots = MAX_CONCURRENT_FILES - activeCount;

      if (slots <= 0) {
        processingRef.current = false;
        return prev;
      }

      const queued = prev.filter(
        (u) => u.status === "queued" && !startedIdsRef.current.has(u.id)
      );
      const toStart = queued.slice(0, slots);

      if (toStart.length === 0) {
        processingRef.current = false;
        return prev;
      }

      // Mark as started BEFORE firing async work
      for (const upload of toStart) {
        startedIdsRef.current.add(upload.id);
        uploadSingleFile(upload);
      }

      processingRef.current = false;

      // Atomically set status to "uploading" so they're never re-picked
      const startedIds = new Set(toStart.map((u) => u.id));
      return prev.map((u) =>
        startedIds.has(u.id) ? { ...u, status: "uploading" as const } : u
      );
    });
  }, [uploadSingleFile]);

  useEffect(() => {
    processQueue();
  }, [uploads, processQueue]);

  const handleFiles = useCallback((files: FileList) => {
    const newUploads: UploadingFile[] = Array.from(files).map((file) => ({
      id: `upload-${++uploadIdCounter}`,
      file,
      totalChunks: Math.ceil(file.size / CHUNK_SIZE),
      completedChunks: 0,
      status: "queued" as const,
    }));

    setUploads((prev) => [...prev, ...newUploads]);
    setMinimized(false);
    setExpanded(true);
  }, []);

  const retryUpload = useCallback((id: string) => {
    startedIdsRef.current.delete(id);
    setUploads((prev) =>
      prev.map((u) =>
        u.id === id
          ? {
              ...u,
              status: "queued" as const,
              completedChunks: 0,
              error: undefined,
            }
          : u
      )
    );
  }, []);

  const removeUpload = (id: string) => {
    startedIdsRef.current.delete(id);
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

  const hasActiveUploads = uploads.length > 0;

  return (
    <>
      {/* Drop Zone — stays inline */}
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
              Files are split into 5MB chunks • {MAX_CONCURRENT_FILES} files
              upload concurrently
            </p>
          </div>
        </div>
      </div>

      {/* Floating Upload Queue Dialog — bottom right */}
      {hasActiveUploads && (
        <div
          className={`fixed bottom-4 right-4 z-[100] w-[380px] rounded-2xl border border-border bg-surface shadow-2xl shadow-black/40 animate-slide-up transition-all ${
            minimized ? "h-auto" : ""
          }`}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-4 py-3 border-b border-border/50">
            <div className="flex items-center gap-3">
              <span className="text-sm font-semibold text-foreground">
                Uploads
              </span>
              <div className="flex items-center gap-1.5">
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

            <div className="flex items-center gap-1">
              {counts.success > 0 && (
                <button
                  onClick={clearCompleted}
                  className="rounded-lg px-2 py-1 text-xs text-muted hover:text-foreground transition-colors"
                >
                  Clear done
                </button>
              )}
              <button
                onClick={() => setExpanded(!expanded)}
                className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-foreground"
              >
                {expanded ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronUp className="h-4 w-4" />
                )}
              </button>
              <button
                onClick={() => setMinimized(true)}
                className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-foreground"
              >
                <Minus className="h-4 w-4" />
              </button>
              <button
                onClick={() =>
                  setUploads((prev) =>
                    prev.filter(
                      (u) =>
                        u.status !== "success" && u.status !== "failed"
                    )
                  )
                }
                className="flex h-7 w-7 items-center justify-center rounded-lg text-muted transition-colors hover:bg-surface-hover hover:text-danger"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          </div>

          {/* Items */}
          {expanded && !minimized && (
            <div className="max-h-72 overflow-y-auto divide-y divide-border/30">
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
                    className="flex items-center gap-3 px-4 py-2.5 animate-fade-in"
                  >
                    {/* Status Icon */}
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-background">
                      {upload.status === "queued" && (
                        <Clock className="h-3.5 w-3.5 text-muted" />
                      )}
                      {(upload.status === "uploading" ||
                        upload.status === "completing") && (
                        <Loader2 className="h-3.5 w-3.5 animate-spin text-accent" />
                      )}
                      {upload.status === "success" && (
                        <CheckCircle2 className="h-3.5 w-3.5 text-success" />
                      )}
                      {upload.status === "failed" && (
                        <AlertCircle className="h-3.5 w-3.5 text-danger" />
                      )}
                    </div>

                    {/* File Info */}
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium">
                        {upload.file.name}
                      </p>

                      {upload.status === "queued" && (
                        <p className="text-[11px] text-muted">Waiting…</p>
                      )}

                      {upload.status === "uploading" && (
                        <div className="mt-1">
                          <div className="progress-bar">
                            <div
                              className="progress-bar-fill"
                              style={{ width: `${progress}%` }}
                            />
                          </div>
                          <p className="mt-0.5 text-[11px] text-muted">
                            {upload.completedChunks}/{upload.totalChunks} •{" "}
                            {progress}%
                          </p>
                        </div>
                      )}

                      {upload.status === "completing" && (
                        <p className="text-[11px] text-warning">
                          Finalizing…
                        </p>
                      )}

                      {upload.status === "success" && (
                        <p className="text-[11px] text-success">Done</p>
                      )}

                      {upload.status === "failed" && (
                        <p className="text-[11px] text-danger truncate">
                          {upload.error || "Failed"}
                        </p>
                      )}
                    </div>

                    {/* Actions */}
                    <div className="flex shrink-0 items-center gap-0.5">
                      {upload.status === "failed" && (
                        <button
                          onClick={() => retryUpload(upload.id)}
                          className="flex h-6 w-6 items-center justify-center rounded text-muted hover:text-accent"
                          title="Retry"
                        >
                          <RotateCcw className="h-3 w-3" />
                        </button>
                      )}
                      {(upload.status === "success" ||
                        upload.status === "failed" ||
                        upload.status === "queued") && (
                        <button
                          onClick={() => removeUpload(upload.id)}
                          className="flex h-6 w-6 items-center justify-center rounded text-muted hover:text-foreground"
                          title="Remove"
                        >
                          <X className="h-3 w-3" />
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

      {/* Minimized FAB */}
      {hasActiveUploads && minimized && (
        <button
          onClick={() => setMinimized(false)}
          className="fixed bottom-4 right-4 z-[100] flex h-12 w-12 items-center justify-center rounded-full gradient-accent shadow-lg shadow-accent/20 text-white transition-transform hover:scale-110 animate-pulse-glow"
        >
          <Upload className="h-5 w-5" />
          {(counts.uploading > 0 || counts.queued > 0) && (
            <span className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-danger text-[10px] font-bold">
              {counts.uploading + counts.queued}
            </span>
          )}
        </button>
      )}
    </>
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
}
