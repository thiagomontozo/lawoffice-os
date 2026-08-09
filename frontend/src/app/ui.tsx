import type { ButtonHTMLAttributes, ReactNode } from "react";
import { LoaderCircle, Inbox } from "lucide-react";
export function Button({
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement>) {
  return (
    <button
      className={`rounded-xl bg-brand-primary px-4 py-2.5 text-sm font-semibold text-white transition hover:brightness-110 disabled:opacity-50 ${className}`}
      {...props}
    />
  );
}
export function Card({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={`rounded-2xl border border-slate-200 bg-white p-5 shadow-panel ${className}`}
    >
      {children}
    </section>
  );
}
export function Badge({
  children,
  tone = "slate",
}: {
  children: ReactNode;
  tone?: "slate" | "red" | "amber" | "green" | "blue";
}) {
  const c = {
    slate: "bg-slate-100 text-slate-700",
    red: "bg-red-100 text-red-700",
    amber: "bg-amber-100 text-amber-800",
    green: "bg-emerald-100 text-emerald-700",
    blue: "bg-blue-100 text-blue-700",
  }[tone];
  return (
    <span
      className={`inline-flex rounded-full px-2.5 py-1 text-xs font-semibold ${c}`}
    >
      {children}
    </span>
  );
}
export function PageHeader({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <header className="mb-6 flex flex-col justify-between gap-3 md:flex-row md:items-end">
      <div>
        <h1 className="text-2xl font-bold text-slate-950">{title}</h1>
        <p className="mt-1 text-sm text-slate-500">{description}</p>
      </div>
      {action}
    </header>
  );
}
export function Loading() {
  return (
    <div className="grid min-h-52 place-items-center text-slate-500">
      <LoaderCircle className="animate-spin" aria-label="Carregando" />
    </div>
  );
}
export function Empty({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="grid min-h-52 place-items-center rounded-2xl border border-dashed border-slate-300 bg-slate-50 p-8 text-center">
      <div>
        <Inbox className="mx-auto mb-3 text-slate-400" />
        <h3 className="font-semibold">{title}</h3>
        <p className="mt-1 text-sm text-slate-500">{description}</p>
      </div>
    </div>
  );
}
