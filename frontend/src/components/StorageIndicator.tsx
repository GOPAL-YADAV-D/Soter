import React from 'react';
import { HardDrive, AlertTriangle } from 'lucide-react';
import { StorageUsage } from '../services/fileService';

interface StorageIndicatorProps {
  storageUsage: StorageUsage | null;
  className?: string;
  showDetails?: boolean;
}

const StorageIndicator: React.FC<StorageIndicatorProps> = ({ 
  storageUsage, 
  className = '',
  showDetails = false 
}) => {
  if (!storageUsage) {
    return null;
  }

  const { usedSpaceMb, allocatedSpaceMb, usagePercentage, fileCount, duplicatesSavedMb, duplicateCount } = storageUsage;

  const getUsageColor = (percentage: number) => {
    if (percentage >= 90) return 'bg-red-500';
    if (percentage >= 75) return 'bg-amber-500';
    if (percentage >= 50) return 'bg-yellow-500';
    return 'bg-green-500';
  };

  const getUsageTextColor = (percentage: number) => {
    if (percentage >= 90) return 'text-red-600';
    if (percentage >= 75) return 'text-amber-600';
    if (percentage >= 50) return 'text-yellow-600';
    return 'text-green-600';
  };

  const formatSize = (sizeMb: number) => {
    if (sizeMb >= 1024) {
      return `${(sizeMb / 1024).toFixed(1)} GB`;
    }
    return `${sizeMb.toFixed(1)} MB`;
  };

  if (showDetails) {
    return (
      <div className={`bg-white rounded-lg shadow-sm border p-6 ${className}`}>
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-medium text-gray-900 flex items-center">
            <HardDrive className="h-5 w-5 mr-2" />
            Storage Usage
          </h3>
          {usagePercentage >= 80 && (
            <AlertTriangle className="h-5 w-5 text-amber-500" />
          )}
        </div>

        {/* Progress Bar */}
        <div className="mb-4">
          <div className="flex justify-between text-sm text-gray-600 mb-2">
            <span>{formatSize(usedSpaceMb)} used</span>
            <span>{formatSize(allocatedSpaceMb)} total</span>
          </div>
          <div className="w-full bg-gray-200 rounded-full h-3">
            <div 
              className={`h-3 rounded-full transition-all duration-300 ${getUsageColor(usagePercentage)}`}
              style={{ width: `${Math.min(usagePercentage, 100)}%` }}
            ></div>
          </div>
          <div className="flex justify-between text-xs text-gray-500 mt-1">
            <span>0%</span>
            <span className={getUsageTextColor(usagePercentage)}>
              {usagePercentage.toFixed(1)}%
            </span>
            <span>100%</span>
          </div>
        </div>

        {/* Storage Stats */}
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="text-gray-600 mb-1">Files</div>
            <div className="text-lg font-semibold text-gray-900">{fileCount.toLocaleString()}</div>
          </div>
          
          <div className="bg-gray-50 rounded-lg p-3">
            <div className="text-gray-600 mb-1">Available</div>
            <div className="text-lg font-semibold text-gray-900">
              {formatSize(allocatedSpaceMb - usedSpaceMb)}
            </div>
          </div>
        </div>

        {/* Deduplication Stats */}
        {duplicateCount > 0 && (
          <div className="mt-4 p-3 bg-green-50 border border-green-200 rounded-lg">
            <div className="flex items-center justify-between">
              <div>
                <div className="text-sm font-medium text-green-800">Space Saved by Deduplication</div>
                <div className="text-xs text-green-600">
                  {duplicateCount} duplicate files • {formatSize(duplicatesSavedMb)} saved
                </div>
              </div>
              <div className="text-lg font-semibold text-green-700">
                {formatSize(duplicatesSavedMb)}
              </div>
            </div>
          </div>
        )}

        {/* Warning for high usage */}
        {usagePercentage >= 90 && (
          <div className="mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
            <div className="flex items-center">
              <AlertTriangle className="h-4 w-4 text-red-500 mr-2" />
              <div>
                <div className="text-sm font-medium text-red-800">Storage Almost Full</div>
                <div className="text-xs text-red-600">
                  Consider deleting unused files or contact your administrator for more space
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    );
  }

  // Compact view for header
  return (
    <div className={`flex items-center space-x-2 ${className}`}>
      <HardDrive className="h-4 w-4 text-gray-500" />
      <div className="flex items-center space-x-2">
        <div className="w-16 bg-gray-200 rounded-full h-2">
          <div 
            className={`h-2 rounded-full transition-all duration-300 ${getUsageColor(usagePercentage)}`}
            style={{ width: `${Math.min(usagePercentage, 100)}%` }}
          ></div>
        </div>
        <span className={`text-xs font-medium ${getUsageTextColor(usagePercentage)}`}>
          {usagePercentage.toFixed(0)}%
        </span>
      </div>
      <span className="text-xs text-gray-600">
        {formatSize(usedSpaceMb)} / {formatSize(allocatedSpaceMb)}
      </span>
      {usagePercentage >= 90 && (
        <AlertTriangle className="h-4 w-4 text-red-500" />
      )}
    </div>
  );
};

export default StorageIndicator;