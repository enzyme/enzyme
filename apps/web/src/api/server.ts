import { get, type ServerInfo } from '@enzyme/api-client';

export const serverApi = {
  getServerInfo: () => get<ServerInfo>('/server-info'),
};
