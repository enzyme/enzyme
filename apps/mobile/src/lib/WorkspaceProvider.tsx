import { createContext, useContext, useState, type ReactNode } from 'react';

interface WorkspaceContextValue {
  activeWorkspaceId: string | null;
  setActiveWorkspaceId: (id: string | null) => void;
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const [activeWorkspaceId, setActiveWorkspaceId] = useState<string | null>(null);

  return (
    <WorkspaceContext.Provider value={{ activeWorkspaceId, setActiveWorkspaceId }}>
      {children}
    </WorkspaceContext.Provider>
  );
}

export function useActiveWorkspace(): WorkspaceContextValue {
  const context = useContext(WorkspaceContext);
  if (!context) {
    throw new Error('useActiveWorkspace must be used within a WorkspaceProvider');
  }
  return context;
}
