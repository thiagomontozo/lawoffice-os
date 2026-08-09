import { useEffect, useMemo, useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { Columns3, List, Plus, Search } from "lucide-react";
import { Badge, Button, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api, post } from "../services/api";
import type { Matter } from "../types";
export function MattersPage() {
  const [items, setItems] = useState<Matter[]>([]);
  const [loading, setLoading] = useState(true);
  const [view, setView] = useState<"table" | "kanban">(
    (localStorage.getItem("matters-view") as "table" | "kanban") || "table",
  );
  const [filters, setFilters] = useState(
    () =>
      JSON.parse(
        localStorage.getItem("matter-filters") ??
          '{"q":"","status":"","priority":""}',
      ) as { q: string; status: string; priority: string },
  );
  useEffect(() => {
    localStorage.setItem("matter-filters", JSON.stringify(filters));
    const timer = setTimeout(() => {
      setLoading(true);
      api<{ items: Matter[] }>(
        `/api/v1/matters?q=${encodeURIComponent(filters.q)}&status=${filters.status}&priority=${filters.priority}`,
      )
        .then((x) => setItems(x.items))
        .finally(() => setLoading(false));
    }, 250);
    return () => clearTimeout(timer);
  }, [filters]);
  function change(v: "table" | "kanban") {
    setView(v);
    localStorage.setItem("matters-view", v);
  }
  return (
    <>
      <PageHeader
        title="Matters"
        description="Processos, consultorias, contratos e demais assuntos jurídicos."
        action={
          <a
            href="/app/matters/new"
            className="flex rounded-xl bg-brand-primary px-4 py-2.5 text-sm font-semibold text-white"
          >
            <Plus size={17} className="mr-2" />
            Novo Matter
          </a>
        }
      />
      <Card>
        <div className="grid gap-3 md:grid-cols-[1fr_180px_180px_auto]">
          <label className="flex items-center gap-2 rounded-xl border px-3">
            <Search size={17} />
            <input
              aria-label="Buscar Matters"
              value={filters.q}
              onChange={(e) => setFilters({ ...filters, q: e.target.value })}
              placeholder="Número, título ou processo"
              className="w-full py-2.5 outline-none"
            />
          </label>
          <Select
            label="Status"
            value={filters.status}
            onChange={(v) => setFilters({ ...filters, status: v })}
            options={["", "draft", "active", "on_hold", "closing"]}
          />
          <Select
            label="Prioridade"
            value={filters.priority}
            onChange={(v) => setFilters({ ...filters, priority: v })}
            options={["", "low", "normal", "high", "critical"]}
          />
          <div className="flex rounded-xl border p-1">
            <button
              aria-label="Tabela"
              onClick={() => change("table")}
              className={`rounded-lg p-2 ${view === "table" ? "bg-slate-100" : ""}`}
            >
              <List />
            </button>
            <button
              aria-label="Kanban"
              onClick={() => change("kanban")}
              className={`rounded-lg p-2 ${view === "kanban" ? "bg-slate-100" : ""}`}
            >
              <Columns3 />
            </button>
          </div>
        </div>
      </Card>
      {loading ? (
        <Loading />
      ) : items.length === 0 ? (
        <div className="mt-6">
          <Empty
            title="Nenhum Matter encontrado"
            description="Ajuste os filtros ou crie o primeiro assunto jurídico."
          />
        </div>
      ) : view === "table" ? (
        <MatterTable items={items} />
      ) : (
        <Kanban items={items} />
      )}
    </>
  );
}
function MatterTable({ items }: { items: Matter[] }) {
  return (
    <Card className="mt-6 overflow-x-auto p-0">
      <table className="w-full min-w-[880px] text-left">
        <thead className="border-b bg-slate-50 text-xs uppercase text-slate-500">
          <tr>
            {[
              "Matter",
              "Tipo",
              "Área",
              "Responsável",
              "Status",
              "Prioridade",
            ].map((x) => (
              <th className="px-5 py-4" key={x}>
                {x}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y">
          {items.map((m) => (
            <tr key={m.id} className="hover:bg-slate-50">
              <td className="px-5 py-4">
                <a
                  className="font-semibold text-brand-primary"
                  href={`/app/matters/${m.id}`}
                >
                  {m.title}
                </a>
                <p className="text-xs text-slate-500">
                  {m.internalNumber}
                  {m.caseNumber && ` · ${m.caseNumber}`}
                </p>
              </td>
              <td className="px-5 text-sm">{m.type.replaceAll("_", " ")}</td>
              <td className="px-5 text-sm">{m.legalAreaName ?? "—"}</td>
              <td className="px-5 text-sm">
                {m.responsibleName ?? "Não atribuído"}
              </td>
              <td className="px-5">
                <Badge tone="blue">{m.status}</Badge>
              </td>
              <td className="px-5">
                <Badge
                  tone={
                    m.priority === "critical"
                      ? "red"
                      : m.priority === "high"
                        ? "amber"
                        : "slate"
                  }
                >
                  {m.priority}
                </Badge>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </Card>
  );
}
function Kanban({ items }: { items: Matter[] }) {
  const columns = ["draft", "active", "on_hold", "closing"];
  return (
    <div className="mt-6 grid gap-4 xl:grid-cols-4">
      {columns.map((status) => (
        <section key={status} className="rounded-2xl bg-slate-100 p-3">
          <h2 className="mb-3 flex justify-between px-1 font-semibold">
            {status.replace("_", " ")}
            <Badge>{items.filter((x) => x.status === status).length}</Badge>
          </h2>
          <div className="space-y-3">
            {items
              .filter((x) => x.status === status)
              .map((m) => (
                <a
                  href={`/app/matters/${m.id}`}
                  key={m.id}
                  className="block rounded-xl bg-white p-4 shadow-sm"
                >
                  <p className="font-semibold">{m.title}</p>
                  <p className="mt-2 text-xs text-slate-500">
                    {m.internalNumber}
                  </p>
                  <div className="mt-3">
                    <Badge tone={m.priority === "critical" ? "red" : "slate"}>
                      {m.priority}
                    </Badge>
                  </div>
                </a>
              ))}
          </div>
        </section>
      ))}
    </div>
  );
}
export function NewMatter() {
  const nav = useNavigate();
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    title: "",
    type: "judicial_process",
    internalNumber: `MAT-${new Date().getFullYear()}-`,
    description: "",
    status: "draft",
    priority: "normal",
    confidentiality: "normal",
    openedAt: new Date().toISOString(),
  });
  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      const x = await post<Matter>("/api/v1/matters", form);
      nav(`/app/matters/${x.id}`);
    } catch (x) {
      setError(x instanceof Error ? x.message : "Erro ao criar");
    }
  }
  return (
    <>
      <PageHeader
        title="Novo Matter"
        description="Cadastre um assunto jurídico com contexto, responsabilidade e segurança."
      />
      <Card className="max-w-4xl">
        <form onSubmit={submit} className="grid gap-5 md:grid-cols-2">
          <Input
            label="Título"
            value={form.title}
            set={(title) => setForm({ ...form, title })}
          />
          <Input
            label="Número interno"
            value={form.internalNumber}
            set={(internalNumber) => setForm({ ...form, internalNumber })}
          />
          <Select
            label="Tipo"
            value={form.type}
            onChange={(type) => setForm({ ...form, type })}
            options={[
              "judicial_process",
              "administrative_process",
              "legal_consultation",
              "contract",
              "advisory",
              "arbitration",
              "extrajudicial",
              "internal_legal_project",
              "other",
            ]}
          />
          <Select
            label="Visibilidade"
            value={form.confidentiality}
            onChange={(confidentiality) =>
              setForm({ ...form, confidentiality })
            }
            options={["normal", "team_only", "partners_only", "restricted"]}
          />
          <Select
            label="Prioridade"
            value={form.priority}
            onChange={(priority) => setForm({ ...form, priority })}
            options={["low", "normal", "high", "critical"]}
          />
          <label className="md:col-span-2 text-sm font-medium">
            Descrição
            <textarea
              value={form.description}
              onChange={(e) =>
                setForm({ ...form, description: e.target.value })
              }
              className="mt-1.5 min-h-28 w-full rounded-xl border p-3"
            />
          </label>
          {error && <p className="md:col-span-2 text-red-700">{error}</p>}
          <div className="md:col-span-2">
            <Button>Criar Matter</Button>
          </div>
        </form>
      </Card>
    </>
  );
}
function Select({
  label,
  value,
  onChange,
  options,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  options: string[];
}) {
  return (
    <label className="text-sm font-medium">
      {label}
      <select
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1.5 w-full rounded-xl border bg-white px-3 py-2.5"
      >
        {options.map((x) => (
          <option key={x} value={x}>
            {x ? x.replaceAll("_", " ") : "Todos"}
          </option>
        ))}
      </select>
    </label>
  );
}
function Input({
  label,
  value,
  set,
}: {
  label: string;
  value: string;
  set: (v: string) => void;
}) {
  return (
    <label className="text-sm font-medium">
      {label}
      <input
        required
        value={value}
        onChange={(e) => set(e.target.value)}
        className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
      />
    </label>
  );
}
