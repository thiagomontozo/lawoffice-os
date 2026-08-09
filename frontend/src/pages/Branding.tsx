import { useEffect, useState } from "react";
import { Image, MonitorSmartphone, Palette, Save } from "lucide-react";
import { Button, Card, Loading, PageHeader } from "../app/ui";
import { api, put } from "../services/api";
import type { Branding } from "../types";
export function BrandingPage() {
  const [data, setData] = useState<Branding>();
  const [saved, setSaved] = useState(false);
  useEffect(() => {
    void api<Branding>("/api/v1/branding").then(setData);
  }, []);
  if (!data) return <Loading />;
  async function save() {
    const updated = await put<Branding>("/api/v1/branding", data);
    setData(updated);
    setSaved(true);
    setTimeout(() => setSaved(false), 2500);
  }
  return (
    <>
      <PageHeader
        title="Brand Studio"
        description="A marca do escritório, não a do produto, conduz a experiência operacional e o portal."
        action={
          <Button onClick={() => void save()}>
            <Save size={16} className="mr-2 inline" />
            {saved ? "Salvo" : "Salvar identidade"}
          </Button>
        }
      />
      <div className="grid gap-6 xl:grid-cols-[.85fr_1.15fr]">
        <div className="space-y-6">
          <Card>
            <h2 className="flex items-center gap-2 font-bold">
              <Palette />
              Identidade
            </h2>
            <div className="mt-5 grid gap-4">
              <Field
                label="Título do sistema"
                value={data.systemTitle}
                set={(systemTitle) => setData({ ...data, systemTitle })}
              />
              <Field
                label="Nome exibido"
                value={data.firmDisplayName}
                set={(firmDisplayName) => setData({ ...data, firmDisplayName })}
              />
              {(["primaryColor", "secondaryColor", "accentColor"] as const).map(
                (key) => (
                  <label
                    key={key}
                    className="flex items-center justify-between rounded-xl border p-3 text-sm font-medium"
                  >
                    <span>
                      {
                        {
                          primaryColor: "Cor principal",
                          secondaryColor: "Cor secundária",
                          accentColor: "Destaque",
                        }[key]
                      }
                    </span>
                    <div className="flex items-center gap-3">
                      <code>{data[key]}</code>
                      <input
                        type="color"
                        value={data[key]}
                        onChange={(e) =>
                          setData({ ...data, [key]: e.target.value })
                        }
                      />
                    </div>
                  </label>
                ),
              )}
            </div>
          </Card>
          <Card>
            <h2 className="flex items-center gap-2 font-bold">
              <Image />
              Imagens da marca
            </h2>
            <p className="mt-2 text-sm text-slate-500">
              PNG, JPEG ou WEBP. SVG não é aceito na V0.1 para reduzir risco de
              conteúdo ativo.
            </p>
            <div className="mt-4 grid gap-3 sm:grid-cols-2">
              {["logo-light", "logo-dark", "favicon", "login-background"].map(
                (kind) => (
                  <label
                    key={kind}
                    className="cursor-pointer rounded-xl border border-dashed p-4 text-center text-sm"
                  >
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/webp"
                      className="sr-only"
                      onChange={(e) => {
                        const file = e.target.files?.[0];
                        if (!file) return;
                        const body = new FormData();
                        body.append("file", file);
                        void api(`/api/v1/branding/assets/${kind}`, {
                          method: "POST",
                          body,
                        });
                      }}
                    />
                    {kind.replace("-", " ")}
                  </label>
                ),
              )}
            </div>
          </Card>
          <Card>
            <h2 className="font-bold">Portal do cliente</h2>
            <div className="mt-4 space-y-4">
              <Field
                label="Título"
                value={data.clientPortalTitle}
                set={(clientPortalTitle) =>
                  setData({ ...data, clientPortalTitle })
                }
              />
              <label className="text-sm font-medium">
                Mensagem de boas-vindas
                <textarea
                  className="mt-1.5 w-full rounded-xl border p-3"
                  value={data.clientPortalWelcomeText}
                  onChange={(e) =>
                    setData({
                      ...data,
                      clientPortalWelcomeText: e.target.value,
                    })
                  }
                />
              </label>
            </div>
          </Card>
        </div>
        <Card className="sticky top-28 h-fit">
          <h2 className="flex items-center gap-2 font-bold">
            <MonitorSmartphone />
            Preview em tempo real
          </h2>
          <div className="mt-5 overflow-hidden rounded-2xl border shadow-xl">
            <div
              style={{ background: data.secondaryColor }}
              className="flex h-16 items-center px-5 text-white"
            >
              <div>
                <p className="text-xs" style={{ color: data.accentColor }}>
                  {data.firmDisplayName}
                </p>
                <p className="font-bold">{data.systemTitle}</p>
              </div>
            </div>
            <div className="bg-slate-50 p-6">
              <p className="text-xs uppercase text-slate-500">Command Center</p>
              <h3 className="mt-1 text-xl font-bold">Bom dia, equipe</h3>
              <div className="mt-5 grid gap-3 sm:grid-cols-2">
                <div className="rounded-xl bg-white p-4 shadow-sm">
                  <p className="text-sm text-slate-500">Prazos críticos</p>
                  <p className="mt-1 text-2xl font-bold">3</p>
                </div>
                <div className="rounded-xl bg-white p-4 shadow-sm">
                  <p className="text-sm text-slate-500">Audiências</p>
                  <p className="mt-1 text-2xl font-bold">2</p>
                </div>
              </div>
              <button
                style={{ background: data.primaryColor }}
                className="mt-5 rounded-xl px-4 py-2.5 text-white"
              >
                Novo Matter
              </button>
            </div>
          </div>
        </Card>
      </div>
    </>
  );
}
function Field({
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
        value={value}
        onChange={(e) => set(e.target.value)}
        className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
      />
    </label>
  );
}
