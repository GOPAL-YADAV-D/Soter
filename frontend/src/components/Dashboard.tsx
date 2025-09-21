import React, { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Upload, 
  Search, 
  Grid, 
  List, 
  Download, 
  Trash2, 
  Info, 
  Building, 
  User as UserIcon, 
  LogOut,
  Folder
} from 'lucide-react';
import { authService, User, Organization } from '../services/authService';
import { fileService, FileInfo, StorageUsage, FileListResponse } from '../services/fileService';
import FileUpload from './FileUpload';
import FileMetadataModal from './FileMetadataModal';
import StorageIndicator from './StorageIndicator';

interface DashboardProps {}

const Dashboard: React.FC<DashboardProps> = () => {
  const navigate = useNavigate();
  const [user, setUser] = useState<User | null>(null);
  const [organization, setOrganization] = useState<Organization | null>(null);
  const [files, setFiles] = useState<FileInfo[]>([]);
  const [storageUsage, setStorageUsage] = useState<StorageUsage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string>('');
  
  // UI State
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedMimeType, setSelectedMimeType] = useState('');
  const [sortBy, setSortBy] = useState<'userFilename' | 'fileSize' | 'uploadedAt'>('uploadedAt');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize] = useState(20);
  const [totalPages, setTotalPages] = useState(1);
  
  // Modal states
  const [showUploadModal, setShowUploadModal] = useState(false);
  const [selectedFileId, setSelectedFileId] = useState<string | null>(null);
  const [showMetadataModal, setShowMetadataModal] = useState(false);
  
  // Define functions first
  const loadFiles = useCallback(async () => {
    try {
      const response: FileListResponse = await fileService.getFiles({
        page: currentPage,
        pageSize,
        search: searchTerm || undefined,
        mimeType: selectedMimeType || undefined,
        sortBy,
        sortOrder,
      });

      setFiles(response.files);
      setStorageUsage(response.storageUsage);
      setTotalPages(response.pagination.totalPages);
    } catch (err: any) {
      console.error('Failed to load files:', err);
      setError('Failed to load files');
    }
  }, [currentPage, pageSize, searchTerm, selectedMimeType, sortBy, sortOrder]);

  const loadDashboardData = useCallback(async () => {
    try {
      setLoading(true);
      
      // Check authentication
      if (!authService.isAuthenticated()) {
        navigate('/login');
        return;
      }

      // Load user and organization from localStorage (fast)
      const currentUser = authService.getCurrentUser();
      const currentOrg = authService.getCurrentOrganization();
      
      if (currentUser && currentOrg) {
        setUser(currentUser);
        setOrganization(currentOrg);
      }

      // Load fresh profile data
      try {
        const profile = await authService.getProfile();
        setUser(profile.user);
        setOrganization(profile.organization);
        setStorageUsage(profile.storageUsage);
        
        // Update localStorage with fresh data
        localStorage.setItem('user', JSON.stringify(profile.user));
        localStorage.setItem('organization', JSON.stringify(profile.organization));
      } catch (profileError) {
        console.error('Failed to load profile:', profileError);
        // Continue with cached data if profile fetch fails
      }

      // Load files
      await loadFiles();
      
    } catch (err: any) {
      console.error('Dashboard load error:', err);
      if (err.response?.status === 401) {
        navigate('/login');
      } else {
        setError('Failed to load dashboard data');
      }
    } finally {
      setLoading(false);
    }
  }, [navigate, loadFiles]);

  // Load initial data
  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  // Load files when filters change
  useEffect(() => {
    loadFiles();
  }, [loadFiles]);

  const handleLogout = async () => {
    try {
      await authService.logout();
      navigate('/login');
    } catch (err) {
      // Even if logout fails, redirect to login
      navigate('/login');
    }
  };

  const handleFileUploadComplete = () => {
    setShowUploadModal(false);
    loadFiles(); // Refresh files list
  };

  const handleDownloadFile = async (fileId: string, fileName: string) => {
    try {
      const blob = await fileService.downloadFile(fileId);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = fileName;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      window.URL.revokeObjectURL(url);
    } catch (err) {
      console.error('Download failed:', err);
      setError('Failed to download file');
    }
  };

  const handleDeleteFile = async (fileId: string) => {
    if (!window.confirm('Are you sure you want to delete this file?')) {
      return;
    }

    try {
      await fileService.deleteFile(fileId);
      await loadFiles(); // Refresh files list
    } catch (err) {
      console.error('Delete failed:', err);
      setError('Failed to delete file');
    }
  };

  const handleShowMetadata = (fileId: string) => {
    setSelectedFileId(fileId);
    setShowMetadataModal(true);
  };

  const mimeTypeOptions = [
    { value: '', label: 'All Types' },
    { value: 'image/', label: 'Images' },
    { value: 'video/', label: 'Videos' },
    { value: 'audio/', label: 'Audio' },
    { value: 'application/pdf', label: 'PDF Documents' },
    { value: 'text/', label: 'Text Files' },
    { value: 'application/zip', label: 'Archives' },
  ];

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
          <p className="text-gray-600">Loading dashboard...</p>
        </div>
      </div>
    );
  }

  if (!user || !organization) {
    return null; // Will redirect to login
  }

  return (
    <div className="min-h-screen bg-gray-50">
      {/* Header */}
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center h-16">
            <div className="flex items-center">
              <Building className="h-8 w-8 text-indigo-600 mr-3" />
              <div>
                <h1 className="text-lg font-semibold text-gray-900">
                  {organization.name}
                </h1>
                <p className="text-sm text-gray-500">File Vault</p>
              </div>
            </div>

            <div className="flex items-center space-x-4">
              <StorageIndicator storageUsage={storageUsage} />
              
              <div className="flex items-center space-x-2 text-sm text-gray-700">
                <UserIcon className="h-4 w-4" />
                <span>{user.name}</span>
              </div>

              <button
                onClick={handleLogout}
                className="flex items-center space-x-1 px-3 py-2 text-sm text-gray-700 hover:text-gray-900 hover:bg-gray-100 rounded-md"
              >
                <LogOut className="h-4 w-4" />
                <span>Logout</span>
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-600">{error}</p>
            <button
              onClick={() => setError('')}
              className="text-red-500 underline text-sm mt-1"
            >
              Dismiss
            </button>
          </div>
        )}

        {/* Action Bar */}
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-6 space-y-4 sm:space-y-0">
          <h2 className="text-2xl font-bold text-gray-900">My Files</h2>
          
          <button
            onClick={() => setShowUploadModal(true)}
            className="flex items-center space-x-2 bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700"
          >
            <Upload className="h-5 w-5" />
            <span>Upload Files</span>
          </button>
        </div>

        {/* Filters and Search */}
        <div className="bg-white rounded-lg shadow-sm border p-4 mb-6">
          <div className="flex flex-col lg:flex-row lg:items-center space-y-4 lg:space-y-0 lg:space-x-4">
            {/* Search */}
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-3 h-5 w-5 text-gray-400" />
                <input
                  type="text"
                  placeholder="Search files..."
                  className="pl-10 w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                />
              </div>
            </div>

            {/* File Type Filter */}
            <div className="w-full lg:w-48">
              <select
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                value={selectedMimeType}
                onChange={(e) => setSelectedMimeType(e.target.value)}
              >
                {mimeTypeOptions.map(option => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </select>
            </div>

            {/* Sort */}
            <div className="flex space-x-2">
              <select
                className="p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                value={sortBy}
                onChange={(e) => setSortBy(e.target.value as 'userFilename' | 'fileSize' | 'uploadedAt')}
              >
                <option value="uploadedAt">Date</option>
                <option value="userFilename">Name</option>
                <option value="fileSize">Size</option>
              </select>

              <select
                className="p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                value={sortOrder}
                onChange={(e) => setSortOrder(e.target.value as 'asc' | 'desc')}
              >
                <option value="desc">Newest First</option>
                <option value="asc">Oldest First</option>
              </select>
            </div>

            {/* View Toggle */}
            <div className="flex border border-gray-300 rounded-lg overflow-hidden">
              <button
                className={`p-2 ${viewMode === 'grid' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'}`}
                onClick={() => setViewMode('grid')}
              >
                <Grid className="h-5 w-5" />
              </button>
              <button
                className={`p-2 ${viewMode === 'list' ? 'bg-indigo-600 text-white' : 'bg-white text-gray-600 hover:bg-gray-50'}`}
                onClick={() => setViewMode('list')}
              >
                <List className="h-5 w-5" />
              </button>
            </div>
          </div>
        </div>

        {/* Files List/Grid */}
        {files.length === 0 ? (
          <div className="text-center py-12">
            <Folder className="h-16 w-16 text-gray-300 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No files found</h3>
            <p className="text-gray-500 mb-6">
              {searchTerm || selectedMimeType ? 'Try adjusting your search filters' : 'Upload your first file to get started'}
            </p>
            {!searchTerm && !selectedMimeType && (
              <button
                onClick={() => setShowUploadModal(true)}
                className="bg-indigo-600 text-white px-4 py-2 rounded-lg hover:bg-indigo-700"
              >
                Upload Files
              </button>
            )}
          </div>
        ) : (
          <>
            {viewMode === 'grid' ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                {files.map((file) => (
                  <div key={file.id} className="bg-white rounded-lg shadow-sm border hover:shadow-md transition-shadow">
                    <div className="p-4">
                      <div className="flex items-center justify-between mb-3">
                        <div className="text-2xl">{fileService.getFileIcon(file.contentType)}</div>
                        <div className="flex space-x-1">
                          <button
                            onClick={() => handleShowMetadata(file.id)}
                            className="p-1 text-gray-400 hover:text-gray-600"
                            title="File Info"
                          >
                            <Info className="h-4 w-4" />
                          </button>
                          <button
                            onClick={() => handleDownloadFile(file.id, file.userFilename)}
                            className="p-1 text-gray-400 hover:text-gray-600"
                            title="Download"
                          >
                            <Download className="h-4 w-4" />
                          </button>
                          {file.permissions.canDelete && (
                            <button
                              onClick={() => handleDeleteFile(file.id)}
                              className="p-1 text-gray-400 hover:text-red-600"
                              title="Delete"
                            >
                              <Trash2 className="h-4 w-4" />
                            </button>
                          )}
                        </div>
                      </div>
                      
                      <h3 className="font-medium text-gray-900 truncate mb-1" title={file.userFilename}>
                        {file.userFilename}
                      </h3>
                      
                      <div className="text-sm text-gray-500 space-y-1">
                        <p>{fileService.formatFileSize(file.fileSize)}</p>
                        <p>{fileService.formatFileDate(file.uploadedAt)}</p>
                        <p>by {file.owner?.name || 'Unknown'}</p>
                        {file.groups && file.groups.length > 0 && (
                          <p className="text-indigo-600">Group: {file.groups[0].name}</p>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="bg-white rounded-lg shadow-sm border overflow-hidden">
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          File
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Size
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Uploaded
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Group
                        </th>
                        <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Actions
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {files.map((file) => (
                        <tr key={file.id} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center">
                              <div className="text-xl mr-3">
                                {fileService.getFileIcon(file.contentType)}
                              </div>
                              <div>
                                <div className="text-sm font-medium text-gray-900 truncate max-w-xs" title={file.userFilename}>
                                  {file.userFilename}
                                </div>
                                <div className="text-sm text-gray-500">
                                  by {file.owner?.name || 'Unknown'}
                                </div>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {fileService.formatFileSize(file.fileSize)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {fileService.formatFileDate(file.uploadedAt)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-indigo-600">
                            {file.groups && file.groups.length > 0 ? file.groups[0].name : 'Personal'}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                            <div className="flex justify-end space-x-2">
                              <button
                                onClick={() => handleShowMetadata(file.id)}
                                className="text-gray-400 hover:text-gray-600"
                                title="File Info"
                              >
                                <Info className="h-4 w-4" />
                              </button>
                              <button
                                onClick={() => handleDownloadFile(file.id, file.userFilename)}
                                className="text-gray-400 hover:text-gray-600"
                                title="Download"
                              >
                                <Download className="h-4 w-4" />
                              </button>
                              {file.permissions.canDelete && (
                                <button
                                  onClick={() => handleDeleteFile(file.id)}
                                  className="text-gray-400 hover:text-red-600"
                                  title="Delete"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </button>
                              )}
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Pagination */}
            {totalPages > 1 && (
              <div className="flex items-center justify-between mt-6">
                <div className="text-sm text-gray-700">
                  Page {currentPage} of {totalPages}
                </div>
                <div className="flex space-x-2">
                  <button
                    onClick={() => setCurrentPage(Math.max(1, currentPage - 1))}
                    disabled={currentPage === 1}
                    className="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Previous
                  </button>
                  <button
                    onClick={() => setCurrentPage(Math.min(totalPages, currentPage + 1))}
                    disabled={currentPage === totalPages}
                    className="px-3 py-2 text-sm font-medium text-gray-500 bg-white border border-gray-300 rounded-md hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
                  >
                    Next
                  </button>
                </div>
              </div>
            )}
          </>
        )}
      </main>

      {/* Upload Modal */}
      {showUploadModal && (
        <FileUpload
          onClose={() => setShowUploadModal(false)}
          onUploadComplete={handleFileUploadComplete}
          storageUsage={storageUsage}
        />
      )}

      {/* File Metadata Modal */}
      {showMetadataModal && selectedFileId && (
        <FileMetadataModal
          fileId={selectedFileId}
          onClose={() => {
            setShowMetadataModal(false);
            setSelectedFileId(null);
          }}
        />
      )}
    </div>
  );
};

export default Dashboard;