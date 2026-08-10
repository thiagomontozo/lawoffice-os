import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  Download,
  FileClock,
  FileSearch,
  FilePlus2,
  RefreshCw,
  RotateCcw,
  Trash2,
} from "lucide-react";
import {
  Badge,
  Button,
  Card,
  Empty,
  Loading,
  Modal,
  PageHeader,
} from "../app/ui";
import { api } from "../services/api";
import type { DocumentExtraction, DocumentItem, Matter } from "../types";

type Version = {
  id: string;
  versionNumber: number;
  originalFileName: string;
  mimeType: string;
  sizeBytes: number;
  checksum: string;
  createdByName: string;
  createdAt: string;
};

const inputClass =
  "mt-1.5 w-full rounded-xl border border-slate-300 px-3 py-2.5 outline-none focus:border-brand-primary";

export function DocumentsPage() {
  const [items, setItems] = useState<DocumentItem[]>([]);
  const [matters, setMatters] = useState<Matter[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploadOpen, setUploadOpen] = useState(false);
  const [versionsOpen, setVersionsOpen] = useState(false);
  const [retentionOpen, setRetentionOpen] = useState(false);
  const [extractionOpen, setExtractionOpen] = useState(false);
  const [extraction, setExtraction] = useState<DocumentExtraction>();
  const [deleted, setDeleted] = useState<DocumentItem[]>([]);
  const [selected, setSelected] = useState<DocumentItem>();
  const [versions, setVersions] = useState<Version[]>([]);
  const [message, setMessage] = useState("");
  const [form, setForm] = useState({
    title: "",
    category: "other",
    matterId: "",
    clientVisible: false,
    file: undefined as File | undefined,
  });
  const load = useCallback(async () => {
    setLoading(true);
    const [documents, matterList] = await Promise.all([
      api<{ items: DocumentItem[] }>("/api/v1/documents?pageSize=100"),
      api<{ items: Matter[] }>("/api/v1/matters?pageSize=100"),
    ]);
    setItems(documents.items ?? []);
    setMatters(matterList.items ?? []);
    setLoading(false);
  }, []);
  useEffect(() => void load(), [load]);

  async function upload(event: FormEvent) {
    event.preventDefault();
    if (!form.file) return;
    const body = new FormData();
    body.set("title", form.title);
    body.set("category", form.category);
    body.set("matterId", form.matterId);
    body.set("clientVisible", String(form.clientVisible));
    body.set("file", form.file);
    await api("/api/v1/documents", { method: "POST", body });
    setUploadOpen(false);
    setMessage("Documento armazenado com checksum e primeira versão imutável.");
    setForm({
      title: "",
      category: "other",
      matterId: "",
      clientVisible: false,
      file: undefined,
    });
    await load();
  }
  async function showVersions(document: DocumentItem) {
    const result = await api<{ items: Version[] }>(
      `/api/v1/documents/${document.id}/versions`,
    );
    setSelected(document);
    setVersions(result.items ?? []);
    setVersionsOpen(true);
  }
  async function showExtraction(document: DocumentItem) {
    setSelected(document);
    setExtractionOpen(true);
    setExtraction(undefined);
    const result = await api<DocumentExtraction>(
      `/api/v1/documents/${document.id}/extraction?pageSize=100`,
    );
    setExtraction(result);
  }
  async function reprocessExtraction() {
    if (!selected) return;
    await api(`/api/v1/documents/${selected.id}/extraction/reprocess`, {
      method: "POST",
    });
    setExtraction((current) =>
      current
        ? { ...current, status: "pending", errorCode: undefined }
        : current,
    );
    setMessage("Documento reenfileirado para extração de texto/OCR.");
  }
  async function remove(document: DocumentItem) {
    if (
      !window.confirm(
        `Mover “${document.title}” para retenção? O arquivo e suas versões serão preservados.`,
      )
    )
      return;
    await api(`/api/v1/documents/${document.id}`, { method: "DELETE" });
    setMessage(
      "Documento removido da biblioteca ativa; metadados e versões foram preservados.",
    );
    await load();
  }

  async function showRetention() {
    const result = await api<{ items: DocumentItem[] }>(
      "/api/v1/documents/deleted",
    );
    setDeleted(result.items ?? []);
    setRetentionOpen(true);
  }

  async function restore(document: DocumentItem) {
    await api(`/api/v1/documents/${document.id}/restore`, { method: "POST" });
    setDeleted(deleted.filter((item) => item.id !== document.id));
    setMessage("Documento recuperado para a biblioteca ativa.");
    await load();
  }

  return (
    <>
      <PageHeader
        title="Document Center"
        description="Biblioteca segura com versionamento, checksums e controle de visibilidade."
        action={
          <div className="flex gap-2">
            <button
              onClick={() => void showRetention()}
              className="rounded-xl border bg-white px-4 py-2.5 text-sm font-semibold text-slate-700"
            >
              Retenção
            </button>
            <Button onClick={() => setUploadOpen(true)}>
              <FilePlus2 size={16} className="mr-2 inline" />
              Enviar documento
            </Button>
          </div>
        }
      />
      {message && (
        <div
          role="status"
          className="mb-4 rounded-xl bg-emerald-50 px-4 py-3 text-sm text-emerald-800"
        >
          {message}
        </div>
      )}
      {loading ? (
        <Loading />
      ) : items.length === 0 ? (
        <Empty
          title="Nenhum documento"
          description="Envie o primeiro arquivo e associe-o a um Matter quando necessário."
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {items.map((document) => (
            <Card key={document.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <Badge tone="blue">{document.category}</Badge>
                  <h2 className="mt-3 font-bold">{document.title}</h2>
                  <p className="mt-1 truncate text-sm text-slate-500">
                    {document.originalFileName}
                  </p>
                </div>
                <span className="rounded-lg bg-slate-100 px-2 py-1 text-xs font-semibold">
                  v{document.versionNumber}
                </span>
              </div>
              <dl className="mt-4 grid grid-cols-2 gap-3 text-xs">
                <div>
                  <dt className="text-slate-400">Tamanho</dt>
                  <dd className="font-medium">
                    {formatBytes(document.sizeBytes)}
                  </dd>
                </div>
                <div>
                  <dt className="text-slate-400">Portal</dt>
                  <dd className="font-medium">
                    {document.clientVisible ? "Visível" : "Interno"}
                  </dd>
                </div>
              </dl>
              <div className="mt-5 flex flex-wrap gap-3 border-t pt-4">
                <a
                  className="inline-flex items-center gap-1 text-sm font-semibold text-brand-primary"
                  href={`/api/v1/documents/${document.id}/download`}
                >
                  <Download size={15} />
                  Baixar
                </a>
                <button
                  onClick={() => void showVersions(document)}
                  className="inline-flex items-center gap-1 text-sm font-semibold text-slate-700"
                >
                  <FileClock size={15} />
                  Versões
                </button>
                <button
                  onClick={() => void showExtraction(document)}
                  className="inline-flex items-center gap-1 text-sm font-semibold text-slate-700"
                >
                  <FileSearch size={15} />
                  Texto/OCR
                </button>
                <button
                  onClick={() => void remove(document)}
                  className="ml-auto inline-flex items-center gap-1 text-sm font-semibold text-red-700"
                >
                  <Trash2 size={15} />
                  Remover
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}
      <Modal
        open={uploadOpen}
        title="Enviar documento"
        onClose={() => setUploadOpen(false)}
      >
        <form onSubmit={(event) => void upload(event)} className="grid gap-4">
          <label className="text-sm font-medium">
            Título
            <input
              required
              maxLength={220}
              className={inputClass}
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
            />
          </label>
          <label className="text-sm font-medium">
            Categoria
            <select
              className={inputClass}
              value={form.category}
              onChange={(e) => setForm({ ...form, category: e.target.value })}
            >
              {[
                "petition",
                "contract",
                "evidence",
                "power_of_attorney",
                "decision",
                "judgment",
                "correspondence",
                "invoice",
                "receipt",
                "report",
                "internal",
                "other",
              ].map((category) => (
                <option key={category} value={category}>
                  {category}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium">
            Matter (opcional)
            <select
              className={inputClass}
              value={form.matterId}
              onChange={(e) => setForm({ ...form, matterId: e.target.value })}
            >
              <option value="">Documento geral</option>
              {matters.map((matter) => (
                <option key={matter.id} value={matter.id}>
                  {matter.internalNumber} · {matter.title}
                </option>
              ))}
            </select>
          </label>
          <label className="text-sm font-medium">
            Arquivo
            <input
              required
              type="file"
              accept=".pdf,.docx,.xlsx,.png,.jpg,.jpeg,.webp,.txt"
              className={inputClass}
              onChange={(e) => setForm({ ...form, file: e.target.files?.[0] })}
            />
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={form.clientVisible}
              onChange={(e) =>
                setForm({ ...form, clientVisible: e.target.checked })
              }
            />
            Disponibilizar no portal do cliente
          </label>
          <p className="text-xs text-slate-500">
            O backend valida tamanho, conteúdo real e extensão. O nome original
            nunca é usado como caminho de armazenamento.
          </p>
          <Button type="submit">Armazenar documento</Button>
        </form>
      </Modal>
      <Modal
        open={versionsOpen}
        title={`Versões · ${selected?.title ?? "Documento"}`}
        onClose={() => setVersionsOpen(false)}
      >
        <div className="space-y-3">
          {versions.map((version) => (
            <div key={version.id} className="rounded-xl border p-4">
              <div className="flex items-center justify-between">
                <strong>Versão {version.versionNumber}</strong>
                <a
                  className="text-sm font-semibold text-brand-primary"
                  href={`/api/v1/documents/${selected?.id}/download?version=${version.versionNumber}`}
                >
                  Baixar
                </a>
              </div>
              <p className="mt-1 text-sm text-slate-600">
                {version.originalFileName}
              </p>
              <p className="mt-2 text-xs text-slate-400">
                {version.createdByName} ·{" "}
                {new Date(version.createdAt).toLocaleString("pt-BR")} ·{" "}
                {formatBytes(version.sizeBytes)}
              </p>
              <code className="mt-2 block truncate rounded bg-slate-50 p-2 text-[10px] text-slate-500">
                SHA-256 {version.checksum}
              </code>
            </div>
          ))}
        </div>
      </Modal>
      <Modal
        open={extractionOpen}
        title={`Texto/OCR · ${selected?.title ?? "Documento"}`}
        onClose={() => setExtractionOpen(false)}
      >
        {!extraction ? (
          <Loading />
        ) : (
          <div className="space-y-4">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone={extractionTone(extraction.status)}>
                {extractionLabel(extraction.status)}
              </Badge>
              {extraction.provider ? (
                <span className="text-xs text-slate-500">
                  {extraction.provider} · {extraction.pageCount} página(s)
                </span>
              ) : null}
            </div>
            {extraction.errorCode ? (
              <p
                role="alert"
                className="rounded-xl bg-amber-50 p-3 text-sm text-amber-900"
              >
                Código: {extraction.errorCode}. O documento original permanece
                disponível.
              </p>
            ) : null}
            {extraction.pages.map((page) => (
              <article key={page.pageNumber} className="rounded-xl border p-4">
                <h3 className="text-xs font-bold uppercase tracking-wide text-slate-400">
                  Página {page.pageNumber}
                </h3>
                <pre className="mt-3 max-h-72 overflow-auto whitespace-pre-wrap font-sans text-sm text-slate-700">
                  {page.content}
                </pre>
              </article>
            ))}
            {extraction.status !== "processing" ? (
              <Button onClick={() => void reprocessExtraction()}>
                <RefreshCw size={15} className="mr-2 inline" />
                Reprocessar versão atual
              </Button>
            ) : null}
          </div>
        )}
      </Modal>
      <Modal
        open={retentionOpen}
        title="Documentos em retenção"
        onClose={() => setRetentionOpen(false)}
      >
        {deleted.length === 0 ? (
          <Empty
            title="Nenhum documento retido"
            description="Documentos removidos aparecem aqui e mantêm todas as versões."
          />
        ) : (
          <div className="space-y-3">
            {deleted.map((document) => (
              <div
                key={document.id}
                className="flex items-center justify-between gap-4 rounded-xl border p-4"
              >
                <div>
                  <strong>{document.title}</strong>
                  <p className="mt-1 text-xs text-slate-500">
                    Removido em{" "}
                    {document.deletedAt
                      ? new Date(document.deletedAt).toLocaleString("pt-BR")
                      : "data indisponível"}
                  </p>
                </div>
                <button
                  onClick={() => void restore(document)}
                  className="inline-flex items-center gap-2 text-sm font-semibold text-brand-primary"
                >
                  <RotateCcw size={15} />
                  Restaurar
                </button>
              </div>
            ))}
          </div>
        )}
      </Modal>
    </>
  );
}

function extractionTone(
  status: DocumentExtraction["status"],
): "slate" | "red" | "amber" | "green" | "blue" {
  if (status === "succeeded") return "green";
  if (status === "failed") return "red";
  if (status === "pending" || status === "processing") return "blue";
  return "amber";
}

function extractionLabel(status: DocumentExtraction["status"]) {
  return {
    pending: "Na fila",
    processing: "Processando",
    succeeded: "Texto disponível",
    failed: "Falhou",
    unsupported: "Formato não suportado",
  }[status];
}

function formatBytes(value: number) {
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / 1024 / 1024).toFixed(1)} MB`;
}
