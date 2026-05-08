import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { type ReactElement } from 'react';
import { AuthProvider, useAuth } from './context/AuthContext';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';

function ProtectedRoute({ children }: { children: ReactElement }) {
  const { adminKey } = useAuth();
  return adminKey ? children : <Navigate to="/admin/ui/login" replace />;
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/admin/ui/login" element={<Login />} />
          <Route path="/admin/ui/" element={<ProtectedRoute><Dashboard /></ProtectedRoute>} />
          <Route path="/admin/ui" element={<Navigate to="/admin/ui/" replace />} />
          <Route path="*" element={<Navigate to="/admin/ui/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
