import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import {
  Archive,
  BriefcaseBusiness,
  CalendarDays,
  ChevronDown,
  FileStack,
  Gavel,
  LayoutDashboard,
  Menu,
  Search,
  Settings,
  ShieldCheck,
  Users,
  Workflow,
  WalletCards,
  X,
  Zap,
} from "lucide-react";
import { useAuth } from "../app/AuthContext";
import { api } from "../services/api";
import { useRealtime } from "../hooks/useRealtime";
const links = [
  ["/app", "Command Center", LayoutDashboard],
  ["/app/matters", "Matters", BriefcaseBusiness],
  ["/app/clients", "Clientes", Users],
  ["/app/documents", "Documentos", FileStack],
  ["/app/calendar", "Calendário", CalendarDays],
  ["/app/tasks", "Tarefas", Gavel],
  ["/app/workflows", "Workflows", Workflow],
  ["/app/archive", "Arquivo Jurídico", Archive],
  ["/app/conflicts", "Conflict Check", ShieldCheck],
  ["/app/finance", "Financeiro", WalletCards],
] as const;
export function AppLayout() {
  const { session, logout } = useAuth();
  const [mobile, setMobile] = useState(false);
  const [palette, setPalette] = useState(false);
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<
    { id: string; type: string; title: string; subtitle: string }[]
  >([]);
  const navigate = useNavigate();
  const realtime = useRealtime();
  useEffect(() => {
    const listener = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setPalette(true);
      }
    };
    window.addEventListener("keydown", listener);
    return () => window.removeEventListener("keydown", listener);
  }, []);
  useEffect(() => {
    if (query.length < 2) {
      setResults([]);
      return;
    }
    const timer = setTimeout(
      () =>
        void api<{ items: typeof results }>(
          `/api/v1/search?q=${encodeURIComponent(query)}`,
        )
          .then((x) => setResults(x.items))
          .catch(() => setResults([])),
      250,
    );
    return () => clearTimeout(timer);
  }, [query]);
  return (
    <div className="min-h-screen bg-slate-50">
      <aside
        className={`fixed inset-y-0 left-0 z-40 w-72 bg-slate-950 text-slate-200 transition-transform lg:translate-x-0 ${mobile ? "translate-x-0" : "-translate-x-full"}`}
      >
        <div className="flex h-20 items-center justify-between border-b border-white/10 px-6">
          <div>
            <p className="text-xs uppercase tracking-[.22em] text-brand-accent">
              {session?.branding.firmDisplayName}
            </p>
            <h2 className="mt-1 font-bold text-white">
              {session?.branding.systemTitle}
            </h2>
          </div>
          <button
            className="lg:hidden"
            onClick={() => setMobile(false)}
            aria-label="Fechar menu"
          >
            <X />
          </button>
        </div>
        <nav className="scrollbar h-[calc(100vh-10rem)] overflow-y-auto p-4">
          {links.map(([to, label, Icon]) => (
            <NavLink
              end={to === "/app"}
              key={to}
              to={to}
              onClick={() => setMobile(false)}
              className={({ isActive }) =>
                `mb-1 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium ${isActive ? "bg-white/12 text-white" : "text-slate-400 hover:bg-white/5 hover:text-white"}`
              }
            >
              <Icon size={18} />
              {label}
            </NavLink>
          ))}
          <p className="mb-2 mt-6 px-3 text-xs uppercase tracking-widest text-slate-600">
            Administração
          </p>
          {(
            [
              ["/app/users", "Usuários", Users],
              ["/app/roles", "Papéis", ShieldCheck],
              ["/app/audit", "Auditoria", Gavel],
              ["/app/branding", "Brand Studio", Zap],
              ["/app/settings", "Configurações", Settings],
            ] as const
          ).map(([to, label, Icon]) => (
            <NavLink
              key={String(to)}
              to={String(to)}
              className={({ isActive }) =>
                `mb-1 flex items-center gap-3 rounded-xl px-3 py-2.5 text-sm ${isActive ? "bg-white/12 text-white" : "text-slate-400 hover:text-white"}`
              }
            >
              <Icon size={18} />
              {String(label)}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="lg:pl-72">
        <header className="sticky top-0 z-30 flex h-20 items-center gap-3 border-b border-slate-200 bg-white/95 px-4 backdrop-blur md:px-7">
          <button
            className="lg:hidden"
            onClick={() => setMobile(true)}
            aria-label="Abrir menu"
          >
            <Menu />
          </button>
          <button
            onClick={() => setPalette(true)}
            className="flex min-w-0 flex-1 items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 px-4 py-2.5 text-left text-sm text-slate-500 md:max-w-xl"
          >
            <Search size={17} />
            Pesquisar Matter, cliente ou documento{" "}
            <kbd className="ml-auto hidden rounded border bg-white px-2 py-0.5 text-xs md:block">
              Ctrl K
            </kbd>
          </button>
          <button className="hidden rounded-xl bg-brand-accent px-4 py-2.5 text-sm font-semibold text-slate-950 md:flex">
            <Zap size={16} className="mr-2" />
            Criar
          </button>
          <button
            onClick={() => void logout().then(() => navigate("/login"))}
            className="flex items-center gap-2 text-sm font-medium"
          >
            <span className="hidden sm:block">{session?.user.name}</span>
            <ChevronDown size={16} />
          </button>
          <span
            className={`hidden h-2.5 w-2.5 rounded-full sm:block ${realtime.connected ? "bg-emerald-500" : "bg-amber-400"}`}
            title={
              realtime.connected
                ? "Atualizações em tempo real conectadas"
                : "Reconectando atualizações"
            }
            aria-label={
              realtime.connected
                ? "Tempo real conectado"
                : "Tempo real reconectando"
            }
          />
        </header>
        <main className="mx-auto max-w-[1500px] p-4 md:p-7">
          <Outlet />
        </main>
      </div>
      {palette && (
        <div
          className="fixed inset-0 z-50 bg-slate-950/50 p-4 pt-[10vh] backdrop-blur-sm"
          onMouseDown={() => setPalette(false)}
        >
          <div
            className="mx-auto max-w-2xl overflow-hidden rounded-2xl bg-white shadow-2xl"
            onMouseDown={(e) => e.stopPropagation()}
          >
            <div className="flex items-center gap-3 border-b p-4">
              <Search />
              <input
                autoFocus
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Buscar ou digitar um comando…"
                className="w-full border-0 text-lg outline-none"
              />
              <button onClick={() => setPalette(false)}>
                <X />
              </button>
            </div>
            <div className="max-h-96 overflow-y-auto p-2">
              <Command
                label="Criar novo Matter"
                onClick={() => navigate("/app/matters/new")}
              />
              <Command
                label="Criar novo cliente"
                onClick={() => navigate("/app/clients?create=true")}
              />
              <Command
                label="Abrir calendário"
                onClick={() => navigate("/app/calendar")}
              />
              {results.map((item) => (
                <Command
                  key={`${item.type}-${item.id}`}
                  label={item.title}
                  detail={`${item.type} · ${item.subtitle}`}
                  onClick={() =>
                    navigate(
                      item.type === "matter"
                        ? `/app/matters/${item.id}`
                        : `/app/${item.type}s`,
                    )
                  }
                />
              ))}
            </div>
          </div>
        </div>
      )}
      {realtime.latest && (
        <div
          role="status"
          className="fixed bottom-5 right-5 z-50 max-w-sm rounded-xl bg-slate-950 px-4 py-3 text-sm text-white shadow-2xl"
        >
          Atualização recebida: {realtime.latest.resourceType}
        </div>
      )}
    </div>
  );
}
function Command({
  label,
  detail,
  onClick,
}: {
  label: string;
  detail?: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className="flex w-full items-center justify-between rounded-xl px-4 py-3 text-left hover:bg-slate-100"
    >
      <span className="font-medium">{label}</span>
      {detail && <span className="text-xs text-slate-500">{detail}</span>}
    </button>
  );
}
