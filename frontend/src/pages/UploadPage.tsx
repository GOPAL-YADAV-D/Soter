import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { HardDrive, Folder, ArrowLeft } from 'lucide-react';
import FileUploadComponent from '../components/FileUpload/FileUploadComponent';

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

const UploadPage: React.FC = () => {
  const navigate = useNavigate();
  const [currentFolder, setCurrentFolder] = useState('/');
  const [uploadResults, setUploadResults] = useState<FileUploadProgress[]>([]);
  const [showResults, setShowResults] = useState(false);

  const handleUploadComplete = (files: FileUploadProgress[]) => {
    setUploadResults(files);
    setShowResults(true);
  };

  const formatFileSize = (bytes: number): string => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  };

  const getTotalSavings = () => {
    return uploadResults.reduce((total, file) => total + (file.savingsBytes || 0), 0);
  };

  const getSuccessfulUploads = () => {
    return uploadResults.filter(file => file.status === 'completed').length;
  };

  const getFailedUploads = () => {
    return uploadResults.filter(file => file.status === 'error').length;
  };

  const getDeduplicatedFiles = () => {
    return uploadResults.filter(file => file.isExisting).length;
  };

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <button
                onClick={() => navigate('/dashboard')}
                className="flex items-center text-gray-600 hover:text-gray-900 mr-4"
              >
                <ArrowLeft className="h-5 w-5 mr-1" />
                Back to Dashboard
              </button>
              <h1 className="text-xl font-semibold text-gray-900">Upload Files</h1>
            </div>
            <div className="flex items-center text-sm text-gray-500">
              <Folder className="h-4 w-4 mr-1" />
              {currentFolder}
            </div>
          </div>
        </div>
      </div>

      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {!showResults ? (
          <>
            {/* Upload Instructions */}
            <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
              <h2 className="text-lg font-medium text-gray-900 mb-4">Upload Files to Your Vault</h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 text-sm text-gray-600">
                <div className="flex items-start">
                  <HardDrive className="h-5 w-5 text-blue-500 mr-2 mt-0.5" />
                  <div>
                    <div className="font-medium">Smart Deduplication</div>
                    <div>Identical files are automatically detected and deduplicated to save storage space.</div>
                  </div>
                </div>
                <div className="flex items-start">
                  <Folder className="h-5 w-5 text-green-500 mr-2 mt-0.5" />
                  <div>
                    <div className="font-medium">Secure Upload</div>
                    <div>Files are validated for security threats and encrypted in transit.</div>
                  </div>
                </div>
                <div className="flex items-start">
                  <ArrowLeft className="h-5 w-5 text-purple-500 mr-2 mt-0.5 transform rotate-180" />
                  <div>
                    <div className="font-medium">Progress Tracking</div>
                    <div>Real-time upload progress with detailed status for each file.</div>
                  </div>
                </div>
              </div>
            </div>

            {/* Folder Selector */}
            <div className="bg-white rounded-lg shadow-sm p-6 mb-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Upload Destination</h3>
              <div className="flex items-center">
                <Folder className="h-5 w-5 text-gray-400 mr-2" />
                <input
                  type="text"
                  value={currentFolder}
                  onChange={(e) => setCurrentFolder(e.target.value)}
                  className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                  placeholder="Enter folder path (e.g., /documents/projects)"
                />
              </div>
              <p className="text-xs text-gray-500 mt-2">
                Files will be uploaded to this folder. Leave as "/" for root directory.
              </p>
            </div>

            {/* File Upload Component */}
            <div className="bg-white rounded-lg shadow-sm">
              <FileUploadComponent
                onUploadComplete={handleUploadComplete}
                folderPath={currentFolder}
                maxFiles={10}
                maxFileSize={100 * 1024 * 1024 * 1024} // 100GB
              />
            </div>
          </>
        ) : (
          /* Upload Results */
          <div className="space-y-6">
            {/* Summary */}
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h2 className="text-lg font-medium text-gray-900 mb-4">Upload Complete</h2>
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                <div className="text-center p-4 bg-green-50 rounded-lg">
                  <div className="text-2xl font-bold text-green-600">{getSuccessfulUploads()}</div>
                  <div className="text-sm text-green-600">Successful</div>
                </div>
                <div className="text-center p-4 bg-red-50 rounded-lg">
                  <div className="text-2xl font-bold text-red-600">{getFailedUploads()}</div>
                  <div className="text-sm text-red-600">Failed</div>
                </div>
                <div className="text-center p-4 bg-blue-50 rounded-lg">
                  <div className="text-2xl font-bold text-blue-600">{getDeduplicatedFiles()}</div>
                  <div className="text-sm text-blue-600">Deduplicated</div>
                </div>
                <div className="text-center p-4 bg-purple-50 rounded-lg">
                  <div className="text-2xl font-bold text-purple-600">{formatFileSize(getTotalSavings())}</div>
                  <div className="text-sm text-purple-600">Space Saved</div>
                </div>
              </div>
            </div>

            {/* Detailed Results */}
            <div className="bg-white rounded-lg shadow-sm p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-4">Upload Details</h3>
              <div className="space-y-3">
                {uploadResults.map((file, index) => (
                  <div key={index} className="flex items-center justify-between p-4 border border-gray-200 rounded-lg">
                    <div className="flex-1 min-w-0">
                      <div className="text-sm font-medium text-gray-900 truncate">
                        {file.file.name}
                      </div>
                      <div className="text-xs text-gray-500">
                        {formatFileSize(file.file.size)}
                        {file.isExisting && (
                          <span className="ml-2 px-2 py-1 bg-blue-100 text-blue-800 rounded-full text-xs">
                            Deduplicated - Saved {formatFileSize(file.savingsBytes || 0)}
                          </span>
                        )}
                      </div>
                      {file.warnings && file.warnings.length > 0 && (
                        <div className="text-xs text-yellow-600 mt-1">
                          {file.warnings.join(', ')}
                        </div>
                      )}
                      {file.error && (
                        <div className="text-xs text-red-600 mt-1">
                          Error: {file.error}
                        </div>
                      )}
                    </div>
                    <div className={`px-3 py-1 rounded-full text-xs font-medium ${
                      file.status === 'completed' ? 'bg-green-100 text-green-800' :
                      file.status === 'error' ? 'bg-red-100 text-red-800' :
                      'bg-gray-100 text-gray-800'
                    }`}>
                      {file.status.charAt(0).toUpperCase() + file.status.slice(1)}
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Actions */}
            <div className="flex justify-between">
              <button
                onClick={() => {
                  setShowResults(false);
                  setUploadResults([]);
                }}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              >
                Upload More Files
              </button>
              <button
                onClick={() => navigate('/dashboard')}
                className="px-4 py-2 bg-blue-600 border border-transparent rounded-md text-sm font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              >
                View My Files
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default UploadPage;