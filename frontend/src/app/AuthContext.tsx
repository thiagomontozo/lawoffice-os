import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { api } from "../services/api";
import type { Session } from "../types";
type AuthState = {
  session: Session | null;
  loading: boolean;
  refresh: () => Promise<void>;
  logout: () => Promise<void>;
};
const AuthContext = createContext<AuthState | null>(null);
export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [loading, setLoading] = useState(true);
  async function refresh() {
    try {
      setSession(await api<Session>("/api/v1/auth/me"));
    } catch {
      setSession(null);
    } finally {
      setLoading(false);
    }
  }
  useEffect(() => {
    void refresh();
  }, []);
  useEffect(() => {
    const b = session?.branding;
    if (!b) return;
    const root = document.documentElement;
    root.style.setProperty("--brand-primary", b.primaryColor);
    root.style.setProperty("--brand-secondary", b.secondaryColor);
    root.style.setProperty("--brand-accent", b.accentColor);
    document.title = b.systemTitle;
  }, [session]);
  const value = useMemo(
    () => ({
      session,
      loading,
      refresh,
      logout: async () => {
        await api("/api/v1/auth/logout", { method: "POST" });
        setSession(null);
      },
    }),
    [session, loading],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("AuthProvider missing");
  return value;
}
