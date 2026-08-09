import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  Check,
  Copy,
  ExternalLink,
  KeyRound,
  Power,
  UserRoundPlus,
} from "lucide-react";
import { Badge, Button, Card, Empty, Loading, PageHeader } from "../app/ui";
import { api, patch, post } from "../services/api";
import type { Matter } from "../types";

type Client = { id: string; name: string; email?: string };
type PortalUser = {
  id: string;
  clientId: string;
  clientName: string;
  email: string;
  active: boolean;
  lastLoginAt?: string;
  createdAt: string;
  matterCount: number;
};
type Invitation = { id: string; invitationUrl: string; expiresAt: string };

export function PortalManagementPage() {
  const [users, setUsers] = useState<PortalUser[]>([]);
  const [clients, setClients] = useState<Client[]>([]);
  const [matters, setMatters] = useState<Matter[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [invitation, setInvitation] = useState<Invitation>();
  const [copied, setCopied] = useState(false);
  const [form, setForm] = useState({
    clientId: "",
    email: "",
    matterIds: [] as string[],
  });

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [portal, clientData, matterData] = await Promise.all([
        api<{ items: PortalUser[] }>("/api/v1/portal/users"),
        api<{ items: Client[] }>("/api/v1/clients?pageSize=100"),
        api<{ items: Matter[] }>("/api/v1/matters?pageSize=100"),
      ]);
      setUsers(portal.items ?? []);
      setClients(clientData.items ?? []);
      setMatters(matterData.items ?? []);
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Não foi possível carregar o portal.",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => void load(), [load]);

  async function invite(event: FormEvent) {
    event.preventDefault();
    setError("");
    try {
      const result = await post<Invitation>("/api/v1/portal/invitations", form);
      setInvitation(result);
      setForm({ clientId: "", email: "", matterIds: [] });
      await load();
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Não foi possível criar o convite.",
      );
    }
  }

  async function setActive(portalUser: PortalUser) {
    await patch(`/api/v1/portal/users/${portalUser.id}/active`, {
      active: !portalUser.active,
    });
    await load();
  }

  async function copyInvitation() {
    if (!invitation) return;
    await navigator.clipboard.writeText(invitation.invitationUrl);
    setCopied(true);
  }

  return (
    <>
      <PageHeader
        title="Portal do Cliente"
        description="Convide clientes e controle exatamente quais assuntos podem ser consultados."
      />
      {error && (
        <div
          role="alert"
          className="mb-5 rounded-xl bg-red-50 px-4 py-3 text-sm text-red-800"
        >
          {error}
        </div>
      )}
      {invitation && (
        <Card className="mb-6 border-emerald-200 bg-emerald-50">
          <div className="flex items-start gap-3">
            <KeyRound className="mt-1 text-emerald-700" />
            <div className="min-w-0 flex-1">
              <h2 className="font-bold text-emerald-950">Convite criado</h2>
              <p className="mt-1 text-sm text-emerald-800">
                Compartilhe este link por um canal seguro. Ele é exibido uma
                única vez e expira em 72 horas.
              </p>
              <div className="mt-3 flex gap-2">
                <input
                  readOnly
                  value={invitation.invitationUrl}
                  className="min-w-0 flex-1 rounded-lg border border-emerald-200 bg-white px-3 py-2 text-sm"
                />
                <button
                  type="button"
                  onClick={() => void copyInvitation()}
                  className="rounded-lg bg-emerald-800 px-3 text-white"
                  aria-label="Copiar convite"
                >
                  {copied ? <Check size={18} /> : <Copy size={18} />}
                </button>
              </div>
            </div>
          </div>
        </Card>
      )}
      <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
        <Card className="overflow-x-auto p-0">
          {loading ? (
            <Loading />
          ) : users.length === 0 ? (
            <Empty
              title="Nenhum cliente convidado"
              description="Crie o primeiro acesso usando o formulário ao lado."
            />
          ) : (
            <table className="w-full min-w-[680px] text-left text-sm">
              <thead className="bg-slate-50 text-xs uppercase text-slate-500">
                <tr>
                  <th className="px-5 py-4">Cliente</th>
                  <th>E-mail</th>
                  <th>Matters</th>
                  <th>Último acesso</th>
                  <th className="px-5">Status</th>
                </tr>
              </thead>
              <tbody className="divide-y">
                {users.map((item) => (
                  <tr key={item.id}>
                    <td className="px-5 py-4 font-semibold">
                      {item.clientName}
                    </td>
                    <td>{item.email}</td>
                    <td>{item.matterCount}</td>
                    <td>
                      {item.lastLoginAt
                        ? new Date(item.lastLoginAt).toLocaleString("pt-BR")
                        : "Nunca"}
                    </td>
                    <td className="px-5">
                      <button
                        onClick={() => void setActive(item)}
                        className="flex items-center gap-2 font-semibold"
                      >
                        <Power size={15} />
                        <Badge tone={item.active ? "green" : "slate"}>
                          {item.active ? "Ativo" : "Revogado"}
                        </Badge>
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
        <Card>
          <h2 className="flex items-center gap-2 font-bold">
            <UserRoundPlus size={19} />
            Novo convite
          </h2>
          <form
            className="mt-5 space-y-4"
            onSubmit={(event) => void invite(event)}
          >
            <label className="block text-sm font-medium">
              Cliente
              <select
                required
                className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
                value={form.clientId}
                onChange={(event) => {
                  const client = clients.find(
                    (item) => item.id === event.target.value,
                  );
                  setForm({
                    ...form,
                    clientId: event.target.value,
                    email: client?.email ?? form.email,
                  });
                }}
              >
                <option value="">Selecione</option>
                {clients.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="block text-sm font-medium">
              E-mail
              <input
                required
                type="email"
                className="mt-1.5 w-full rounded-xl border px-3 py-2.5"
                value={form.email}
                onChange={(event) =>
                  setForm({ ...form, email: event.target.value })
                }
              />
            </label>
            <fieldset>
              <legend className="text-sm font-medium">
                Assuntos compartilhados
              </legend>
              <div className="mt-2 max-h-56 space-y-2 overflow-y-auto rounded-xl border p-3">
                {matters.map((matter) => (
                  <label key={matter.id} className="flex gap-2 text-sm">
                    <input
                      type="checkbox"
                      checked={form.matterIds.includes(matter.id)}
                      onChange={(event) =>
                        setForm({
                          ...form,
                          matterIds: event.target.checked
                            ? [...form.matterIds, matter.id]
                            : form.matterIds.filter((id) => id !== matter.id),
                        })
                      }
                    />
                    <span>{matter.title}</span>
                  </label>
                ))}
              </div>
            </fieldset>
            <Button
              className="w-full"
              disabled={!form.clientId || form.matterIds.length === 0}
            >
              Gerar convite seguro
            </Button>
          </form>
          <p className="mt-4 flex gap-2 text-xs text-slate-500">
            <ExternalLink size={14} />O cliente define a própria senha ao abrir
            o convite.
          </p>
        </Card>
      </div>
    </>
  );
}
