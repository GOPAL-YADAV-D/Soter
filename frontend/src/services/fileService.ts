import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

export interface FileInfo {
  id: string;
  userFilename: string;
  originalName: string;
  fileSize: number;
  contentType: string;
  uploadedAt: string;
  downloadCount: number;
  lastAccessed?: string;
  folderPath: string;
  isDeduped: boolean;
  permissions: FilePermissions;
  owner?: {
    id: string;
    name: string;
    username: string;
  };
  groups?: Array<{
    id: string;
    name: string;
  }>;
}

export interface PermissionSet {
  read: boolean;
  write: boolean;
  execute: boolean;
}

export interface FilePermissions {
  // Simple boolean permissions for UI
  canRead: boolean;
  canWrite: boolean;
  canDownload: boolean;
  canDelete: boolean;
  canShare: boolean;
  
  // Detailed Linux-style permissions
  owner: PermissionSet;
  group: PermissionSet;
  others: PermissionSet;
  octal: string; // e.g., "644"
}

export interface UploadSession {
  sessionToken: string;
  totalFiles: number;
  totalBytes: number;
  duplicateFiles: number;
}

export interface FileInput {
  filename: string;
  mimeType: string;
  fileSize: number;
  folderPath: string;
  contentHash: string;
}

export interface UploadSessionRequest {
  files: FileInput[];
  totalBytes: number;
}

export interface StorageUsage {
  usedSpaceMb: number;
  allocatedSpaceMb: number;
  fileCount: number;
  usagePercentage: number;
  duplicatesSavedMb: number;
  duplicateCount: number;
}

export interface FileListResponse {
  files: FileInfo[];
  pagination: {
    page: number;
    pageSize: number;
    totalPages: number;
    totalFiles: number;
  };
  storageUsage: StorageUsage;
}

export interface FileMetadata {
  id: string;
  userFilename: string;
  originalName: string;
  fileSize: number;
  contentType: string;
  uploadedAt: string;
  downloadCount: number;
  lastAccessed?: string;
  folderPath: string;
  isDeduped: boolean;
  permissions: FilePermissions;
  owner?: {
    id: string;
    name: string;
    username: string;
  };
  groups?: Array<{
    id: string;
    name: string;
  }>;
  tags?: string[];
  duplicateCount?: number;
  isOriginal?: boolean;
  hash?: string; // Content hash
  groupName?: string; // Primary group name for backward compatibility
  relatedFiles?: Array<{
    id: string;
    userFilename: string;
    uploadedBy: string;
    uploaderName: string;
    uploadedAt: string;
  }>;
}

// Create axios instance with base configuration
const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add request interceptor to include auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

export const fileService = {
  // Create upload session
  async createUploadSession(data: UploadSessionRequest): Promise<UploadSession> {
    const response = await api.post('/files/upload-session', data);
    return response.data;
  },

  // Upload file with session
  async uploadFile(sessionId: string, file: File, onProgress?: (progress: number) => void): Promise<FileInfo> {
    const formData = new FormData();
    formData.append('file', file);

    const response = await api.post(`/files/upload/${sessionId}`, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          const progress = (progressEvent.loaded / progressEvent.total) * 100;
          onProgress(progress);
        }
      },
    });

    return response.data;
  },

  // Complete upload session
  async completeUploadSession(sessionToken: string): Promise<void> {
    await api.post(`/files/upload-session/${sessionToken}/complete`);
  },

  // Get files list
  async getFiles(params?: {
    page?: number;
    pageSize?: number;
    search?: string;
    mimeType?: string;
    sortBy?: 'userFilename' | 'fileSize' | 'uploadedAt';
    sortOrder?: 'asc' | 'desc';
  }): Promise<FileListResponse> {
    const queryParams = new URLSearchParams();
    
    if (params?.page) queryParams.append('page', params.page.toString());
    if (params?.pageSize) queryParams.append('pageSize', params.pageSize.toString());
    if (params?.search) queryParams.append('search', params.search);
    if (params?.mimeType) queryParams.append('mimeType', params.mimeType);
    if (params?.sortBy) queryParams.append('sortBy', params.sortBy);
    if (params?.sortOrder) queryParams.append('sortOrder', params.sortOrder);

    const url = queryParams.toString() ? `/files?${queryParams.toString()}` : '/files';
    const response = await api.get(url);
    return response.data;
  },

  // Get file metadata
  async getFileMetadata(fileId: string): Promise<FileMetadata> {
    const response = await api.get(`/files/${fileId}`);
    return response.data;
  },

  // Download file
  async downloadFile(fileId: string): Promise<Blob> {
    const response = await api.get(`/files/${fileId}/download`, {
      responseType: 'blob',
    });
    return response.data;
  },

  // Get download URL for direct download
  getDownloadUrl(fileId: string): string {
    const token = localStorage.getItem('token');
    return `${API_BASE_URL}/files/${fileId}/download?token=${encodeURIComponent(token || '')}`;
  },

  // Delete file
  async deleteFile(fileId: string): Promise<void> {
    await api.delete(`/files/${fileId}`);
  },

  // Get storage usage
  async getStorageUsage(): Promise<StorageUsage> {
    const response = await api.get('/files/storage-usage');
    return response.data;
  },

  // Update file permissions (TODO: Implement backend endpoint)
  async updateFilePermissions(fileId: string, permissions: {
    // Placeholder for future permission update functionality
    [key: string]: any;
  }): Promise<FileInfo> {
    const response = await api.put(`/files/${fileId}/permissions`, permissions);
    return response.data;
  },

  // Helper methods
  formatFileSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  },

  formatFileDate(dateString: string): string {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 1) return 'Just now';
    if (diffMins < 60) return `${diffMins} minutes ago`;
    if (diffHours < 24) return `${diffHours} hours ago`;
    if (diffDays < 7) return `${diffDays} days ago`;
    
    return date.toLocaleDateString();
  },

  getFileIcon(contentType: string): string {
    if (!contentType) return '📁'; // Handle undefined/null contentType
    if (contentType.startsWith('image/')) return '🖼️';
    if (contentType.startsWith('video/')) return '🎥';
    if (contentType.startsWith('audio/')) return '🎵';
    if (contentType.includes('pdf')) return '📄';
    if (contentType.includes('word') || contentType.includes('document')) return '📝';
    if (contentType.includes('excel') || contentType.includes('spreadsheet')) return '📊';
    if (contentType.includes('powerpoint') || contentType.includes('presentation')) return '📋';
    if (contentType.includes('zip') || contentType.includes('archive')) return '📦';
    if (contentType.includes('text/')) return '📃';
    return '📁';
  },

  // Calculate SHA-256 hash of file for deduplication
  async calculateFileHash(file: File): Promise<string> {
    const buffer = await file.arrayBuffer();
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
    return hashHex;
  },

  // Validate file before upload
  validateFile(file: File, maxSizeMb: number = 100): { valid: boolean; error?: string } {
    const maxSizeBytes = maxSizeMb * 1024 * 1024;
    
    if (file.size > maxSizeBytes) {
      return {
        valid: false,
        error: `File size exceeds ${maxSizeMb}MB limit`,
      };
    }

    // Check for potentially dangerous file types
    const dangerousTypes = [
      'application/x-executable',
      'application/x-msdownload',
      'application/x-msdos-program',
    ];

    if (dangerousTypes.includes(file.type)) {
      return {
        valid: false,
        error: 'File type not allowed for security reasons',
      };
    }

    return { valid: true };
  },
};

export default fileService;