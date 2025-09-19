import React from 'react';
import HealthCard from '../components/HealthCard';

const Dashboard: React.FC = () => {
  return (
    <div>
      <header className="header">
        <div className="container">
          <h1>Soter - Secure File Vault</h1>
        </div>
      </header>
      
      <div className="container">
        <div className="dashboard">
          <HealthCard />
          
          <div className="card">
            <h2>Quick Stats</h2>
            <div className="metric">
              <span className="metric-label">Total Files:</span>
              <span className="metric-value">0</span>
            </div>
            <div className="metric">
              <span className="metric-label">Storage Used:</span>
              <span className="metric-value">0 MB</span>
            </div>
            <div className="metric">
              <span className="metric-label">Users:</span>
              <span className="metric-value">1</span>
            </div>
          </div>
          
          <div className="card">
            <h2>Recent Activity</h2>
            <p style={{ color: '#666', fontStyle: 'italic' }}>
              No recent activity to display
            </p>
          </div>
        </div>
        
        <div className="card">
          <h2>Getting Started</h2>
          <p>Welcome to Soter, your secure file vault system. This is a production-ready starter scaffold featuring:</p>
          <ul>
            <li>🔒 Secure file storage with deduplication</li>
            <li>📊 Real-time health monitoring</li>
            <li>🚀 GraphQL API with playground</li>
            <li>🐳 Docker containerization</li>
            <li>📈 Prometheus metrics</li>
            <li>🔍 Structured logging</li>
          </ul>
          <p>
            <strong>API Endpoints:</strong><br />
            GraphQL Playground: <a href="/playground" target="_blank" rel="noopener noreferrer">/playground</a><br />
            Health Check: <a href="/healthz" target="_blank" rel="noopener noreferrer">/healthz</a><br />
            Metrics: <a href="/metrics" target="_blank" rel="noopener noreferrer">/metrics</a>
          </p>
        </div>
      </div>
    </div>
  );
};

export default Dashboard;