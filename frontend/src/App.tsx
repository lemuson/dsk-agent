import React from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import { AssistantProvider } from './context/AssistantContext';
import { Navbar } from './components/layout/Navbar';
import { FloatingAIAssistantWidget } from './components/chat/FloatingAIAssistantWidget';
import { ApartmentsPage } from './pages/ApartmentsPage';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { ProfilePage } from './pages/ProfilePage';
import { ManagerDashboardPage } from './pages/ManagerDashboardPage';
import { SupervisorDashboardPage } from './pages/SupervisorDashboardPage';
import { SupervisorDiscountsPage } from './pages/SupervisorDiscountsPage';
import { DealDetailsPage } from './pages/DealDetailsPage';

const ProtectedUserRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, isAuthenticated, isLoading } = useAuth();
  if (isLoading) return <div className="p-12 text-center text-slate-400">Загрузка...</div>;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
};

const ProtectedManagerRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, isAuthenticated, isLoading } = useAuth();
  if (isLoading) return <div className="p-12 text-center text-slate-400">Загрузка...</div>;
  if (!isAuthenticated || (user?.role !== 'manager' && user?.role !== 'supervisor')) {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
};

const ProtectedSupervisorRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, isAuthenticated, isLoading } = useAuth();
  if (isLoading) return <div className="p-12 text-center text-slate-400">Загрузка...</div>;
  if (!isAuthenticated || user?.role !== 'supervisor') {
    return <Navigate to="/login" replace />;
  }
  return <>{children}</>;
};

export const App: React.FC = () => {
  return (
    <AuthProvider>
      <AssistantProvider>
        <BrowserRouter>
          <div className="min-h-screen flex flex-col bg-[#F8FAFC] text-slate-900">
          <Navbar />
          <main className="flex-1">
            <Routes>

              <Route path="/" element={<ApartmentsPage />} />
              <Route path="/apartments" element={<ApartmentsPage />} />
              <Route path="/apartments/:id" element={<ApartmentsPage />} />

              <Route path="/login" element={<LoginPage />} />
              <Route path="/register" element={<RegisterPage />} />

              <Route
                path="/profile"
                element={
                  <ProtectedUserRoute>
                    <ProfilePage />
                  </ProtectedUserRoute>
                }
              />

              <Route
                path="/deals/:id"
                element={
                  <ProtectedUserRoute>
                    <DealDetailsPage />
                  </ProtectedUserRoute>
                }
              />

              <Route
                path="/manager"
                element={
                  <ProtectedManagerRoute>
                    <ManagerDashboardPage />
                  </ProtectedManagerRoute>
                }
              />

              <Route
                path="/supervisor"
                element={
                  <ProtectedSupervisorRoute>
                    <SupervisorDashboardPage />
                  </ProtectedSupervisorRoute>
                }
              />
              <Route
                path="/supervisor/discounts"
                element={
                  <ProtectedSupervisorRoute>
                    <SupervisorDiscountsPage />
                  </ProtectedSupervisorRoute>
                }
              />

              <Route path="*" element={<Navigate to="/apartments" replace />} />
            </Routes>
          </main>
          <FloatingAIAssistantWidget />
          </div>
        </BrowserRouter>
      </AssistantProvider>
    </AuthProvider>
  );
};

export default App;
