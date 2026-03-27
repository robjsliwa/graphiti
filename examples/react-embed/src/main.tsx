import { StrictMode, useMemo } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { GraphitiClient } from '@graphiti/client';
import { GraphitiProvider } from '@graphiti/react';
import { Dashboard } from './components/Dashboard.js';
import { Builder } from './components/Builder.js';
import './styles/app.css';

const GRAPHITI_URL = import.meta.env.VITE_GRAPHITI_URL || 'http://localhost:8080';
const GRAPHITI_TOKEN = import.meta.env.VITE_GRAPHITI_TOKEN || 'dev-token';

function App() {
  const client = useMemo(
    () =>
      new GraphitiClient({
        baseUrl: GRAPHITI_URL,
        token: GRAPHITI_TOKEN,
      }),
    [],
  );

  return (
    <GraphitiProvider client={client}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Dashboard />} />
          <Route path="/workflows/:id" element={<Builder />} />
        </Routes>
      </BrowserRouter>
    </GraphitiProvider>
  );
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
