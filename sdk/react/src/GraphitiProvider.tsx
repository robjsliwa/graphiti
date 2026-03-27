import type { ReactNode } from 'react';
import type { GraphitiClient } from '@graphiti/client';
import { GraphitiContext } from './hooks.js';

export interface GraphitiProviderProps {
  client: GraphitiClient;
  children: ReactNode;
}

export function GraphitiProvider({ client, children }: GraphitiProviderProps) {
  return (
    <GraphitiContext.Provider value={client}>
      {children}
    </GraphitiContext.Provider>
  );
}
