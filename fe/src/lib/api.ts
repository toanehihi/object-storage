import type { ApiResponse } from "./types";

const BASE_URL = "/api/v1";

function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("access_token");
}

function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("refresh_token");
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem("access_token", access);
  localStorage.setItem("refresh_token", refresh);
}

export function clearTokens() {
  localStorage.removeItem("access_token");
  localStorage.removeItem("refresh_token");
}

async function refreshAccessToken(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  try {
    const res = await fetch(`${BASE_URL}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refreshToken }),
    });

    if (!res.ok) return false;

    const json: ApiResponse<{ accessToken: string; refreshToken: string }> =
      await res.json();
    if (json.success && json.data) {
      setTokens(json.data.accessToken, json.data.refreshToken);
      return true;
    }
    return false;
  } catch {
    return false;
  }
}

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

async function apiFetch<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<ApiResponse<T>> {
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>),
  };

  const token = getToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  if (
    options.body &&
    typeof options.body === "string" &&
    !headers["Content-Type"]
  ) {
    headers["Content-Type"] = "application/json";
  }

  const res = await fetch(`${BASE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  // Handle 401 — try refresh once
  if (res.status === 401 && token) {
    if (!isRefreshing) {
      isRefreshing = true;
      refreshPromise = refreshAccessToken();
    }

    const refreshed = await refreshPromise;
    isRefreshing = false;
    refreshPromise = null;

    if (refreshed) {
      const newToken = getToken();
      headers["Authorization"] = `Bearer ${newToken}`;
      const retryRes = await fetch(`${BASE_URL}${endpoint}`, {
        ...options,
        headers,
      });
      return retryRes.json();
    }

    clearTokens();
    if (typeof window !== "undefined") {
      window.location.href = "/login";
    }
  }

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({}));
    throw new ApiError(
      res.status,
      errorBody.message || `Request failed with status ${res.status}`
    );
  }

  return res.json();
}

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "ApiError";
  }
}

export const authApi = {
  register(data: { email: string; password: string; fullName?: string }) {
    return apiFetch<import("./types").AuthResponse>("/auth/register", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  login(data: { email: string; password: string }) {
    return apiFetch<import("./types").AuthResponse>("/auth/login", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  me() {
    return apiFetch<import("./types").User>("/auth/me");
  },
};

export const uploadApi = {
  init(data: import("./types").InitUploadRequest) {
    return apiFetch<import("./types").InitUploadResponse>("/uploads", {
      method: "POST",
      body: JSON.stringify(data),
    });
  },

  getChunkURL(fileId: string, chunkIndex: number) {
    return apiFetch<import("./types").ChunkUploadURLResponse>(
      `/uploads/${fileId}/chunks/${chunkIndex}/url`
    );
  },

  markChunkComplete(fileId: string, chunkIndex: number, etag: string) {
    return apiFetch<unknown>(`/uploads/${fileId}/chunks/${chunkIndex}`, {
      method: "PUT",
      body: JSON.stringify({ chunkIndex, etag }),
    });
  },

  status(fileId: string) {
    return apiFetch<import("./types").UploadStatusResponse>(
      `/uploads/${fileId}/status`
    );
  },

  complete(fileId: string) {
    return apiFetch<unknown>(`/uploads/${fileId}/complete`, {
      method: "POST",
    });
  },
};

export const fileApi = {
  list(page = 1, pageSize = 20) {
    return apiFetch<import("./types").PaginatedResponse<import("./types").FileItem>>(
      `/files?page=${page}&pageSize=${pageSize}`
    );
  },

  metadata(fileId: string) {
    return apiFetch<import("./types").FileItem>(`/files/${fileId}`);
  },

  downloadURL(fileId: string) {
    return apiFetch<import("./types").DownloadURLResponse>(
      `/files/${fileId}/download`
    );
  },

  delete(fileId: string) {
    return apiFetch<unknown>(`/files/${fileId}`, { method: "DELETE" });
  },
};
