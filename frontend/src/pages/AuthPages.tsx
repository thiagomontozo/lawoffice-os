import { useState, type FormEvent } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import {
  Building2,
  CheckCircle2,
  Palette,
  ShieldCheck,
  Workflow,
} from "lucide-react";
import { Button } from "../app/ui";
import { useAuth } from "../app/AuthContext";
import { api, post } from "../services/api";
export function LoginPage() {
  const { session, refresh } = useAuth();
  const nav = useNavigate();
  const [form, setForm] = useState({ firmSlug: "", email: "", password: "" });
  const [error, setError] = useState("");
  if (session) return <Navigate to="/app" replace />;
  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await post("/api/v1/auth/login", form);
      await refresh();
      nav("/app");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Não foi possível entrar.");
    }
  }
  return (
    <AuthShell>
      <h1 className="text-3xl font-bold">Acesse seu workspace</h1>
      <p className="mt-2 text-slate-500">
        Entre com a identificação do escritório e sua conta.
      </p>
      <form onSubmit={submit} className="mt-8 space-y-4">
        <Field
          label="Escritório"
          value={form.firmSlug}
          onChange={(v) => setForm({ ...form, firmSlug: v })}
          placeholder="montozo-associados"
        />
        <Field
          label="E-mail"
          value={form.email}
          onChange={(v) => setForm({ ...form, email: v })}
          type="email"
        />
        <Field
          label="Senha"
          value={form.password}
          onChange={(v) => setForm({ ...form, password: v })}
          type="password"
        />
        {error && (
          <p
            role="alert"
            className="rounded-xl bg-red-50 p-3 text-sm text-red-700"
          >
            {error}
          </p>
        )}
        <Button className="w-full">Entrar</Button>
      </form>
    </AuthShell>
  );
}
export function SetupPage() {
  const { refresh } = useAuth();
  const nav = useNavigate();
  const [step, setStep] = useState(0);
  const [error, setError] = useState("");
  const [form, setForm] = useState({
    legalName: "",
    displayName: "",
    slug: "",
    email: "",
    adminName: "",
    adminEmail: "",
    password: "",
    timezone: "America/Sao_Paulo",
    locale: "pt-BR",
    primaryColor: "#17324D",
    secondaryColor: "#334E68",
    accentColor: "#C9A227",
  });
  const steps = [
    ["Boas-vindas", Building2],
    ["Escritório", Building2],
    ["Identidade", Palette],
    ["Administrador", ShieldCheck],
    ["Estrutura jurídica", Workflow],
    ["Finalizar", CheckCircle2],
  ] as const;
  async function finish() {
    setError("");
    try {
      await post("/api/v1/setup", form);
      await refresh();
      nav("/app");
    } catch (e) {
      setError(e instanceof Error ? e.message : "Não foi possível concluir.");
    }
  }
  return (
    <div className="min-h-screen bg-slate-950 p-4 md:p-10">
      <div className="mx-auto grid max-w-6xl overflow-hidden rounded-3xl bg-white shadow-2xl lg:grid-cols-[320px_1fr]">
        <aside className="bg-slate-900 p-8 text-white">
          <p className="text-xs uppercase tracking-[.25em] text-amber-400">
            LawOffice OS
          </p>
          <h1 className="mt-3 text-2xl font-bold">
            Configure seu Legal Workspace
          </h1>
          <div className="mt-10 space-y-2">
            {steps.map(([name, Icon], index) => (
              <div
                key={name}
                className={`flex items-center gap-3 rounded-xl p-3 ${index === step ? "bg-white/10" : "text-slate-500"}`}
              >
                <Icon size={18} />
                <span className="text-sm">{name}</span>
              </div>
            ))}
          </div>
        </aside>
        <main className="min-h-[680px] p-6 md:p-12">
          <p className="text-sm font-semibold text-brand-primary">
            Etapa {step + 1} de {steps.length}
          </p>
          <h2 className="mt-2 text-3xl font-bold">{steps[step]?.[0]}</h2>
          <div className="mt-8">
            {step === 0 && <Intro />}
            {step === 1 && (
              <div className="grid gap-4 md:grid-cols-2">
                <Field
                  label="Razão social"
                  value={form.legalName}
                  onChange={(v) => setForm({ ...form, legalName: v })}
                />
                <Field
                  label="Nome exibido"
                  value={form.displayName}
                  onChange={(v) => setForm({ ...form, displayName: v })}
                />
                <Field
                  label="Identificador do escritório"
                  value={form.slug}
                  onChange={(v) => setForm({ ...form, slug: v.toLowerCase() })}
                  placeholder="montozo-associados"
                />
                <Field
                  label="E-mail institucional"
                  value={form.email}
                  onChange={(v) => setForm({ ...form, email: v })}
                  type="email"
                />
              </div>
            )}
            {step === 2 && <BrandPreview form={form} setForm={setForm} />}{" "}
            {step === 3 && (
              <div className="max-w-xl space-y-4">
                <Field
                  label="Nome do administrador"
                  value={form.adminName}
                  onChange={(v) => setForm({ ...form, adminName: v })}
                />
                <Field
                  label="E-mail"
                  value={form.adminEmail}
                  onChange={(v) => setForm({ ...form, adminEmail: v })}
                  type="email"
                />
                <Field
                  label="Senha (mínimo 10 caracteres)"
                  value={form.password}
                  onChange={(v) => setForm({ ...form, password: v })}
                  type="password"
                />
              </div>
            )}
            {step === 4 && (
              <div className="grid gap-4 md:grid-cols-3">
                {[
                  "Áreas jurídicas personalizáveis",
                  "Tipos de Matter flexíveis",
                  "Workflow inicial configurável",
                ].map((x) => (
                  <div className="rounded-2xl border p-5" key={x}>
                    <CheckCircle2 className="text-emerald-600" />
                    <p className="mt-3 font-semibold">{x}</p>
                    <p className="mt-1 text-sm text-slate-500">
                      Um conjunto inicial será criado e poderá ser ajustado
                      depois.
                    </p>
                  </div>
                ))}
              </div>
            )}
            {step === 5 && (
              <div className="rounded-2xl bg-emerald-50 p-7">
                <CheckCircle2 size={38} className="text-emerald-600" />
                <h3 className="mt-4 text-xl font-bold">
                  Tudo pronto para criar seu escritório
                </h3>
                <p className="mt-2 text-slate-600">
                  O sistema criará o escritório, o papel Owner, permissões,
                  áreas jurídicas e workflow inicial.
                </p>
              </div>
            )}
          </div>
          {error && (
            <p className="mt-6 rounded-xl bg-red-50 p-3 text-red-700">
              {error}
            </p>
          )}
          <footer className="mt-10 flex justify-between">
            <Button
              type="button"
              className="bg-slate-200 text-slate-800"
              disabled={step === 0}
              onClick={() => setStep(step - 1)}
            >
              Voltar
            </Button>
            {step < steps.length - 1 ? (
              <Button type="button" onClick={() => setStep(step + 1)}>
                Continuar
              </Button>
            ) : (
              <Button type="button" onClick={() => void finish()}>
                Criar escritório
              </Button>
            )}
          </footer>
        </main>
      </div>
    </div>
  );
}
export function PortalLogin() {
  const nav = useNavigate();
  const [params] = useSearchParams();
  const [form, setForm] = useState({
    firmSlug: params.get("firm") ?? "",
    email: "",
    password: "",
  });
  const [error, setError] = useState("");
  async function submit(e: FormEvent) {
    e.preventDefault();
    try {
      await post("/api/v1/portal/login", form);
      nav("/portal");
    } catch (x) {
      setError(x instanceof Error ? x.message : "Falha no acesso");
    }
  }
  return (
    <AuthShell portal>
      <h1 className="text-3xl font-bold">Portal do Cliente</h1>
      <p className="mt-2 text-slate-500">
        Acesse apenas os assuntos e documentos compartilhados com você.
      </p>
      <form onSubmit={submit} className="mt-8 space-y-4">
        <Field
          label="Escritório"
          value={form.firmSlug}
          onChange={(v) => setForm({ ...form, firmSlug: v })}
        />
        <Field
          label="E-mail"
          type="email"
          value={form.email}
          onChange={(v) => setForm({ ...form, email: v })}
        />
        <Field
          label="Senha"
          type="password"
          value={form.password}
          onChange={(v) => setForm({ ...form, password: v })}
        />
        {error && <p className="text-red-700">{error}</p>}
        <Button className="w-full">Acessar portal</Button>
      </form>
    </AuthShell>
  );
}
function AuthShell({
  children,
  portal = false,
}: {
  children: React.ReactNode;
  portal?: boolean;
}) {
  return (
    <div className="grid min-h-screen bg-slate-950 lg:grid-cols-2">
      <div className="hidden p-14 text-white lg:flex lg:flex-col lg:justify-between">
        <p className="text-sm font-bold uppercase tracking-[.25em] text-amber-400">
          {portal ? "Área segura" : "LawOffice OS"}
        </p>
        <div>
          <h2 className="max-w-lg text-5xl font-bold leading-tight">
            Um workspace jurídico moldado para o seu escritório.
          </h2>
          <p className="mt-5 max-w-lg text-lg text-slate-400">
            Matters, prazos, documentos versionados, workflows e segurança em um
            só lugar.
          </p>
        </div>
        <p className="text-sm text-slate-500">
          A customizable legal operations workspace for modern law firms.
        </p>
      </div>
      <main className="grid place-items-center bg-white p-6 lg:rounded-l-[3rem]">
        <div className="w-full max-w-md">{children}</div>
      </main>
    </div>
  );
}
function Field({
  label,
  value,
  onChange,
  type = "text",
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
}) {
  return (
    <label className="block text-sm font-medium text-slate-700">
      {label}
      <input
        required
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        className="mt-1.5 w-full rounded-xl border border-slate-300 px-4 py-3"
      />
    </label>
  );
}
function Intro() {
  return (
    <div className="max-w-2xl">
      <p className="text-lg text-slate-600">
        Este assistente cria a fundação do escritório sem esconder decisões
        importantes. Você configurará identidade, administrador, áreas e fluxo
        inicial.
      </p>
      <div className="mt-6 grid gap-3 md:grid-cols-2">
        {[
          "Isolamento por escritório",
          "Papéis e permissões",
          "Brand Studio white-label",
          "Segurança por Matter",
        ].map((x) => (
          <div
            className="flex items-center gap-3 rounded-xl bg-slate-50 p-4"
            key={x}
          >
            <CheckCircle2 className="text-emerald-600" size={20} />
            {x}
          </div>
        ))}
      </div>
    </div>
  );
}
function BrandPreview({
  form,
  setForm,
}: {
  form: {
    primaryColor: string;
    secondaryColor: string;
    accentColor: string;
    displayName: string;
  };
  setForm: React.Dispatch<
    React.SetStateAction<{
      legalName: string;
      displayName: string;
      slug: string;
      email: string;
      adminName: string;
      adminEmail: string;
      password: string;
      timezone: string;
      locale: string;
      primaryColor: string;
      secondaryColor: string;
      accentColor: string;
    }>
  >;
}) {
  return (
    <div className="grid gap-7 lg:grid-cols-2">
      <div className="space-y-4">
        {(["primaryColor", "secondaryColor", "accentColor"] as const).map(
          (key, index) => (
            <label
              className="flex items-center justify-between rounded-xl border p-4"
              key={key}
            >
              <span>
                {["Cor principal", "Cor secundária", "Destaque"][index]}
              </span>
              <input
                type="color"
                value={form[key]}
                onChange={(e) =>
                  setForm((v) => ({ ...v, [key]: e.target.value }))
                }
              />
            </label>
          ),
        )}
        <p className="text-sm text-slate-500">
          Logos e favicon podem ser enviados no Brand Studio após o primeiro
          acesso.
        </p>
      </div>
      <div
        style={{ background: form.secondaryColor }}
        className="overflow-hidden rounded-2xl p-5 text-white shadow-xl"
      >
        <p
          className="text-xs uppercase tracking-widest"
          style={{ color: form.accentColor }}
        >
          {form.displayName || "Seu escritório"}
        </p>
        <h3 className="mt-2 text-xl font-bold">Portal Jurídico</h3>
        <div className="mt-8 rounded-xl bg-white p-4 text-slate-900">
          <p className="font-semibold">Command Center</p>
          <button
            style={{ background: form.primaryColor }}
            className="mt-5 rounded-lg px-4 py-2 text-white"
          >
            Nova atividade
          </button>
        </div>
      </div>
    </div>
  );
}
