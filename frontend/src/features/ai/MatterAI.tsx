import { useState, type FormEvent } from "react";
import {
  MessageSquareText,
  Sparkles,
  ThumbsDown,
  ThumbsUp,
} from "lucide-react";
import { Badge, Button, Card } from "../../app/ui";
import { api } from "../../services/api";
import type { AIAnswer, MatterDetail } from "../../types";

const suggestedQuestions = [
  "Resuma os principais fatos, pedidos e obrigações dos documentos.",
  "Monte uma cronologia preliminar dos eventos e datas citados.",
  "Liste prazos, vencimentos e datas que precisam de revisão humana.",
  "Compare as cláusulas que tratam de pagamento, rescisão e penalidades.",
];

export function MatterAI({
  matterId,
  data,
}: {
  matterId: string;
  data: MatterDetail;
}) {
  const [question, setQuestion] = useState("");
  const [selectedDocuments, setSelectedDocuments] = useState<string[]>([]);
  const [answer, setAnswer] = useState<AIAnswer>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [feedback, setFeedback] = useState<"helpful" | "not_helpful">();

  const toggleDocument = (documentId: string) =>
    setSelectedDocuments((current) =>
      current.includes(documentId)
        ? current.filter((id) => id !== documentId)
        : [...current, documentId],
    );

  const ask = async (event: FormEvent) => {
    event.preventDefault();
    if (!question.trim()) return;
    setLoading(true);
    setError("");
    setAnswer(undefined);
    setFeedback(undefined);
    try {
      const result = await api<AIAnswer>(
        "/api/v1/matters/" + matterId + "/ai/query",
        {
          method: "POST",
          body: JSON.stringify({
            question: question.trim(),
            documentIds: selectedDocuments,
          }),
        },
      );
      setAnswer(result);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Não foi possível consultar o Matter AI Workspace.",
      );
    } finally {
      setLoading(false);
    }
  };

  const rate = async (rating: "helpful" | "not_helpful") => {
    if (!answer) return;
    try {
      await api<void>("/api/v1/matters/" + matterId + "/ai/feedback", {
        method: "POST",
        body: JSON.stringify({ responseId: answer.id, rating }),
      });
      setFeedback(rating);
    } catch (requestError) {
      setError(
        requestError instanceof Error
          ? requestError.message
          : "Não foi possível registrar a avaliação.",
      );
    }
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
      <Card>
        <div className="flex items-start gap-3">
          <div className="rounded-xl bg-violet-100 p-3 text-violet-700">
            <Sparkles size={22} />
          </div>
          <div>
            <h2 className="text-lg font-bold">Matter AI Workspace</h2>
            <p className="mt-1 text-sm text-slate-500">
              Faça perguntas sobre documentos extraídos deste Matter. Toda
              resposta deve apontar para as páginas usadas como fonte.
            </p>
          </div>
        </div>

        <div
          className="mt-6 flex flex-wrap gap-2"
          aria-label="Perguntas sugeridas"
        >
          {suggestedQuestions.map((suggestion) => (
            <button
              key={suggestion}
              type="button"
              onClick={() => setQuestion(suggestion)}
              className="rounded-full border border-violet-200 bg-violet-50 px-3 py-2 text-left text-xs font-semibold text-violet-800 hover:bg-violet-100"
            >
              {suggestion}
            </button>
          ))}
        </div>

        <form className="mt-6" onSubmit={ask}>
          <label htmlFor="matter-ai-question" className="text-sm font-semibold">
            Pergunta
          </label>
          <textarea
            id="matter-ai-question"
            value={question}
            onChange={(event) => setQuestion(event.target.value)}
            maxLength={2000}
            rows={4}
            placeholder="Ex.: Quais obrigações e datas aparecem no contrato?"
            className="mt-2 w-full rounded-xl border border-slate-300 p-3 outline-none focus:border-brand-primary focus:ring-2 focus:ring-brand-primary/20"
          />
          <div className="mt-3 flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-slate-500">
              {question.length}/2000 · a pergunta não é gravada no audit log
            </p>
            <Button disabled={loading || question.trim().length < 3}>
              {loading ? "Analisando fontes…" : "Perguntar com fontes"}
            </Button>
          </div>
        </form>

        {error && (
          <div
            role="alert"
            className="mt-5 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-800"
          >
            {error}
          </div>
        )}

        {answer && (
          <article className="mt-7 border-t border-slate-200 pt-6">
            <div className="flex flex-wrap items-center gap-2">
              <Badge tone="blue">
                {answer.retrieval === "hybrid"
                  ? "RAG híbrido"
                  : "Busca textual"}
              </Badge>
              <span className="text-xs text-slate-500">
                Modelo: {answer.model}
              </span>
            </div>
            <div className="mt-4 whitespace-pre-wrap text-sm leading-7 text-slate-800">
              {answer.answer}
            </div>
            <div className="mt-6">
              <h3 className="flex items-center gap-2 font-bold">
                <MessageSquareText size={17} /> Fontes consultadas
              </h3>
              <div className="mt-3 grid gap-3">
                {answer.citations.map((citation) => (
                  <details
                    key={citation.sourceId}
                    className="rounded-xl border border-slate-200 bg-slate-50 p-4"
                  >
                    <summary className="cursor-pointer text-sm font-semibold">
                      [{citation.sourceId}] {citation.documentTitle} · página{" "}
                      {citation.pageNumber}
                    </summary>
                    <p className="mt-3 text-sm leading-6 text-slate-600">
                      {citation.excerpt}
                    </p>
                    <a
                      className="mt-3 inline-block text-xs font-semibold text-brand-primary"
                      href={
                        "/api/v1/documents/" + citation.documentId + "/download"
                      }
                    >
                      Abrir versão atual do documento
                    </a>
                  </details>
                ))}
              </div>
            </div>
            <p className="mt-5 rounded-xl bg-amber-50 p-3 text-xs leading-5 text-amber-900">
              {answer.disclaimer}
            </p>
            <div className="mt-4 flex items-center gap-2 text-sm text-slate-500">
              Esta resposta ajudou?
              <button
                type="button"
                aria-label="Resposta útil"
                onClick={() => void rate("helpful")}
                className={
                  "rounded-lg p-2 " +
                  (feedback === "helpful"
                    ? "bg-emerald-100 text-emerald-700"
                    : "hover:bg-slate-100")
                }
              >
                <ThumbsUp size={16} />
              </button>
              <button
                type="button"
                aria-label="Resposta não útil"
                onClick={() => void rate("not_helpful")}
                className={
                  "rounded-lg p-2 " +
                  (feedback === "not_helpful"
                    ? "bg-red-100 text-red-700"
                    : "hover:bg-slate-100")
                }
              >
                <ThumbsDown size={16} />
              </button>
            </div>
          </article>
        )}
      </Card>

      <Card className="h-fit">
        <h2 className="font-bold">Escopo documental</h2>
        <p className="mt-1 text-xs leading-5 text-slate-500">
          Sem seleção, todas as versões atuais com texto extraído são
          consideradas. Selecione documentos para restringir a consulta.
        </p>
        <div className="mt-4 space-y-2">
          {data.documents.length === 0 ? (
            <p className="rounded-xl bg-slate-50 p-3 text-sm text-slate-500">
              Este Matter ainda não possui documentos.
            </p>
          ) : (
            data.documents.map((document) => (
              <label
                key={document.id}
                className="flex cursor-pointer gap-3 rounded-xl border border-slate-200 p-3 text-sm hover:bg-slate-50"
              >
                <input
                  type="checkbox"
                  checked={selectedDocuments.includes(document.id)}
                  onChange={() => toggleDocument(document.id)}
                  className="mt-0.5"
                />
                <span>
                  <span className="block font-semibold">{document.title}</span>
                  <span className="text-xs text-slate-500">
                    v{document.versionNumber} · {document.category}
                  </span>
                </span>
              </label>
            ))
          )}
        </div>
      </Card>
    </div>
  );
}
