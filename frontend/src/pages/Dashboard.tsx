import { useEffect, useState } from "react";
import {
  AlertTriangle,
  Archive,
  CalendarClock,
  CheckCircle2,
  FileText,
  Gavel,
  Scale,
  Users,
} from "lucide-react";
import { Badge, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api } from "../services/api";
import type {
  Deadline,
  DocumentItem,
  Matter,
  MatterEvent,
  Task,
} from "../types";
type Center = {
  criticalDeadlines: Deadline[];
  todayDeadlines: Deadline[];
  hearings: {
    id: string;
    title: string;
    matterTitle: string;
    scheduledAt: string;
  }[];
  myTasks: Task[];
  recentDocuments: DocumentItem[];
  priorityMatters: Matter[];
  recentActivity: MatterEvent[];
  archiveReady: number;
};
export function Dashboard() {
  const [data, setData] = useState<Center>();
  const [error, setError] = useState("");
  useEffect(() => {
    void api<Center>("/api/v1/dashboard")
      .then(setData)
      .catch((e) => setError(e instanceof Error ? e.message : "Erro"));
  }, []);
  if (!data && !error) return <Loading />;
  return (
    <>
      <PageHeader
        title="Command Center"
        description="O que exige ação agora — sem gráficos decorativos."
        action={
          <a
            href="/app/matters/new"
            className="rounded-xl bg-brand-primary px-4 py-2.5 text-sm font-semibold text-white"
          >
            Novo Matter
          </a>
        }
      />
      {error && <Card className="border-red-200 text-red-700">{error}</Card>}
      {data && (
        <>
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <Metric
              icon={<AlertTriangle />}
              label="Prazos críticos"
              value={data.criticalDeadlines.length}
              tone="red"
            />
            <Metric
              icon={<CalendarClock />}
              label="Prazos hoje"
              value={data.todayDeadlines.length}
              tone="amber"
            />
            <Metric
              icon={<Gavel />}
              label="Audiências (7 dias)"
              value={data.hearings.length}
              tone="blue"
            />
            <Metric
              icon={<Archive />}
              label="Prontos para arquivo"
              value={data.archiveReady}
              tone="green"
            />
          </div>
          <div className="mt-6 grid gap-6 xl:grid-cols-[1.25fr_.75fr]">
            <Card>
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-bold">Meu Dia</h2>
                <Badge tone="blue">foco pessoal</Badge>
              </div>
              <p className="mb-5 mt-1 text-sm text-slate-500">
                Ordenado por criticidade, atraso e proximidade.
              </p>
              {data.myTasks.length === 0 ? (
                <Empty
                  title="Dia organizado"
                  description="Nenhuma tarefa pendente está atribuída a você."
                />
              ) : (
                <div className="space-y-2">
                  {data.myTasks.slice(0, 7).map((t) => (
                    <div
                      key={t.id}
                      className="flex items-center gap-3 rounded-xl border p-3"
                    >
                      <CheckCircle2 className="text-slate-300" />
                      <div className="min-w-0 flex-1">
                        <p className="truncate font-medium">{t.title}</p>
                        <p className="text-xs text-slate-500">
                          {t.matterTitle ?? "Atividade interna"} ·{" "}
                          {t.dueAt
                            ? new Date(t.dueAt).toLocaleDateString("pt-BR")
                            : "sem prazo"}
                        </p>
                      </div>
                      <Badge
                        tone={
                          t.priority === "critical"
                            ? "red"
                            : t.priority === "high"
                              ? "amber"
                              : "slate"
                        }
                      >
                        {t.priority}
                      </Badge>
                    </div>
                  ))}
                </div>
              )}
            </Card>
            <Card>
              <h2 className="text-lg font-bold">Matters prioritários</h2>
              <div className="mt-4 space-y-3">
                {data.priorityMatters.map((m) => (
                  <a
                    key={m.id}
                    href={`/app/matters/${m.id}`}
                    className="block rounded-xl bg-slate-50 p-4 hover:bg-slate-100"
                  >
                    <div className="flex justify-between gap-2">
                      <p className="font-semibold">{m.title}</p>
                      <Badge tone="red">{m.priority}</Badge>
                    </div>
                    <p className="mt-1 text-xs text-slate-500">
                      {m.internalNumber} ·{" "}
                      {m.responsibleName ?? "Sem responsável"}
                    </p>
                  </a>
                ))}
              </div>
            </Card>
          </div>
          <div className="mt-6 grid gap-6 lg:grid-cols-2">
            <Card>
              <h2 className="flex items-center gap-2 text-lg font-bold">
                <FileText size={20} />
                Documentos recentes
              </h2>
              <div className="mt-4 divide-y">
                {data.recentDocuments.map((d) => (
                  <div
                    className="flex items-center justify-between py-3"
                    key={d.id}
                  >
                    <div>
                      <p className="font-medium">{d.title}</p>
                      <p className="text-xs text-slate-500">
                        {d.category} · versão {d.versionNumber}
                      </p>
                    </div>
                    <Badge>
                      {new Date(d.createdAt).toLocaleDateString("pt-BR")}
                    </Badge>
                  </div>
                ))}
              </div>
            </Card>
            <Card>
              <h2 className="flex items-center gap-2 text-lg font-bold">
                <Scale size={20} />
                Atividade jurídica recente
              </h2>
              {data.recentActivity.length === 0 ? (
                <Empty
                  title="Sem atualizações"
                  description="Eventos de Matters aparecerão aqui."
                />
              ) : (
                <div className="mt-4 space-y-4">
                  {data.recentActivity.map((e) => (
                    <div
                      key={e.id}
                      className="border-l-2 border-brand-primary pl-4"
                    >
                      <p className="font-medium">{e.summary}</p>
                      <p className="text-xs text-slate-500">
                        {e.actorName} ·{" "}
                        {new Date(e.createdAt).toLocaleString("pt-BR")}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </Card>
          </div>
        </>
      )}
    </>
  );
}
function Metric({
  icon,
  label,
  value,
  tone,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
  tone: "red" | "amber" | "blue" | "green";
}) {
  const c = {
    red: "bg-red-50 text-red-700",
    amber: "bg-amber-50 text-amber-800",
    blue: "bg-blue-50 text-blue-700",
    green: "bg-emerald-50 text-emerald-700",
  }[tone];
  return (
    <Card>
      <div className={`mb-4 inline-flex rounded-xl p-2.5 ${c}`}>{icon}</div>
      <p className="text-3xl font-bold">{value}</p>
      <p className="mt-1 text-sm text-slate-500">{label}</p>
    </Card>
  );
}
