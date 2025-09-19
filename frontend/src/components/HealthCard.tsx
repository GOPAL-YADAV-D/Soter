import React, { useState, useEffect } from 'react';
import { healthService, HealthStatus } from '../services/api';

const HealthCard: React.FC = () => {
  const [health, setHealth] = useState<HealthStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchHealth = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await healthService.getHealth();
      setHealth(data);
    } catch (err) {
      setError('Failed to fetch health status');
      console.error('Health check failed:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  if (loading && !health) {
    return (
      <div className="card">
        <h2>System Health</h2>
        <div className="loading">Loading health status...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="card">
        <h2>System Health</h2>
        <div className="error">{error}</div>
        <button className="btn" onClick={fetchHealth}>
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="card">
      <h2>System Health</h2>
      <div className="metric">
        <span className="metric-label">Overall Status:</span>
        <span className={`status ${health?.status}`}>
          {health?.status || 'Unknown'}
        </span>
      </div>
      <div className="metric">
        <span className="metric-label">Database:</span>
        <span className={`status ${health?.database === 'healthy' ? 'healthy' : 'unhealthy'}`}>
          {health?.database || 'Unknown'}
        </span>
      </div>
      <div className="metric">
        <span className="metric-label">Storage:</span>
        <span className={`status ${health?.storage === 'healthy' ? 'healthy' : 'unhealthy'}`}>
          {health?.storage || 'Unknown'}
        </span>
      </div>
      <div className="metric">
        <span className="metric-label">Last Check:</span>
        <span className="metric-value">
          {health?.timestamp ? new Date(health.timestamp).toLocaleString() : 'Unknown'}
        </span>
      </div>
      <button className="btn" onClick={fetchHealth} disabled={loading}>
        {loading ? 'Refreshing...' : 'Refresh'}
      </button>
    </div>
  );
};

export default HealthCard;