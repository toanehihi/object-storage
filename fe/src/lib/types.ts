// ───── API Response Wrapper ─────
export interface ApiResponse<T> {
  success: boolean;
  message?: string;
  data: T;
}

// ───── Auth ─────
export interface User {
  id: string;
  email: string;
  fullName?: string;
  avatarUrl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface TokenPair {
  accessToken: string;
  refreshToken: string;
}

export interface AuthResponse {
  user: User;
  token: TokenPair;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface RegisterRequest {
  email: string;
  password: string;
  fullName?: string;
}

// ───── Files ─────
export type FileStatus = "UPLOADING" | "PROCESSING" | "READY" | "FAILED" | "DELETED";

export interface FileItem {
  id: string;
  ownerId: string;
  filename: string;
  objectKey: string;
  size: number;
  contentType: string;
  status: FileStatus;
  checksum?: string;
  createdAt: string;
  updatedAt: string;
}

export interface FileListResponse {
  files: FileItem[] | null;
  limit: number;
  offset: number;
}

export interface DownloadURLResponse {
  fileId: string;
  filename: string;
  downloadUrl: string;
  expiresIn: number;
}

// ───── Upload ─────
export interface InitUploadRequest {
  filename: string;
  size: number;
  contentType?: string;
  chunkSize: number;
}

export interface InitUploadResponse {
  fileId: string;
  objectKey: string;
  chunkSize: number;
  totalChunks: number;
}

export interface ChunkUploadURLResponse {
  chunkIndex: number;
  url: string;
}

export interface UploadStatusResponse {
  fileId: string;
  status: string;
  uploadedChunks: number[] | null;
  totalChunks: number;
}
