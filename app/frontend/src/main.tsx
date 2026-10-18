import React, { lazy, Suspense } from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";
import "./globals.css";

import { Toaster } from "@/components/ui/toaster";
import Providers from "@/components/provider/main-provider";
import AuthProvider from "@/components/provider/auth-provider";
import AppShell from "@/components/provider/app-shell";

// One chunk per page, so the editor's heavy libraries load only on its routes.
const LoginPage = lazy(() => import("@/pages/login/LoginPage"));
const ForgotPasswordPage = lazy(() => import("@/pages/login/PasswordPages").then((m) => ({ default: m.ForgotPasswordPage })));
const SetPasswordPage = lazy(() => import("@/pages/login/PasswordPages").then((m) => ({ default: m.SetPasswordPage })));
const DashboardPage = lazy(() => import("@/pages/dashboard/DashboardPage"));
const ConnectorsPage = lazy(() => import("@/pages/connectors/ConnectorsPage"));
const AlertsPage = lazy(() => import("@/pages/alerts/AlertsPage"));
const AlertsDashboardPage = lazy(() => import("@/pages/alerts/AlertsDashboardPage"));
const AlertDetailPage = lazy(() => import("@/pages/alerts/AlertDetailPage"));
const IncidentsPage = lazy(() => import("@/pages/incidents/IncidentsPage"));
const IncidentsDashboardPage = lazy(() => import("@/pages/incidents/IncidentsDashboardPage"));
const IncidentDetailPage = lazy(() => import("@/pages/incidents/IncidentDetailPage"));
const PlaybooksLayout = lazy(() => import("@/pages/playbooks/PlaybooksLayout"));
const PlaybooksPage = lazy(() => import("@/pages/playbooks/PlaybooksPage"));
const ExecutionsPage = lazy(() => import("@/pages/playbooks/ExecutionsPage"));
const PlaybookEditorPage = lazy(() => import("@/pages/playbooks/PlaybookEditorPage"));
const HistoryLayout = lazy(() => import("@/pages/playbooks/HistoryLayout"));
const HistoryPage = lazy(() => import("@/pages/playbooks/HistoryPage"));
const HistoryDetailPage = lazy(() => import("@/pages/playbooks/HistoryDetailPage"));
const SettingsLayout = lazy(() => import("@/pages/settings/SettingsLayout"));
const SettingsPage = lazy(() => import("@/pages/settings/SettingsPage"));
const UsersPage = lazy(() => import("@/pages/settings/UsersPage"));
const TeamsPage = lazy(() => import("@/pages/settings/TeamsPage"));
const RolesPage = lazy(() => import("@/pages/settings/RolesPage"));
const ProvidersPage = lazy(() => import("@/pages/settings/ProvidersPage"));
const AuditPage = lazy(() => import("@/pages/settings/AuditPage"));
const APIKeysPage = lazy(() => import("@/pages/settings/APIKeysPage"));

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <Providers>
        {/* AuthProvider resolves the session before anything protected
            renders; AppShell then decides whether the page gets the sidebar
            chrome, so /login can render bare. */}
        <AuthProvider>
          <AppShell>
            <Suspense>
              <Routes>
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="login" element={<LoginPage />} />
                <Route path="forgot-password" element={<ForgotPasswordPage />} />
                <Route path="set-password" element={<SetPasswordPage />} />
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="connectors" element={<ConnectorsPage />} />
                <Route path="alerts">
                  <Route index element={<AlertsPage />} />
                  <Route path="dashboard" element={<AlertsDashboardPage />} />
                  <Route path=":alertId" element={<AlertDetailPage />} />
                </Route>
                <Route path="incidents">
                  <Route index element={<IncidentsPage />} />
                  <Route path="dashboard" element={<IncidentsDashboardPage />} />
                  <Route path=":incidentId" element={<IncidentDetailPage />} />
                </Route>
                <Route path="playbooks">
                  <Route element={<PlaybooksLayout />}>
                    <Route index element={<PlaybooksPage />} />
                    <Route path="executions" element={<ExecutionsPage />} />
                  </Route>
                  <Route path=":playbookId" element={<PlaybookEditorPage />} />
                  <Route path=":playbookId/history" element={<HistoryLayout />}>
                    <Route index element={<HistoryPage />} />
                    <Route path=":playbookHistoryId" element={<HistoryDetailPage />} />
                  </Route>
                </Route>
                <Route path="settings" element={<SettingsLayout />}>
                  <Route index element={<SettingsPage />} />
                  <Route path="users" element={<UsersPage />} />
                  <Route path="teams" element={<TeamsPage />} />
                  <Route path="roles" element={<RolesPage />} />
                  <Route path="providers" element={<ProvidersPage />} />
                  <Route path="audit" element={<AuditPage />} />
                  <Route path="api-keys" element={<APIKeysPage />} />
                </Route>
                <Route path="*" element={<Navigate to="/dashboard" replace />} />
              </Routes>
            </Suspense>
          </AppShell>
        </AuthProvider>
      </Providers>
      <Toaster />
    </BrowserRouter>
  </React.StrictMode>,
);
