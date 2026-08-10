export type Branding = {
  firmId: string;
  systemTitle: string;
  firmDisplayName: string;
  primaryColor: string;
  secondaryColor: string;
  accentColor: string;
  sidebarStyle: string;
  borderRadiusStyle: string;
  clientPortalTitle: string;
  clientPortalWelcomeText: string;
  logoLightUrl?: string;
  logoDarkUrl?: string;
  faviconUrl?: string;
  supportEmail?: string;
};
export type User = {
  id: string;
  firmId: string;
  name: string;
  email: string;
  active: boolean;
  roles: string[];
  permissions: string[];
};
export type Firm = {
  id: string;
  displayName: string;
  slug: string;
  timezone: string;
  locale: string;
  setupCompleted: boolean;
};
export type Session = { user: User; firm: Firm; branding: Branding };
export type Matter = {
  id: string;
  type: string;
  internalNumber: string;
  title: string;
  description?: string;
  status: string;
  priority: string;
  confidentiality: string;
  openedAt: string;
  caseNumber?: string;
  legalAreaName?: string;
  responsibleName?: string;
  archivedAt?: string;
};
export type Deadline = {
  id: string;
  matterId: string;
  matterTitle: string;
  title: string;
  dueAt: string;
  status: string;
  priority: string;
  assigneeName?: string;
};
export type Task = {
  id: string;
  matterId?: string;
  matterTitle?: string;
  title: string;
  status: string;
  priority: string;
  dueAt?: string;
  assigneeName?: string;
};
export type NotificationPreferences = {
  emailDeadlines: boolean;
  emailTasks: boolean;
};
export type DocumentItem = {
  id: string;
  matterId?: string;
  title: string;
  category: string;
  versionNumber: number;
  originalFileName: string;
  mimeType: string;
  sizeBytes: number;
  createdAt: string;
  deletedAt?: string;
  clientVisible: boolean;
};
export type DocumentExtraction = {
  id: string;
  documentId: string;
  documentVersionId: string;
  status: "pending" | "processing" | "succeeded" | "failed" | "unsupported";
  provider?: string;
  language?: string;
  pageCount: number;
  averageConfidence?: number;
  attempts: number;
  errorCode?: string;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  pages: { pageNumber: number; content: string; confidence?: number }[];
};
export type MatterEvent = {
  id: string;
  type: string;
  summary: string;
  actorName: string;
  createdAt: string;
  clientVisible: boolean;
};
export type MatterDetail = {
  matter: Matter;
  timeline: MatterEvent[];
  documents: DocumentItem[];
  deadlines: Deadline[];
  tasks: Task[];
  parties: { id: string; name: string; role: string; side: string }[];
  notes: {
    id: string;
    content: string;
    visibility: string;
    authorName: string;
    createdAt: string;
  }[];
  financial: {
    feesCents: number;
    paymentsCents: number;
    expensesCents: number;
    courtCostsCents: number;
    reimbursementsCents: number;
    pendingCents: number;
    netCents: number;
  };
};
