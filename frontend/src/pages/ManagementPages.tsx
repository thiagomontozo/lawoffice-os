import { useCallback, useEffect, useState, type FormEvent } from "react";
import { CheckCircle2, Plus, Power } from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Empty,
  Loading,
  Modal,
  PageHeader,
} from "../app/ui";
import { api, patch, post } from "../services/api";
import type { Matter, Task, User } from "../types";

type Client = {
  id: string;
  type: "person" | "company";
  name: string;
  document?: string;
  email?: string;
  phone?: string;
};
type Role = {
  id: string;
  name: string;
  description?: string;
  system: boolean;
  permissions: string[];
};

function useItems<T>(path: string) {
  const [items, setItems] = useState<T[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const reload = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const response = await api<{ items: T[] }>(path);
      setItems(response.items ?? []);
    } catch (reason) {
      setError(
        reason instanceof Error
          ? reason.message
          : "Não foi possível carregar os dados.",
      );
    } finally {
      setLoading(false);
    }
  }, [path]);
  useEffect(() => void reload(), [reload]);
  return { items, loading, error, reload };
}

function Feedback({ message }: { message: string }) {
  if (!message) return null;
  return (
    <div
      role="status"
      className="mb-4 rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-800"
    >
      {message}
    </div>
  );
}

const inputClass =
  "mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2.5 outline-none focus:border-brand-primary";

