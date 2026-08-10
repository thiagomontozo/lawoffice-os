import { useEffect, useState } from "react";
import { BellRing, Save } from "lucide-react";
import { Button, Card, Loading, PageHeader } from "../app/ui";
import { get, put } from "../services/api";
import type { NotificationPreferences } from "../types";

export function SettingsPage() {
  const [preferences, setPreferences] = useState<NotificationPreferences>();
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void get<NotificationPreferences>("/api/v1/notifications/preferences")
      .then(setPreferences)
      .catch(() => setError("Não foi possível carregar as preferências."));
  }, []);

  async function savePreferences() {
    if (!preferences) return;
    setError("");
    try {
      const updated = await put<NotificationPreferences>(
        "/api/v1/notifications/preferences",
        preferences,
      );
      setPreferences(updated);
      setSaved(true);
      setTimeout(() => setSaved(false), 2500);
    } catch {
      setError("Não foi possível salvar as preferências.");
    }
  }

  if (!preferences && !error) return <Loading />;

  return (
    <>
      <PageHeader
        title="Configurações"
        description="Preferências gerais, segurança e parâmetros do escritório."
      />
      <div className="grid gap-5 md:grid-cols-2">
        <Card>
          <h2 className="flex items-center gap-2 font-bold">
            <BellRing size={19} /> Notificações por e-mail
          </h2>
          <p className="mt-2 text-sm text-slate-500">
            Os avisos permanecem no sistema. Ative somente os tipos que também
            devem ser enviados ao seu e-mail profissional.
          </p>
          {error ? (
            <p role="alert" className="mt-4 text-sm text-red-700">
              {error}
            </p>
          ) : null}
          {preferences ? (
            <div className="mt-5 space-y-3">
              <PreferenceToggle
                label="Prazos próximos"
                description="Avisos gerados para prazos abertos atribuídos a você."
                checked={preferences.emailDeadlines}
                onChange={(emailDeadlines) =>
                  setPreferences({ ...preferences, emailDeadlines })
                }
              />
              <PreferenceToggle
                label="Tarefas atrasadas"
                description="Avisos de tarefas não concluídas após o vencimento."
                checked={preferences.emailTasks}
                onChange={(emailTasks) =>
                  setPreferences({ ...preferences, emailTasks })
                }
              />
              <Button onClick={() => void savePreferences()}>
                <Save size={16} className="mr-2 inline" />
                {saved ? "Preferências salvas" : "Salvar preferências"}
              </Button>
            </div>
          ) : null}
        </Card>
        <Card>
          <h2 className="font-bold">Escritório e localidade</h2>
          <p className="mt-2 text-sm text-slate-500">
            Fuso horário, locale e informações institucionais usados em datas e
            comunicação.
          </p>
          <button className="mt-5 text-sm font-semibold text-brand-primary">
            Editar informações
          </button>
        </Card>
        <Card>
          <h2 className="font-bold">Sessões e segurança</h2>
          <p className="mt-2 text-sm text-slate-500">
            Logout, alteração de senha e revogação automática quando uma conta é
            desativada.
          </p>
          <button className="mt-5 text-sm font-semibold text-brand-primary">
            Alterar senha
          </button>
        </Card>
        <Card>
          <h2 className="font-bold">Campos customizados</h2>
          <p className="mt-2 text-sm text-slate-500">
            Defina campos de texto, número, data, booleano e seleção para
            Matters e clientes.
          </p>
        </Card>
        <Card>
          <h2 className="font-bold">Templates de Matter</h2>
          <p className="mt-2 text-sm text-slate-500">
            Padronize área, workflow, pastas, tarefas e tags para novos
            trabalhos.
          </p>
        </Card>
      </div>
    </>
  );
}

function PreferenceToggle({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex cursor-pointer items-start justify-between gap-4 rounded-xl border border-slate-200 p-3">
      <span>
        <span className="block text-sm font-semibold">{label}</span>
        <span className="mt-1 block text-xs text-slate-500">{description}</span>
      </span>
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        className="mt-1 h-4 w-4 accent-[var(--brand-primary)]"
      />
    </label>
  );
}
