import axios from 'axios';

export interface HealthStatus {
  status: string;
  database: string;
  storage: string;
  timestamp: string;
}

export const healthService = {
  async getHealth(): Promise<HealthStatus> {
    const response = await axios.get('/healthz');
    return response.data;
  }
};