export function ClientsPage() {
  const { items, loading, error, reload } = useItems<Client>("/api/v1/clients");
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  const [form, setForm] = useState({
    type: "person",
    name: "",
    document: "",
    email: "",
    phone: "",
  });
  async function save(event: FormEvent) {
    event.preventDefault();
    await post("/api/v1/clients", form);
    setOpen(false);
    setForm({ type: "person", name: "", document: "", email: "", phone: "" });
    setMessage("Cliente criado com sucesso.");
    await reload();
  }
  async function archive(id: string) {
    if (!window.confirm("Arquivar este cliente? O histórico será preservado."))
      return;
    await api(`/api/v1/clients/${id}`, { method: "DELETE" });
    setMessage("Cliente arquivado.");
    await reload();
  }
  return (
    <>
      <PageHeader
        title="Clientes"
        description="Pessoas e empresas relacionadas aos trabalhos jurídicos."
        action={
          <Button onClick={() => setOpen(true)}>
            <Plus size={16} className="mr-2 inline" />
            Novo cliente
          </Button>
        }
      />
      <Feedback message={message} />
      {loading ? (
        <Loading />
      ) : error ? (
        <Empty title="Falha ao carregar" description={error} />
      ) : items.length === 0 ? (
        <Empty
          title="Nenhum cliente"
          description="Cadastre a primeira pessoa ou empresa atendida pelo escritório."
        />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table className="w-full min-w-[720px] text-left">
            <thead className="bg-slate-50 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-5 py-4">Nome</th>
                <th>Tipo</th>
                <th>Documento</th>
                <th>E-mail</th>
                <th className="px-5">Ações</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {items.map((client) => (
                <tr key={client.id}>
                  <td className="px-5 py-4 font-medium">{client.name}</td>
                  <td>
                    <Badge tone="blue">
                      {client.type === "person" ? "Pessoa" : "Empresa"}
                    </Badge>
                  </td>
                  <td>{client.document || "—"}</td>
                  <td>{client.email || "—"}</td>
                  <td className="px-5">
                    <button
                      onClick={() => void archive(client.id)}
                      className="text-sm font-semibold text-red-700"
                    >
                      Arquivar
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
      <Modal open={open} title="Novo cliente" onClose={() => setOpen(false)}>
        <form onSubmit={(event) => void save(event)} className="grid gap-4">
          <label className="text-sm font-medium">
            Tipo
            <select
              className={inputClass}
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
            >
              <option value="person">Pessoa</option>
              <option value="company">Empresa</option>
            </select>
          </label>
          <label className="text-sm font-medium">
            Nome
            <input
              required
              maxLength={180}
              className={inputClass}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Documento
            <input
              maxLength={40}
              className={inputClass}
              value={form.document}
              onChange={(e) => setForm({ ...form, document: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            E-mail
            <input
              type="email"
              className={inputClass}
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Telefone
            <input
              maxLength={40}
              className={inputClass}
              value={form.phone}
              onChange={(e) => setForm({ ...form, phone: e.target.value })}
            />
          </label>
          <Button type="submit">Criar cliente</Button>
        </form>
      </Modal>
    </>
  );
}

export function UsersPage() {
  const users = useItems<User>("/api/v1/users");
  const roles = useItems<Role>("/api/v1/roles");
  const [open, setOpen] = useState(false);
  const [message, setMessage] = useState("");
  const [form, setForm] = useState({
    name: "",
    email: "",
    password: "",
    roleId: "",
  });
  async function save(event: FormEvent) {
    event.preventDefault();
    await post("/api/v1/users", {
      name: form.name,
      email: form.email,
      password: form.password,
      roleIds: [form.roleId],
    });
    setOpen(false);
    setMessage("Usuário criado e vinculado ao papel selecionado.");
    await users.reload();
  }
  async function toggle(user: User) {
    await patch(`/api/v1/users/${user.id}/active`, { active: !user.active });
    setMessage(
      user.active
        ? "Usuário desativado e sessões revogadas."
        : "Usuário reativado.",
    );
    await users.reload();
  }
  return (
    <>
      <PageHeader
        title="Usuários"
        description="Contas internas, atividade e papéis do escritório."
        action={
          <Button
            onClick={() => setOpen(true)}
            disabled={roles.items.length === 0}
          >
            <Plus size={16} className="mr-2 inline" />
            Novo usuário
          </Button>
        }
      />
      <Feedback message={message} />
      {users.loading ? (
        <Loading />
      ) : users.items.length === 0 ? (
        <Empty
          title="Nenhum usuário"
          description="Crie a equipe que participará do workspace."
        />
      ) : (
        <Card className="overflow-x-auto p-0">
          <table className="w-full min-w-[700px] text-left">
            <thead className="bg-slate-50 text-xs uppercase text-slate-500">
              <tr>
                <th className="px-5 py-4">Nome</th>
                <th>E-mail</th>
                <th>Papéis</th>
                <th>Status</th>
                <th className="px-5">Ação</th>
              </tr>
            </thead>
            <tbody className="divide-y">
              {users.items.map((item) => (
                <tr key={item.id}>
                  <td className="px-5 py-4 font-medium">{item.name}</td>
                  <td>{item.email}</td>
                  <td>{item.roles.join(", ") || "Sem papel"}</td>
                  <td>
                    <Badge tone={item.active ? "green" : "slate"}>
                      {item.active ? "Ativo" : "Inativo"}
                    </Badge>
                  </td>
                  <td className="px-5">
                    <button
                      onClick={() => void toggle(item)}
                      className="inline-flex items-center gap-2 text-sm font-semibold text-brand-primary"
                    >
                      <Power size={15} />
                      {item.active ? "Desativar" : "Reativar"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
      <Modal open={open} title="Novo usuário" onClose={() => setOpen(false)}>
        <form onSubmit={(event) => void save(event)} className="grid gap-4">
          <label className="text-sm font-medium">
            Nome
            <input
              required
              className={inputClass}
              value={form.name}
              onChange={(e) => setForm({ ...form, name: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            E-mail
            <input
              required
              type="email"
              className={inputClass}
              value={form.email}
              onChange={(e) => setForm({ ...form, email: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Senha temporária
            <input
              required
              type="password"
              minLength={12}
              className={inputClass}
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Papel
            <select
              required
              className={inputClass}
              value={form.roleId}
              onChange={(e) => setForm({ ...form, roleId: e.target.value })}
            >
              <option value="">Selecione</option>
              {roles.items.map((role) => (
                <option value={role.id} key={role.id}>
                  {role.name}
                </option>
              ))}
            </select>
          </label>
          <Button type="submit">Criar usuário</Button>
        </form>
      </Modal>
    </>
  );
}

export function RolesPage() {
  const roles = useItems<Role>("/api/v1/roles");
  const permissions = useItems<string>("/api/v1/permissions");
  const [open, setOpen] = useState(false);
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  async function save(event: FormEvent) {
    event.preventDefault();
    await post("/api/v1/roles", { name, description, permissions: selected });
    setOpen(false);
    setSelected([]);
    await roles.reload();
  }
  return (
    <>
      <PageHeader
        title="Papéis e permissões"
        description="RBAC personalizável, complementado por acesso específico a Matters."
        action={
          <Button onClick={() => setOpen(true)}>
            <Plus size={16} className="mr-2 inline" />
            Novo papel
          </Button>
        }
      />
      {roles.loading ? (
        <Loading />
      ) : (
        <div className="grid gap-4 lg:grid-cols-2">
          {roles.items.map((role) => (
            <Card key={role.id}>
              <div className="flex items-center justify-between">
                <h2 className="font-bold">{role.name}</h2>
                {role.system && <Badge tone="blue">Sistema</Badge>}
              </div>
              <p className="mt-1 text-sm text-slate-500">
                {role.description || "Sem descrição"}
              </p>
              <p className="mt-4 text-xs font-semibold uppercase text-slate-400">
                {role.permissions.length} permissões
              </p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {role.permissions.slice(0, 8).map((permission) => (
                  <Badge key={permission}>{permission}</Badge>
                ))}
              </div>
            </Card>
          ))}
        </div>
      )}
      <Modal open={open} title="Novo papel" onClose={() => setOpen(false)}>
        <form onSubmit={(event) => void save(event)} className="grid gap-4">
          <label className="text-sm font-medium">
            Nome
            <input
              required
              className={inputClass}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label className="text-sm font-medium">
            Descrição
            <textarea
              className={inputClass}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </label>
          <fieldset>
            <legend className="mb-2 text-sm font-medium">Permissões</legend>
            <div className="max-h-64 space-y-2 overflow-y-auto rounded-xl border p-3">
              {permissions.items.map((permission) => (
                <label
                  key={permission}
                  className="flex items-center gap-2 text-sm"
                >
                  <input
                    type="checkbox"
                    checked={selected.includes(permission)}
                    onChange={(e) =>
                      setSelected(
                        e.target.checked
                          ? [...selected, permission]
                          : selected.filter((value) => value !== permission),
                      )
                    }
                  />
                  {permission}
                </label>
              ))}
            </div>
          </fieldset>
          <Button type="submit">Criar papel</Button>
        </form>
      </Modal>
    </>
  );
}

export function TasksPage() {
  const tasks = useItems<Task>("/api/v1/tasks");
  const matters = useItems<Matter>("/api/v1/matters?pageSize=100");
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    title: "",
    matterId: "",
    priority: "normal",
    dueAt: "",
  });
  async function save(event: FormEvent) {
    event.preventDefault();
    await post("/api/v1/tasks", {
      title: form.title,
      matterId: form.matterId || null,
      priority: form.priority,
      dueAt: form.dueAt ? new Date(form.dueAt).toISOString() : null,
    });
    setOpen(false);
    await tasks.reload();
  }
  async function status(id: string, value: string) {
    await patch(`/api/v1/tasks/${id}/status`, { status: value });
    await tasks.reload();
  }
  return (
    <>
      <PageHeader
        title="Tarefas"
        description="Atividades jurídicas e internas com responsável, prazo e prioridade."
        action={
          <Button onClick={() => setOpen(true)}>
            <Plus size={16} className="mr-2 inline" />
            Nova tarefa
          </Button>
        }
      />
      {tasks.loading ? (
        <Loading />
      ) : tasks.items.length === 0 ? (
        <Empty
          title="Nenhuma tarefa"
          description="Crie uma atividade avulsa ou vinculada a um Matter."
        />
      ) : (
        <div className="grid gap-3">
          {tasks.items.map((task) => (
            <Card
              key={task.id}
              className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between"
            >
              <div>
                <div className="flex items-center gap-2">
                  <CheckCircle2
                    size={18}
                    className={
                      task.status === "done"
                        ? "text-emerald-600"
                        : "text-slate-400"
                    }
                  />
                  <h2 className="font-semibold">{task.title}</h2>
                  <Badge
                    tone={
                      task.priority === "critical"
                        ? "red"
                        : task.priority === "high"
                          ? "amber"
                          : "slate"
                    }
                  >
                    {task.priority}
                  </Badge>
                </div>
                <p className="mt-1 text-sm text-slate-500">
                  {task.matterTitle || "Atividade interna"}
                  {task.dueAt
                    ? ` · ${new Date(task.dueAt).toLocaleString("pt-BR")}`
                    : ""}
                </p>
              </div>
              <select
                aria-label={`Status de ${task.title}`}
                value={task.status}
                onChange={(e) => void status(task.id, e.target.value)}
                className="rounded-xl border px-3 py-2 text-sm"
              >
                <option value="todo">A fazer</option>
                <option value="in_progress">Em andamento</option>
                <option value="blocked">Bloqueada</option>
                <option value="done">Concluída</option>
                <option value="cancelled">Cancelada</option>
              </select>
            </Card>
          ))}
        </div>
      )}
      <Modal open={open} title="Nova tarefa" onClose={() => setOpen(false)}>
        <form onSubmit={(event) => void save(event)} className="grid gap-4">
          <label className="text-sm font-medium">
            Título
            <input
              required
              className={inputClass}
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Matter (opcional)
            <select
              className={inputClass}
              value={form.matterId}
              onChange={(e) => setForm({ ...form, matterId: e.target.value })}
            >
              <option value="">Atividade interna</option>
              {matters.items.map((matter) => (
                <option value={matter.id} key={matter.id}>
                  {matter.internalNumber} · {matter.title}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium">
            Prioridade
            <select
              className={inputClass}
              value={form.priority}
              onChange={(e) => setForm({ ...form, priority: e.target.value })}
            >
              <option value="low">Baixa</option>
              <option value="normal">Normal</option>
              <option value="high">Alta</option>
              <option value="critical">Crítica</option>
            </select>
          </label>
          <label className="text-sm font-medium">
            Prazo
            <input
              type="datetime-local"
              className={inputClass}
              value={form.dueAt}
              onChange={(e) => setForm({ ...form, dueAt: e.target.value })}
            />
          </label>
          <Button type="submit">Criar tarefa</Button>
        </form>
      </Modal>
    </>
  );
}
