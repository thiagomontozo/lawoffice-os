import { useEffect, useState, type FormEvent } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { CalendarClock, Download, FileText, Scale } from "lucide-react";
import { Badge, Button, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api, post } from "../services/api";
import type { Branding, Matter, MatterDetail } from "../types";

export function PortalHome() {
  const [data, setData] = useState<{ items: Matter[]; branding: Branding }>();
  useEffect(() => {
    void api<{ items: Matter[]; branding: Branding }>(
      "/api/v1/portal/matters",
    ).then(setData);
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
          {data.items.map((matter) => (
            <Link key={matter.id} to={`/portal/matters/${matter.id}`}>
              <Card className="h-full hover:border-brand-primary">
                <Scale className="text-brand-primary" />
                <h2 className="mt-4 font-bold">{matter.title}</h2>
                <div className="mt-3 flex gap-2">
                  <Badge tone="blue">{matter.status}</Badge>
                  <Badge>{matter.type.replaceAll("_", " ")}</Badge>
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
  const [data, setData] = useState<{
    detail: MatterDetail;
    branding: Branding;
  }>();
  useEffect(() => {
    if (id)
      void api<{ detail: MatterDetail; branding: Branding }>(
        `/api/v1/portal/matters/${id}`,
      ).then(setData);
  }, [id]);
  if (!data) return <Loading />;
  applyBrand(data.branding);
  const detail = data.detail;
  return (
    <PortalShell branding={data.branding}>
      <PageHeader
        title={detail.matter.title}
        description="Informações selecionadas e compartilhadas pelo escritório."
      />
      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <h2 className="flex items-center gap-2 font-bold">
            <Scale />
            Atualizações compartilhadas
          </h2>
          {detail.timeline.length === 0 ? (
            <Empty
              title="Sem atualizações públicas"
              description="Novos eventos compartilhados aparecerão nesta linha do tempo."
            />
          ) : (
            <div className="mt-5 space-y-5">
              {detail.timeline.map((event) => (
                <div
                  key={event.id}
                  className="border-l-2 border-brand-primary pl-4"
                >
                  <p className="font-medium">{event.summary}</p>
                  <p className="text-xs text-slate-500">
                    {new Date(event.createdAt).toLocaleString("pt-BR")}
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
            {detail.documents.length === 0 ? (
              <p className="mt-2 text-sm text-slate-500">
                Nenhum documento foi compartilhado.
              </p>
            ) : (
              <div className="mt-3 space-y-2">
                {detail.documents.map((document) => (
                  <a
                    key={document.id}
                    href={`/api/v1/portal/documents/${document.id}/download`}
                    className="flex items-center justify-between rounded-lg border p-3 text-sm font-medium hover:bg-slate-50"
                  >
                    <span className="truncate">{document.title}</span>
                    <Download size={16} />
                  </a>
                ))}
              </div>
            )}
          </Card>
        </div>
      </div>
    </PortalShell>
  );
}

export function PortalAcceptInvitation() {
  const [params] = useSearchParams();
  const [password, setPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");
  const [error, setError] = useState("");
  const [accepted, setAccepted] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (password !== confirmation) {
      setError("As senhas precisam ser iguais.");
      return;
    }
    try {
      await post("/api/v1/portal/invitations/accept", {
        token: params.get("token") ?? "",
        password,
      });
      setAccepted(true);
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Não foi possível aceitar o convite.",
      );
    }
  }
  return (
    <div className="grid min-h-screen place-items-center bg-slate-950 p-5">
      <Card className="w-full max-w-md p-8">
        <p className="text-xs font-bold uppercase tracking-[.2em] text-brand-primary">
          Portal seguro
        </p>
        <h1 className="mt-3 text-2xl font-bold">Ative seu acesso</h1>
        {accepted ? (
          <div className="mt-6">
            <p className="text-emerald-700">
              Senha definida. Seu acesso está pronto.
            </p>
            <Link
              to="/portal/login"
              className="mt-5 inline-flex rounded-xl bg-brand-primary px-4 py-2.5 font-semibold text-white"
            >
              Ir para o portal
            </Link>
          </div>
        ) : (
          <form
            onSubmit={(event) => void submit(event)}
            className="mt-6 space-y-4"
          >
            <label className="block text-sm font-medium">
              Nova senha
              <input
                required
                minLength={12}
                type="password"
                className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
              />
            </label>
            <label className="block text-sm font-medium">
              Confirme a senha
              <input
                required
                minLength={12}
                type="password"
                className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
                value={confirmation}
                onChange={(event) => setConfirmation(event.target.value)}
              />
            </label>
            {error && (
              <p role="alert" className="text-sm text-red-700">
                {error}
              </p>
            )}
            <Button className="w-full">Ativar acesso</Button>
          </form>
        )}
      </Card>
    </div>
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
          <div className="flex items-center gap-3">
            {branding?.logoLightUrl && (
              <img
                src={branding.logoLightUrl}
                alt=""
                className="h-9 max-w-44 object-contain"
              />
            )}
            <div>
              <p className="text-xs uppercase tracking-widest text-brand-accent">
                {branding?.firmDisplayName ?? "Portal seguro"}
              </p>
              <p className="font-bold">
                {branding?.clientPortalTitle ?? "Portal do Cliente"}
              </p>
            </div>
          </div>
          <span className="text-xs text-slate-500">Área restrita</span>
        </div>
      </header>
      <main className="mx-auto max-w-6xl p-5 md:p-8">{children}</main>
    </div>
  );
}

function applyBrand(branding: Branding) {
  document.documentElement.style.setProperty(
    "--brand-primary",
    branding.primaryColor,
  );
  document.documentElement.style.setProperty(
    "--brand-secondary",
    branding.secondaryColor,
  );
  document.documentElement.style.setProperty(
    "--brand-accent",
    branding.accentColor,
  );
  document.title = branding.clientPortalTitle;
}
