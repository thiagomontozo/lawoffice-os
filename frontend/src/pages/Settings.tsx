import { Card, PageHeader } from "../app/ui";
export function SettingsPage() {
  return (
    <>
      <PageHeader
        title="Configurações"
        description="Preferências gerais, segurança e parâmetros do escritório."
      />
      <div className="grid gap-5 md:grid-cols-2">
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
