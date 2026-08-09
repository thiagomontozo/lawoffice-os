import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { CalendarClock, FileText, Scale } from "lucide-react";
import { Badge, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api } from "../services/api";
import type { Branding, Matter, MatterDetail } from "../types";
export function PortalHome() {
  const [data, setData] = useState<{ items: Matter[]; branding: Branding }>();
  useEffect(() => {
    void api<typeof data>("/api/v1/portal/matters").then((x) => setData(x));
  }, []);
  if (!data) return <Loading />;
  applyBrand(data.branding);
  return (
    <PortalShell branding={data.branding}>
      <PageHeader
        title="Seus assuntos"
        description={data.branding.clientPortalWelcomeText}
      />
      {data.items.length === 0 ? (
        <Empty
          title="Nenhum assunto compartilhado"
          description="O escritório ainda não liberou informações para esta conta."
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {data.items.map((m) => (
            <Link key={m.id} to={`/portal/matters/${m.id}`}>
              <Card className="h-full hover:border-brand-primary">
                <Scale className="text-brand-primary" />
                <h2 className="mt-4 font-bold">{m.title}</h2>
                <div className="mt-3 flex gap-2">
                  <Badge tone="blue">{m.status}</Badge>
                  <Badge>{m.type.replaceAll("_", " ")}</Badge>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </PortalShell>
  );
}
export function PortalMatter() {
  const { id } = useParams();
  const [data, setData] = useState<MatterDetail>();
  useEffect(() => {
    if (id)
      void api<MatterDetail>(`/api/v1/portal/matters/${id}`).then(setData);
  }, [id]);
  if (!data) return <Loading />;
  return (
    <PortalShell>
      <PageHeader
        title={data.matter.title}
        description="Informações selecionadas e compartilhadas pelo escritório."
      />
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <h2 className="flex items-center gap-2 font-bold">
            <Scale />
            Atualizações compartilhadas
          </h2>
          {data.timeline.length === 0 ? (
            <Empty
              title="Sem atualizações públicas"
              description="Novos eventos compartilhados aparecerão nesta linha do tempo."
            />
          ) : (
            <div className="mt-5 space-y-5">
              {data.timeline.map((e) => (
                <div
                  key={e.id}
                  className="border-l-2 border-brand-primary pl-4"
                >
                  <p className="font-medium">{e.summary}</p>
                  <p className="text-xs text-slate-500">
                    {new Date(e.createdAt).toLocaleString("pt-BR")}
                  </p>
                </div>
              ))}
            </div>
          )}
        </Card>
        <div className="space-y-5">
          <Card>
            <CalendarClock className="text-brand-primary" />
            <h2 className="mt-3 font-bold">Compromissos</h2>
            <p className="mt-2 text-sm text-slate-500">
              Compromissos liberados pelo escritório aparecem aqui.
            </p>
          </Card>
          <Card>
            <FileText className="text-brand-primary" />
            <h2 className="mt-3 font-bold">Documentos</h2>
            <p className="mt-2 text-sm text-slate-500">
              Somente arquivos marcados como visíveis ao cliente ficam
              disponíveis.
            </p>
          </Card>
        </div>
      </div>
    </PortalShell>
  );
}
function PortalShell({
  children,
  branding,
}: {
  children: React.ReactNode;
  branding?: Branding;
}) {
  return (
    <div className="min-h-screen bg-slate-50">
      <header className="border-b bg-white">
        <div className="mx-auto flex h-20 max-w-6xl items-center justify-between px-5">
          <div>
            <p className="text-xs uppercase tracking-widest text-brand-accent">
              {branding?.firmDisplayName ?? "Portal seguro"}
            </p>
            <p className="font-bold">
              {branding?.clientPortalTitle ?? "Portal do Cliente"}
            </p>
          </div>
          <span className="text-xs text-slate-500">Área restrita</span>
        </div>
      </header>
      <main className="mx-auto max-w-6xl p-5 md:p-8">{children}</main>
    </div>
  );
}
function applyBrand(b: Branding) {
  document.documentElement.style.setProperty("--brand-primary", b.primaryColor);
  document.documentElement.style.setProperty(
    "--brand-secondary",
    b.secondaryColor,
  );
  document.documentElement.style.setProperty("--brand-accent", b.accentColor);
  document.title = b.clientPortalTitle;
}
