import React, { useState, useCallback, useRef } from 'react';
import { useDropzone } from 'react-dropzone';
import { Upload, X, File, CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import * as fileService from '../../services/fileService';

interface FileUploadProgress {
  file: File;
  progress: number;
  status: 'pending' | 'uploading' | 'completed' | 'error';
  error?: string;
  warnings?: string[];
  fileId?: string;
  isExisting?: boolean;
  savingsBytes?: number;
}

interface UploadSession {
  sessionToken: string;
  totalFiles: number;
  totalBytes: number;
  duplicateFiles: number;
}

interface FileUploadComponentProps {
  onUploadComplete?: (files: FileUploadProgress[]) => void;
  onUploadProgress?: (progress: number) => void;
  maxFiles?: number;
  maxFileSize?: number;
  acceptedFileTypes?: string[];
  folderPath?: string;
}

const FileUploadComponent: React.FC<FileUploadComponentProps> = ({
  onUploadComplete,
  onUploadProgress,
  maxFiles = 10,
  maxFileSize = 100 * 1024 * 1024 * 1024, // 100GB
  acceptedFileTypes,
  folderPath = '/',
}) => {
  const [uploadFiles, setUploadFiles] = useState<FileUploadProgress[]>([]);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadSession, setUploadSession] = useState<UploadSession | null>(null);
  const [overallProgress, setOverallProgress] = useState(0);
  const [deduplicationSavings, setDeduplicationSavings] = useState(0);
  
  const uploadProgressRef = useRef<{ [key: string]: number }>({});

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const createUploadSession = async (files: File[]): Promise<UploadSession> => {
    const fileInputs: fileService.FileInput[] = await Promise.all(
      files.map(async (file) => ({
        filename: file.name,
        mimeType: file.type,
        fileSize: file.size,
        folderPath: folderPath || '',
        contentHash: await fileService.default.calculateFileHash(file),
      }))
    );

    const totalBytes = files.reduce((sum, file) => sum + file.size, 0);

    const response = await fileService.default.createUploadSession({
      files: fileInputs,
      totalBytes,
    });

    return {
      sessionToken: response.sessionToken,
      totalFiles: response.totalFiles,
      totalBytes: response.totalBytes,
      duplicateFiles: response.duplicateFiles,
    };
  };

  const uploadFile = async (
    file: File,
    sessionToken: string,
    onProgress: (progress: number) => void
  ): Promise<any> => {
    return await fileService.default.uploadFile(sessionToken, file, onProgress);
  };

  const updateOverallProgress = () => {
    const totalFiles = uploadFiles.length;
    if (totalFiles === 0) return;

    const totalProgress = Object.values(uploadProgressRef.current).reduce((sum, progress) => sum + progress, 0);
    const overall = Math.round(totalProgress / totalFiles);
    setOverallProgress(overall);
    onUploadProgress?.(overall);
  };

  const handleUpload = async (files: File[]) => {
    setIsUploading(true);
    setOverallProgress(0);
    setDeduplicationSavings(0);
    uploadProgressRef.current = {};

    // Initialize upload progress tracking
    const initialProgress: FileUploadProgress[] = files.map(file => ({
      file,
      progress: 0,
      status: 'pending',
    }));
    setUploadFiles(initialProgress);

    try {
      // Create upload session
      const session = await createUploadSession(files);
      setUploadSession(session);

      // Upload files concurrently
      const uploadPromises = files.map(async (file, index) => {
        try {
          uploadProgressRef.current[file.name] = 0;

          // Update file status to uploading
          setUploadFiles(prev => prev.map((f, i) => 
            i === index ? { ...f, status: 'uploading' } : f
          ));

          const result = await uploadFile(file, session.sessionToken, (progress) => {
            uploadProgressRef.current[file.name] = progress;
            
            // Update individual file progress
            setUploadFiles(prev => prev.map((f, i) => 
              i === index ? { ...f, progress } : f
            ));
            
            updateOverallProgress();
          });

          // Update file status to completed
          setUploadFiles(prev => prev.map((f, i) => 
            i === index ? {
              ...f,
              progress: 100,
              status: 'completed',
              fileId: result.fileID,
              isExisting: result.isExisting,
              savingsBytes: result.savingsBytes,
              warnings: result.warnings,
            } : f
          ));

          // Update deduplication savings
          if (result.savingsBytes > 0) {
            setDeduplicationSavings(prev => prev + result.savingsBytes);
          }

          return result;
        } catch (error) {
          console.error(`Upload failed for ${file.name}:`, error);
          
          // Update file status to error
          setUploadFiles(prev => prev.map((f, i) => 
            i === index ? {
              ...f,
              status: 'error',
              error: error instanceof Error ? error.message : 'Upload failed',
            } : f
          ));
          
          throw error;
        }
      });

      const results = await Promise.allSettled(uploadPromises);
      
      // Complete upload session
      await fileService.default.completeUploadSession(session.sessionToken);

      // Call completion callback
      onUploadComplete?.(uploadFiles);

    } catch (error) {
      console.error('Upload session failed:', error);
      // Handle overall upload failure
    } finally {
      setIsUploading(false);
    }
  };

  const onDrop = useCallback((acceptedFiles: File[]) => {
    if (acceptedFiles.length > maxFiles) {
      alert(`You can only upload up to ${maxFiles} files at once.`);
      return;
    }

    const oversizedFiles = acceptedFiles.filter(file => file.size > maxFileSize);
    if (oversizedFiles.length > 0) {
      alert(`Some files are too large. Maximum file size is ${formatFileSize(maxFileSize)}.`);
      return;
    }

    handleUpload(acceptedFiles);
  }, [maxFiles, maxFileSize]);

  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    onDrop,
    maxFiles,
    maxSize: maxFileSize,
    accept: acceptedFileTypes ? 
      acceptedFileTypes.reduce((acc, type) => ({ ...acc, [type]: [] }), {}) : 
      undefined,
    disabled: isUploading,
  });

  const removeFile = (index: number) => {
    setUploadFiles(prev => prev.filter((_, i) => i !== index));
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle className="h-5 w-5 text-green-500" />;
      case 'error':
        return <AlertCircle className="h-5 w-5 text-red-500" />;
      case 'uploading':
        return <Loader2 className="h-5 w-5 text-blue-500 animate-spin" />;
      default:
        return <File className="h-5 w-5 text-gray-400" />;
    }
  };

  return (
    <div className="w-full max-w-4xl mx-auto p-6">
      {/* Dropzone */}
      <div
        {...getRootProps()}
        className={`
          border-2 border-dashed rounded-lg p-8 text-center cursor-pointer transition-colors
          ${isDragActive ? 'border-blue-500 bg-blue-50' : 'border-gray-300 hover:border-gray-400'}
          ${isUploading ? 'cursor-not-allowed opacity-50' : ''}
        `}
      >
        <input {...getInputProps()} />
        <Upload className="h-12 w-12 text-gray-400 mx-auto mb-4" />
        <div className="text-lg font-medium text-gray-700 mb-2">
          {isDragActive ? 'Drop files here...' : 'Drag & drop files here'}
        </div>
        <div className="text-sm text-gray-500 mb-4">
          or click to select files
        </div>
        <div className="text-xs text-gray-400">
          Maximum {maxFiles} files, up to {formatFileSize(maxFileSize)} each
        </div>
      </div>

      {/* Upload Progress */}
      {uploadFiles.length > 0 && (
        <div className="mt-6">
          <div className="flex justify-between items-center mb-4">
            <h3 className="text-lg font-medium text-gray-700">
              Uploading {uploadFiles.length} file{uploadFiles.length > 1 ? 's' : ''}
            </h3>
            <div className="text-sm text-gray-500">
              {overallProgress}% complete
            </div>
          </div>

          {/* Overall Progress Bar */}
          <div className="w-full bg-gray-200 rounded-full h-2 mb-6">
            <div
              className="bg-blue-500 h-2 rounded-full transition-all duration-300"
              style={{ width: `${overallProgress}%` }}
            />
          </div>

          {/* Deduplication Savings */}
          {deduplicationSavings > 0 && (
            <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-4">
              <div className="flex items-center">
                <CheckCircle className="h-5 w-5 text-green-500 mr-2" />
                <span className="text-sm text-green-700">
                  Storage saved through deduplication: {formatFileSize(deduplicationSavings)}
                </span>
              </div>
            </div>
          )}

          {/* Individual File Progress */}
          <div className="space-y-3">
            {uploadFiles.map((fileProgress, index) => (
              <div key={index} className="bg-white border border-gray-200 rounded-lg p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center flex-1 min-w-0">
                    {getStatusIcon(fileProgress.status)}
                    <div className="ml-3 flex-1 min-w-0">
                      <div className="text-sm font-medium text-gray-900 truncate">
                        {fileProgress.file.name}
                      </div>
                      <div className="text-xs text-gray-500">
                        {formatFileSize(fileProgress.file.size)}
                        {fileProgress.isExisting && (
                          <span className="ml-2 px-2 py-1 bg-blue-100 text-blue-800 rounded-full text-xs">
                            Deduplicated
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                  
                  <div className="flex items-center ml-4">
                    {fileProgress.status === 'uploading' && (
                      <div className="text-sm text-gray-500 mr-4">
                        {fileProgress.progress}%
                      </div>
                    )}
                    {fileProgress.status === 'pending' && !isUploading && (
                      <button
                        onClick={() => removeFile(index)}
                        className="text-gray-400 hover:text-red-500"
                      >
                        <X className="h-4 w-4" />
                      </button>
                    )}
                  </div>
                </div>

                {/* Progress Bar for Individual File */}
                {fileProgress.status === 'uploading' && (
                  <div className="mt-2">
                    <div className="w-full bg-gray-200 rounded-full h-1">
                      <div
                        className="bg-blue-500 h-1 rounded-full transition-all duration-300"
                        style={{ width: `${fileProgress.progress}%` }}
                      />
                    </div>
                  </div>
                )}

                {/* Error Message */}
                {fileProgress.error && (
                  <div className="mt-2 text-sm text-red-600">
                    Error: {fileProgress.error}
                  </div>
                )}

                {/* Warnings */}
                {fileProgress.warnings && fileProgress.warnings.length > 0 && (
                  <div className="mt-2">
                    {fileProgress.warnings.map((warning, wIndex) => (
                      <div key={wIndex} className="text-sm text-yellow-600">
                        Warning: {warning}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default FileUploadComponent;