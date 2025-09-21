import React, { useState, useRef, useCallback } from 'react';
import { X, Upload, FileText, AlertCircle, CheckCircle, Copy } from 'lucide-react';
import { fileService, UploadSession, StorageUsage } from '../services/fileService';

interface FileUploadProps {
  onClose: () => void;
  onUploadComplete: () => void;
  storageUsage: StorageUsage | null;
}

interface FileUploadState {
  file: File | null;
  dragActive: boolean;
  uploading: boolean;
  progress: number;
  uploadSession: UploadSession | null;
  error: string;
  success: boolean;
  hash: string;
}

const FileUpload: React.FC<FileUploadProps> = ({ onClose, onUploadComplete, storageUsage }) => {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [state, setState] = useState<FileUploadState>({
    file: null,
    dragActive: false,
    uploading: false,
    progress: 0,
    uploadSession: null,
    error: '',
    success: false,
    hash: '',
  });

  const updateState = (updates: Partial<FileUploadState>) => {
    setState(prev => ({ ...prev, ...updates }));
  };

  const handleDrag = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    if (e.type === 'dragenter' || e.type === 'dragover') {
      updateState({ dragActive: true });
    } else if (e.type === 'dragleave') {
      updateState({ dragActive: false });
    }
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    updateState({ dragActive: false });

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      handleFileSelect(e.dataTransfer.files[0]);
    }
  }, []);

  const handleFileInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      handleFileSelect(e.target.files[0]);
    }
  };

  const handleFileSelect = (file: File) => {
    updateState({ 
      file, 
      error: '', 
      success: false, 
      uploadSession: null,
      progress: 0 
    });

    // Validate file
    const validation = fileService.validateFile(file, 50); // 50MB limit for demo
    if (!validation.valid) {
      updateState({ error: validation.error || 'File validation failed' });
      return;
    }

    // Check storage quota
    if (storageUsage) {
      const fileSizeMb = file.size / (1024 * 1024);
      const availableSpaceMb = storageUsage.allocatedSpaceMb - storageUsage.usedSpaceMb;
      
      if (fileSizeMb > availableSpaceMb) {
        updateState({ 
          error: `File size (${fileSizeMb.toFixed(2)} MB) exceeds available storage space (${availableSpaceMb.toFixed(2)} MB)` 
        });
        return;
      }
    }

    // Calculate hash and create upload session
    calculateHashAndCreateSession(file);
  };

  const calculateHashAndCreateSession = async (file: File) => {
    try {
      updateState({ uploading: true, progress: 10 });

      // Calculate file hash
      const hash = await fileService.calculateFileHash(file);
      updateState({ hash, progress: 30 });

      // Create upload session
      const session = await fileService.createUploadSession({
        files: [{
          filename: file.name,
          mimeType: file.type,
          fileSize: file.size,
          folderPath: '',
          contentHash: hash,
        }],
        totalBytes: file.size,
      });

      updateState({ uploadSession: session, progress: 50 });

      // Check if all files are duplicates
      if (session.duplicateFiles === session.totalFiles) {
        // All files are duplicates
        updateState({ 
          uploading: false, 
          success: true, 
          progress: 100,
          error: '' 
        });
      } else {
        // Proceed with upload
        await uploadFile(file, session.sessionToken);
      }
    } catch (err: any) {
      console.error('Upload session creation failed:', err);
      updateState({ 
        uploading: false, 
        error: err.response?.data?.error || 'Failed to create upload session',
        progress: 0 
      });
    }
  };

  const uploadFile = async (file: File, sessionId: string) => {
    try {
      updateState({ progress: 60 });

      const fileInfo = await fileService.uploadFile(
        sessionId,
        file,
        (progressPercent) => {
          // Map upload progress to 60-100% range
          const mappedProgress = 60 + (progressPercent * 0.4);
          updateState({ progress: mappedProgress });
        }
      );

      updateState({ 
        uploading: false, 
        success: true, 
        progress: 100,
        error: '' 
      });

    } catch (err: any) {
      console.error('File upload failed:', err);
      updateState({ 
        uploading: false, 
        error: err.response?.data?.error || 'File upload failed',
        progress: 0 
      });
    }
  };

  const handleComplete = () => {
    onUploadComplete();
    onClose();
  };

  const copyHashToClipboard = () => {
    if (state.hash) {
      navigator.clipboard.writeText(state.hash);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-xl shadow-xl max-w-md w-full max-h-[90vh] overflow-y-auto">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <h2 className="text-xl font-semibold text-gray-900">Upload File</h2>
          <button
            onClick={onClose}
            className="text-gray-400 hover:text-gray-600"
          >
            <X className="h-6 w-6" />
          </button>
        </div>

        {/* Content */}
        <div className="p-6">
          {state.error && (
            <div className="mb-4 p-4 bg-red-50 border border-red-200 rounded-lg flex items-start">
              <AlertCircle className="h-5 w-5 text-red-400 mt-0.5 mr-3 flex-shrink-0" />
              <div>
                <p className="text-red-800 text-sm">{state.error}</p>
                <button
                  onClick={() => updateState({ error: '' })}
                  className="text-red-600 underline text-xs mt-1"
                >
                  Dismiss
                </button>
              </div>
            </div>
          )}

          {state.success && state.uploadSession ? (
            <div className="text-center">
              <CheckCircle className="h-16 w-16 text-green-500 mx-auto mb-4" />
              
              {state.uploadSession.duplicateFiles === state.uploadSession.totalFiles ? (
                <>
                  <h3 className="text-lg font-medium text-gray-900 mb-2">File Already Exists</h3>
                  <p className="text-gray-600 mb-6">
                    This file (or an identical one) is already in your organization. 
                    No storage space was used thanks to deduplication.
                  </p>
                </>
              ) : (
                <>
                  <h3 className="text-lg font-medium text-gray-900 mb-2">Upload Complete</h3>
                  <p className="text-gray-600 mb-6">
                    Your file has been successfully uploaded and is now available.
                  </p>
                </>
              )}

              {/* File Details */}
              <div className="bg-gray-50 rounded-lg p-4 mb-6 text-left">
                <div className="flex items-center mb-3">
                  <FileText className="h-5 w-5 text-gray-400 mr-2" />
                  <span className="font-medium text-gray-900">{state.file?.name}</span>
                </div>
                
                <div className="space-y-2 text-sm text-gray-600">
                  <div className="flex justify-between">
                    <span>Size:</span>
                    <span>{state.file ? fileService.formatFileSize(state.file.size) : '-'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span>Type:</span>
                    <span>{state.file?.type || 'Unknown'}</span>
                  </div>
                  {state.uploadSession.duplicateFiles === state.uploadSession.totalFiles && (
                    <div className="flex justify-between">
                      <span>Status:</span>
                      <span className="text-green-600">Deduplicated</span>
                    </div>
                  )}
                </div>

                {/* File Hash */}
                {state.hash && (
                  <div className="mt-4 pt-4 border-t">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-gray-500">File Hash (SHA-256):</span>
                      <button
                        onClick={copyHashToClipboard}
                        className="text-xs text-indigo-600 hover:text-indigo-500 flex items-center"
                        title="Copy to clipboard"
                      >
                        <Copy className="h-3 w-3 mr-1" />
                        Copy
                      </button>
                    </div>
                    <code className="text-xs text-gray-700 bg-gray-100 p-2 rounded mt-1 block break-all">
                      {state.hash}
                    </code>
                  </div>
                )}
              </div>

              <button
                onClick={handleComplete}
                className="w-full bg-indigo-600 text-white py-2 px-4 rounded-lg hover:bg-indigo-700"
              >
                Done
              </button>
            </div>
          ) : state.uploading ? (
            <div className="text-center">
              <div className="mb-4">
                <div className="inline-flex items-center justify-center w-16 h-16 bg-indigo-100 rounded-full mb-4">
                  <Upload className="h-8 w-8 text-indigo-600" />
                </div>
              </div>
              
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                {state.progress < 50 ? 'Processing file...' : 'Uploading...'}
              </h3>
              
              <div className="w-full bg-gray-200 rounded-full h-2 mb-4">
                <div 
                  className="bg-indigo-600 h-2 rounded-full transition-all duration-300"
                  style={{ width: `${state.progress}%` }}
                ></div>
              </div>
              
              <p className="text-sm text-gray-600">
                {Math.round(state.progress)}% complete
              </p>
              
              {state.file && (
                <p className="text-xs text-gray-500 mt-2">
                  {state.file.name} ({fileService.formatFileSize(state.file.size)})
                </p>
              )}
            </div>
          ) : (
            <>
              {/* Storage Usage Warning */}
              {storageUsage && storageUsage.usagePercentage > 80 && (
                <div className="mb-4 p-4 bg-amber-50 border border-amber-200 rounded-lg">
                  <div className="flex items-center">
                    <AlertCircle className="h-5 w-5 text-amber-400 mr-2" />
                    <span className="text-amber-800 text-sm">
                      Storage {storageUsage.usagePercentage.toFixed(0)}% full 
                      ({fileService.formatFileSize(storageUsage.usedSpaceMb * 1024 * 1024)} / {storageUsage.allocatedSpaceMb} MB)
                    </span>
                  </div>
                </div>
              )}

              {/* File Drop Zone */}
              <div
                className={`border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors ${
                  state.dragActive
                    ? 'border-indigo-500 bg-indigo-50'
                    : 'border-gray-300 hover:border-gray-400'
                }`}
                onDragEnter={handleDrag}
                onDragLeave={handleDrag}
                onDragOver={handleDrag}
                onDrop={handleDrop}
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="h-12 w-12 text-gray-400 mx-auto mb-4" />
                
                {state.file ? (
                  <div>
                    <p className="text-lg font-medium text-gray-900 mb-2">
                      {state.file.name}
                    </p>
                    <p className="text-sm text-gray-600 mb-4">
                      {fileService.formatFileSize(state.file.size)} • {state.file.type}
                    </p>
                    <button
                      onClick={(e) => {
                        e.stopPropagation();
                        updateState({ file: null });
                      }}
                      className="text-sm text-red-600 hover:text-red-500"
                    >
                      Remove file
                    </button>
                  </div>
                ) : (
                  <>
                    <p className="text-lg font-medium text-gray-900 mb-2">
                      Drop your file here
                    </p>
                    <p className="text-sm text-gray-600 mb-4">
                      or click to browse your files
                    </p>
                    <p className="text-xs text-gray-500">
                      Maximum file size: 50 MB
                    </p>
                  </>
                )}
              </div>

              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                onChange={handleFileInputChange}
              />

              {/* Upload Button */}
              {state.file && !state.uploading && (
                <div className="mt-6">
                  <button
                    onClick={() => handleFileSelect(state.file!)}
                    className="w-full bg-indigo-600 text-white py-2 px-4 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    disabled={!state.file}
                  >
                    Upload File
                  </button>
                </div>
              )}

              {/* Info */}
              <div className="mt-6 text-xs text-gray-500 space-y-1">
                <p>• Files are automatically scanned for duplicates</p>
                <p>• Identical files share storage space (deduplication)</p>
                <p>• All uploads are secured with permission controls</p>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
};

export default FileUpload;