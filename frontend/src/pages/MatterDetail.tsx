import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import {
  Archive,
  CalendarClock,
  Download,
  FilePlus2,
  LockKeyhole,
  Plus,
  Scale,
  Upload,
} from "lucide-react";
import { Badge, Button, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api } from "../services/api";
import type { MatterDetail as Detail } from "../types";
const tabs = [
  "Overview",
  "Timeline",
  "Documents",
  "Deadlines",
  "Tasks",
  "Parties",
  "Team",
  "Workflow",
  "Calendar",
  "Notes",
  "Finance",
  "Audit",
] as const;
export function MatterDetailPage() {
  const { id } = useParams();
  const [data, setData] = useState<Detail>();
  const [tab, setTab] = useState<(typeof tabs)[number]>("Overview");
  const [error, setError] = useState("");
  useEffect(() => {
    if (id)
      void api<Detail>(`/api/v1/matters/${id}`)
        .then(setData)
        .catch((e) => setError(e instanceof Error ? e.message : "Erro"));
  }, [id]);
  if (!data && !error) return <Loading />;
  if (error)
    return <Card className="border-red-200 text-red-700">{error}</Card>;
  if (!data) return null;
  const m = data.matter;
  return (
    <>
      <PageHeader
        title={m.title}
        description={`${m.internalNumber}${m.caseNumber ? ` · ${m.caseNumber}` : ""}`}
        action={
          <div className="flex gap-2">
            <Button className="bg-white text-slate-800 ring-1 ring-slate-200">
              <Archive size={16} className="mr-2 inline" />
              Encerrar
            </Button>
            <Button>Editar Matter</Button>
          </div>
        }
      />
      <Card className="mb-5">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
          <Meta label="Status">
            <Badge tone="blue">{m.status}</Badge>
          </Meta>
          <Meta label="Prioridade">
            <Badge tone={m.priority === "critical" ? "red" : "amber"}>
              {m.priority}
            </Badge>
          </Meta>
          <Meta label="Área">{m.legalAreaName ?? "Não informada"}</Meta>
          <Meta label="Responsável">
            {m.responsibleName ?? "Não atribuído"}
          </Meta>
          <Meta label="Segurança">
            <span className="flex items-center gap-1">
              <LockKeyhole size={14} />
              {m.confidentiality}
            </span>
          </Meta>
        </div>
      </Card>
      <div className="mb-5 overflow-x-auto">
        <div className="flex min-w-max gap-1 rounded-xl border bg-white p-1">
          {tabs.map((x) => (
            <button
              onClick={() => setTab(x)}
              key={x}
              className={`rounded-lg px-3 py-2 text-sm ${tab === x ? "bg-brand-primary text-white" : "text-slate-600 hover:bg-slate-100"}`}
            >
              {x}
            </button>
          ))}
        </div>
      </div>
      {tab === "Overview" && <Overview data={data} />}{" "}
      {tab === "Timeline" && <Timeline data={data} />}{" "}
      {tab === "Documents" && <Documents data={data} />}{" "}
      {tab === "Deadlines" && (
        <ListSection
          title="Prazos"
          items={data.deadlines.map((x) => ({
            id: x.id,
            title: x.title,
            subtitle: `${new Date(x.dueAt).toLocaleString("pt-BR")} · ${x.priority}`,
          }))}
        />
      )}{" "}
      {tab === "Tasks" && (
        <ListSection
          title="Tarefas"
          items={data.tasks.map((x) => ({
            id: x.id,
            title: x.title,
            subtitle: `${x.status} · ${x.assigneeName ?? "sem responsável"}`,
          }))}
        />
      )}{" "}
      {tab === "Parties" && (
        <ListSection
          title="Partes"
          items={data.parties.map((x) => ({
            id: x.id,
            title: x.name,
            subtitle: `${x.side} · ${x.role}`,
          }))}
        />
      )}{" "}
      {tab === "Notes" && (
        <ListSection
          title="Notas"
          items={data.notes.map((x) => ({
            id: x.id,
            title: x.content,
            subtitle: `${x.visibility} · ${x.authorName}`,
          }))}
        />
      )}{" "}
      {tab === "Finance" && <Finance data={data} />}{" "}
      {["Team", "Workflow", "Calendar", "Audit"].includes(tab) && (
        <Placeholder title={tab} />
      )}
    </>
  );
}
function Overview({ data }: { data: Detail }) {
  return (
    <div className="grid gap-6 lg:grid-cols-[1.3fr_.7fr]">
      <Card>
        <h2 className="text-lg font-bold">Visão geral</h2>
        <p className="mt-4 leading-7 text-slate-600">
          {data.matter.description ??
            "Adicione uma descrição estratégica, contexto do cliente e objetivos deste Matter."}
        </p>
        <h3 className="mt-8 font-bold">Atividade recente</h3>
        <div className="mt-4 space-y-5">
          {data.timeline.slice(0, 5).map((e) => (
            <div
              className="relative border-l-2 border-slate-200 pl-5"
              key={e.id}
            >
              <span className="absolute -left-[5px] top-1 h-2 w-2 rounded-full bg-brand-primary" />
              <p className="font-medium">{e.summary}</p>
              <p className="text-xs text-slate-500">
                {e.actorName} · {new Date(e.createdAt).toLocaleString("pt-BR")}
              </p>
            </div>
          ))}
        </div>
      </Card>
      <div className="space-y-6">
        <Card>
          <h2 className="font-bold">Próximas ações</h2>
          <div className="mt-4 space-y-3">
            {data.deadlines.slice(0, 3).map((d) => (
              <div key={d.id} className="rounded-xl bg-red-50 p-3">
                <p className="font-semibold text-red-900">{d.title}</p>
                <p className="text-xs text-red-700">
                  {new Date(d.dueAt).toLocaleString("pt-BR")}
                </p>
              </div>
            ))}
          </div>
        </Card>
        <Card>
          <h2 className="font-bold">Ações rápidas</h2>
          <div className="mt-4 grid gap-2">
            <Quick icon={<Upload />} label="Adicionar documento" />
            <Quick icon={<CalendarClock />} label="Criar prazo" />
            <Quick icon={<Plus />} label="Criar tarefa" />
            <Quick icon={<Scale />} label="Adicionar evento" />
          </div>
        </Card>
      </div>
    </div>
  );
}
function Timeline({ data }: { data: Detail }) {
  return (
    <Card>
      <h2 className="text-lg font-bold">Legal Timeline</h2>
      <p className="mt-1 text-sm text-slate-500">
        Registro cronológico de decisões, documentos e mudanças do Matter.
      </p>
      <div className="mt-7 space-y-7">
        {data.timeline.map((e) => (
          <div className="grid gap-3 md:grid-cols-[150px_1fr]" key={e.id}>
            <time className="text-sm text-slate-500">
              {new Date(e.createdAt).toLocaleString("pt-BR")}
            </time>
            <div className="relative border-l-2 border-brand-primary pl-6">
              <span className="absolute -left-[7px] top-0 h-3 w-3 rounded-full bg-brand-primary ring-4 ring-white" />
              <p className="font-semibold">{e.summary}</p>
              <p className="mt-1 text-sm text-slate-500">
                {e.actorName} · {e.type}
              </p>
              {e.clientVisible && (
                <Badge tone="green">visível ao cliente</Badge>
              )}
            </div>
          </div>
        ))}
      </div>
    </Card>
  );
}
function Documents({ data }: { data: Detail }) {
  return (
    <Card>
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-bold">Documentos e versões</h2>
          <p className="text-sm text-slate-500">
            Cada atualização preserva o arquivo anterior e seu checksum.
          </p>
        </div>
        <Button>
          <FilePlus2 size={16} className="mr-2 inline" />
          Enviar
        </Button>
      </div>
      {data.documents.length === 0 ? (
        <Empty
          title="Nenhum documento"
          description="Envie a primeira versão de uma petição, contrato ou evidência."
        />
      ) : (
        <div className="mt-5 divide-y">
          {data.documents.map((d) => (
            <div className="flex items-center gap-4 py-4" key={d.id}>
              <div className="rounded-xl bg-blue-50 p-3 text-blue-700">
                <FilePlus2 />
              </div>
              <div className="min-w-0 flex-1">
                <p className="font-semibold">{d.title}</p>
                <p className="text-xs text-slate-500">
                  {d.category} · v{d.versionNumber} ·{" "}
                  {(d.sizeBytes / 1024).toFixed(1)} KB
                </p>
              </div>
              {d.clientVisible && <Badge tone="green">cliente</Badge>}
              <a
                aria-label={`Baixar ${d.title}`}
                href={`/api/v1/documents/${d.id}/download`}
              >
                <Download />
              </a>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
function Finance({ data }: { data: Detail }) {
  const f = data.financial;
  const money = (v: number) =>
    new Intl.NumberFormat("pt-BR", {
      style: "currency",
      currency: "BRL",
    }).format(v / 100);
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <Card>
        <p className="text-sm text-slate-500">Honorários contratados</p>
        <p className="mt-2 text-2xl font-bold">{money(f.feesCents)}</p>
      </Card>
      <Card>
        <p className="text-sm text-slate-500">Recebido</p>
        <p className="mt-2 text-2xl font-bold text-emerald-700">
          {money(f.paymentsCents)}
        </p>
      </Card>
      <Card>
        <p className="text-sm text-slate-500">Pendente</p>
        <p className="mt-2 text-2xl font-bold text-amber-700">
          {money(f.pendingCents)}
        </p>
      </Card>
      <Card className="md:col-span-3">
        <p className="text-sm text-slate-500">
          Este é um resumo operacional por Matter, não contabilidade oficial.
        </p>
      </Card>
    </div>
  );
}
function ListSection({
  title,
  items,
}: {
  title: string;
  items: { id: string; title: string; subtitle: string }[];
}) {
  return (
    <Card>
      <div className="flex justify-between">
        <h2 className="text-lg font-bold">{title}</h2>
        <Button>
          <Plus size={16} className="mr-2 inline" />
          Adicionar
        </Button>
      </div>
      {items.length === 0 ? (
        <Empty
          title={`Sem ${title.toLowerCase()}`}
          description="Use a ação acima para registrar o primeiro item."
        />
      ) : (
        <div className="mt-5 divide-y">
          {items.map((i) => (
            <div key={i.id} className="py-4">
              <p className="font-semibold">{i.title}</p>
              <p className="text-sm text-slate-500">{i.subtitle}</p>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
function Placeholder({ title }: { title: string }) {
  return (
    <Card>
      <h2 className="text-lg font-bold">{title}</h2>
      <p className="mt-2 text-slate-500">
        Este painel usa os dados, controles de permissão e eventos do Matter.
        Utilize as ações de contexto para administrar esta seção.
      </p>
      <div className="mt-6 rounded-xl border border-dashed p-8 text-center text-sm text-slate-500">
        Nenhum registro nesta visualização.
      </div>
    </Card>
  );
}
function Meta({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <p className="mb-1 text-xs uppercase tracking-wide text-slate-500">
        {label}
      </p>
      <div className="font-medium">{children}</div>
    </div>
  );
}
function Quick({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <button className="flex items-center gap-3 rounded-xl border p-3 text-left text-sm font-medium hover:bg-slate-50">
      {icon}
      {label}
    </button>
  );
}
