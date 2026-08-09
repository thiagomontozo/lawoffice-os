import { Navigate, Route, Routes } from "react-router-dom";
import { useAuth } from "./AuthContext";
import { Loading } from "./ui";
import { AppLayout } from "../layouts/AppLayout";
import { LoginPage, PortalLogin, SetupPage } from "../pages/AuthPages";
import { Dashboard } from "../pages/Dashboard";
import { MattersPage, NewMatter } from "../pages/Matters";
import { MatterDetailPage } from "../pages/MatterDetail";
import {
  ArchivePage,
  AuditPage,
  CalendarPage,
  ConflictPage,
  DocumentsPage,
  FinancePage,
  WorkflowsPage,
} from "../pages/Operations";
import {
  ClientsPage,
  RolesPage,
  TasksPage,
  UsersPage,
} from "../pages/ManagementPages";
import { BrandingPage } from "../pages/Branding";
import { PortalHome, PortalMatter } from "../pages/Portal";
import { SettingsPage } from "../pages/Settings";
function Protected() {
  const { session, loading } = useAuth();
  if (loading) return <Loading />;
  return session ? <AppLayout /> : <Navigate to="/login" replace />;
}
export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<LoginPage />} />
      <Route path="/setup" element={<SetupPage />} />
      <Route path="/portal/login" element={<PortalLogin />} />
      <Route path="/portal" element={<PortalHome />} />
      <Route path="/portal/matters/:id" element={<PortalMatter />} />
      <Route element={<Protected />}>
        <Route path="/app" element={<Dashboard />} />
        <Route path="/app/matters" element={<MattersPage />} />
        <Route path="/app/matters/new" element={<NewMatter />} />
        <Route path="/app/matters/:id" element={<MatterDetailPage />} />
        <Route path="/app/clients" element={<ClientsPage />} />
        <Route path="/app/clients/:id" element={<ClientsPage />} />
        <Route path="/app/documents" element={<DocumentsPage />} />
        <Route path="/app/calendar" element={<CalendarPage />} />
        <Route path="/app/tasks" element={<TasksPage />} />
        <Route path="/app/workflows" element={<WorkflowsPage />} />
        <Route path="/app/workflows/:id" element={<WorkflowsPage />} />
        <Route path="/app/archive" element={<ArchivePage />} />
        <Route path="/app/conflicts" element={<ConflictPage />} />
        <Route path="/app/finance" element={<FinancePage />} />
        <Route path="/app/users" element={<UsersPage />} />
        <Route path="/app/roles" element={<RolesPage />} />
        <Route path="/app/audit" element={<AuditPage />} />
        <Route path="/app/branding" element={<BrandingPage />} />
        <Route path="/app/settings" element={<SettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
