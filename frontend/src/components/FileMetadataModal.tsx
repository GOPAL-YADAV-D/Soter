import React, { useState, useEffect, useCallback } from 'react';
import { X, Download, User as UserIcon, Clock, Hash, Shield, Users, Copy, Eye, Globe } from 'lucide-react';
import { fileService, FileMetadata, PermissionSet } from '../services/fileService';

interface FileMetadataModalProps {
  fileId: string;
  onClose: () => void;
}

// Utility function to format permission set to rwx string
const formatPermissions = (perms: PermissionSet): string => {
  return `${perms.read ? 'r' : '-'}${perms.write ? 'w' : '-'}${perms.execute ? 'x' : '-'}`;
};

const FileMetadataModal: React.FC<FileMetadataModalProps> = ({ fileId, onClose }) => {
  const [metadata, setMetadata] = useState<FileMetadata | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  const [activeTab, setActiveTab] = useState<'details' | 'permissions' | 'duplicates'>('details');

  const loadMetadata = useCallback(async () => {
    try {
      setLoading(true);
      const data = await fileService.getFileMetadata(fileId);
      setMetadata(data);
    } catch (err: any) {
      console.error('Failed to load file metadata:', err);
      setError(err.response?.data?.error || 'Failed to load file metadata');
    } finally {
      setLoading(false);
    }
  }, [fileId]);

  useEffect(() => {
    loadMetadata();
  }, [loadMetadata]);

  const handleDownload = async () => {
    if (!metadata) return;
    
    try {
      const blob = await fileService.downloadFile(metadata.id);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = metadata.userFilename;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Download failed:', err);
      setError('Failed to download file');
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatPermissions = (permissions: { read: boolean; write: boolean; execute: boolean }) => {
    return [
      permissions.read ? 'r' : '-',
      permissions.write ? 'w' : '-',
      permissions.execute ? 'x' : '-',
    ].join('');
  };

  const getPermissionBadgeColor = (accessLevel: string) => {
    switch (accessLevel) {
      case 'owner': return 'bg-green-100 text-green-800';
      case 'group': return 'bg-blue-100 text-blue-800';
      case 'other': return 'bg-gray-100 text-gray-800';
      default: return 'bg-red-100 text-red-800';
    }
  };

  if (loading) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
        <div className="bg-white rounded-xl shadow-xl max-w-2xl w-full p-6">
          <div className="text-center">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading file metadata...</p>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
        <div className="bg-white rounded-xl shadow-xl max-w-md w-full p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-gray-900">Error</h2>
            <button onClick={onClose} className="text-gray-400 hover:text-gray-600">
              <X className="h-6 w-6" />
            </button>
          </div>
          <div className="text-center">
            <p className="text-red-600 mb-4">{error}</p>
            <button
              onClick={onClose}
              className="bg-gray-600 text-white px-4 py-2 rounded-lg hover:bg-gray-700"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (!metadata) return null;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white rounded-xl shadow-xl max-w-4xl w-full max-h-[90vh] overflow-hidden">
        {/* Header */}
        <div className="flex items-center justify-between p-6 border-b">
          <div className="flex items-center">
            <div className="text-2xl mr-3">
              {fileService.getFileIcon(metadata.contentType)}
            </div>
            <div>
              <h2 className="text-xl font-semibold text-gray-900 truncate max-w-md" title={metadata.userFilename}>
                {metadata.userFilename}
              </h2>
              <p className="text-sm text-gray-600">
                {fileService.formatFileSize(metadata.fileSize)} • {metadata.contentType}
              </p>
            </div>
          </div>
          
          <div className="flex items-center space-x-3">
            {metadata.permissions.canRead && (
              <button
                onClick={handleDownload}
                className="flex items-center space-x-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700"
              >
                <Download className="h-4 w-4" />
                <span>Download</span>
              </button>
            )}
            
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-600"
            >
              <X className="h-6 w-6" />
            </button>
          </div>
        </div>

        {/* Tabs */}
        <div className="border-b">
          <nav className="flex px-6">
            {[
              { id: 'details', label: 'Details', icon: Eye },
              { id: 'permissions', label: 'Permissions', icon: Shield },
              { id: 'duplicates', label: 'Duplicates', icon: Hash },
            ].map(({ id, label, icon: Icon }) => (
              <button
                key={id}
                onClick={() => setActiveTab(id as any)}
                className={`flex items-center space-x-2 px-4 py-3 border-b-2 font-medium text-sm ${
                  activeTab === id
                    ? 'border-indigo-500 text-indigo-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                <Icon className="h-4 w-4" />
                <span>{label}</span>
                {id === 'duplicates' && metadata.duplicateCount && metadata.duplicateCount > 0 && (
                  <span className="bg-red-100 text-red-800 text-xs px-2 py-1 rounded-full">
                    {metadata.duplicateCount}
                  </span>
                )}
              </button>
            ))}
          </nav>
        </div>

        {/* Content */}
        <div className="p-6 overflow-y-auto max-h-[60vh]">
          {activeTab === 'details' && (
            <div className="space-y-6">
              {/* Basic Information */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">File Information</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">File Name</label>
                      <div className="flex items-center space-x-2">
                        <span className="text-sm text-gray-900 break-all">{metadata.userFilename}</span>
                        <button
                          onClick={() => copyToClipboard(metadata.userFilename)}
                          className="text-gray-400 hover:text-gray-600"
                          title="Copy to clipboard"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                      </div>
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Size</label>
                      <span className="text-sm text-gray-900">{fileService.formatFileSize(metadata.fileSize)}</span>
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">MIME Type</label>
                      <span className="text-sm text-gray-900">{metadata.contentType}</span>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Group</label>
                      <span className="text-sm text-indigo-600">{metadata.groups?.[0]?.name || metadata.groupName || 'Personal'}</span>
                    </div>
                  </div>
                  
                  <div className="space-y-3">
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Uploaded By</label>
                      <div className="flex items-center space-x-2">
                        <UserIcon className="h-4 w-4 text-gray-400" />
                        <span className="text-sm text-gray-900">{metadata.owner?.name || 'Unknown'}</span>
                      </div>
                    </div>
                    
                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Upload Date</label>
                      <div className="flex items-center space-x-2">
                        <Clock className="h-4 w-4 text-gray-400" />
                        <span className="text-sm text-gray-900">
                          {new Date(metadata.uploadedAt).toLocaleString()}
                        </span>
                      </div>
                    </div>
                    
                    {metadata.lastAccessed && (
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-1">Last Accessed</label>
                        <div className="flex items-center space-x-2">
                          <Clock className="h-4 w-4 text-gray-400" />
                          <span className="text-sm text-gray-900">
                            {new Date(metadata.lastAccessed).toLocaleString()}
                          </span>
                        </div>
                      </div>
                    )}

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-1">Status</label>
                      <div className="flex items-center space-x-2">
                        {metadata.isOriginal ? (
                          <span className="bg-green-100 text-green-800 text-xs px-2 py-1 rounded-full">
                            Original
                          </span>
                        ) : (
                          <span className="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded-full">
                            Deduplicated
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              {/* File Hash */}
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">File Hash (SHA-256)</h3>
                <div className="bg-gray-50 rounded-lg p-4">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium text-gray-700">SHA-256:</span>
                    <button
                      onClick={() => copyToClipboard(metadata.hash || 'N/A')}
                      className="text-indigo-600 hover:text-indigo-500 text-sm flex items-center space-x-1"
                    >
                      <Copy className="h-4 w-4" />
                      <span>Copy</span>
                    </button>
                  </div>
                  <code className="text-xs text-gray-700 bg-white p-3 rounded border break-all block">
                    {metadata.hash || 'Hash not available'}
                  </code>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'permissions' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">Access Permissions</h3>
                
                {/* User Access Level */}
                <div className="mb-6">
                  <label className="block text-sm font-medium text-gray-700 mb-2">Your Permissions</label>
                  <div className="flex items-center space-x-2">
                    <span className="px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
                      User Access
                    </span>
                    <div className="text-sm text-gray-600">
                      {metadata.permissions.canRead && '• Read '}
                      {metadata.permissions.canWrite && '• Write '}
                      {metadata.permissions.canDownload && '• Download '}
                      {metadata.permissions.canDelete && '• Delete '}
                      {metadata.permissions.canShare && '• Share'}
                    </div>
                  </div>
                </div>

                {/* Linux-style Permissions */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div className="bg-gray-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-gray-900 mb-2 flex items-center">
                      <UserIcon className="h-4 w-4 mr-2" />
                      Owner
                    </h4>
                    <div className="space-y-2">
                      <div className="text-lg font-mono text-gray-900">
                        {formatPermissions(metadata.permissions.owner)}
                      </div>
                      <div className="text-xs text-gray-600 space-y-1">
                        <div className="flex justify-between">
                          <span>Read:</span>
                          <span className={metadata.permissions.owner.read ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.owner.read ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Write:</span>
                          <span className={metadata.permissions.owner.write ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.owner.write ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Execute:</span>
                          <span className={metadata.permissions.owner.execute ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.owner.execute ? 'Yes' : 'No'}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="bg-gray-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-gray-900 mb-2 flex items-center">
                      <Users className="h-4 w-4 mr-2" />
                      Group
                    </h4>
                    <div className="space-y-2">
                      <div className="text-lg font-mono text-gray-900">
                        {formatPermissions(metadata.permissions.group)}
                      </div>
                      <div className="text-xs text-gray-600 space-y-1">
                        <div className="flex justify-between">
                          <span>Read:</span>
                          <span className={metadata.permissions.group.read ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.group.read ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Write:</span>
                          <span className={metadata.permissions.group.write ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.group.write ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Execute:</span>
                          <span className={metadata.permissions.group.execute ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.group.execute ? 'Yes' : 'No'}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div className="bg-gray-50 rounded-lg p-4">
                    <h4 className="text-sm font-medium text-gray-900 mb-2 flex items-center">
                      <Globe className="h-4 w-4 mr-2" />
                      Others
                    </h4>
                    <div className="space-y-2">
                      <div className="text-lg font-mono text-gray-900">
                        {formatPermissions(metadata.permissions.others)}
                      </div>
                      <div className="text-xs text-gray-600 space-y-1">
                        <div className="flex justify-between">
                          <span>Read:</span>
                          <span className={metadata.permissions.others.read ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.others.read ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Write:</span>
                          <span className={metadata.permissions.others.write ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.others.write ? 'Yes' : 'No'}
                          </span>
                        </div>
                        <div className="flex justify-between">
                          <span>Execute:</span>
                          <span className={metadata.permissions.others.execute ? 'text-green-600' : 'text-red-600'}>
                            {metadata.permissions.others.execute ? 'Yes' : 'No'}
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Permission String */}
                <div className="mt-4">
                  <label className="block text-sm font-medium text-gray-700 mb-2">Permission String</label>
                  <div className="bg-gray-100 rounded-lg p-3">
                    <code className="text-sm text-gray-900">
                      {metadata.permissions.octal} ({formatPermissions(metadata.permissions.owner)}{formatPermissions(metadata.permissions.group)}{formatPermissions(metadata.permissions.others)})
                    </code>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'duplicates' && (
            <div className="space-y-6">
              <div>
                <h3 className="text-lg font-medium text-gray-900 mb-4">
                  File Duplicates ({metadata.duplicateCount})
                </h3>
                
                {metadata.duplicateCount === 0 ? (
                  <div className="text-center py-8">
                    <Hash className="h-12 w-12 text-gray-300 mx-auto mb-4" />
                    <p className="text-gray-500">This file has no duplicates</p>
                    <p className="text-gray-400 text-sm mt-2">
                      Files with identical content (same SHA-256 hash) would appear here
                    </p>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                      <p className="text-blue-800 text-sm">
                        <strong>Deduplication Active:</strong> This file shares storage space with {metadata.duplicateCount} other identical file(s).
                        Only one copy is stored on disk, saving space for your organization.
                      </p>
                    </div>

                    {metadata.relatedFiles && metadata.relatedFiles.length > 0 && (
                      <div>
                        <h4 className="text-md font-medium text-gray-900 mb-3">Related Files</h4>
                        <div className="space-y-2">
                          {metadata.relatedFiles.map((relatedFile) => (
                            <div key={relatedFile.id} className="bg-gray-50 rounded-lg p-4 flex items-center justify-between">
                              <div>
                                <div className="text-sm font-medium text-gray-900">
                                  {relatedFile.userFilename}
                                </div>
                                <div className="text-xs text-gray-600">
                                  Uploaded by {relatedFile.uploaderName || 'Unknown'} on {new Date(relatedFile.uploadedAt).toLocaleDateString()}
                                </div>
                              </div>
                              <div className="text-xs text-gray-500">
                                {relatedFile.id === metadata.id ? (
                                  <span className="bg-green-100 text-green-800 px-2 py-1 rounded">Current</span>
                                ) : (
                                  <span className="bg-gray-100 text-gray-800 px-2 py-1 rounded">Duplicate</span>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default FileMetadataModal;