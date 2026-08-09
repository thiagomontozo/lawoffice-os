package domain

import (
	"encoding/json"
	"time"
)

type Firm struct {
	ID             string  `json:"id"`
	LegalName      string  `json:"legalName"`
	DisplayName    string  `json:"displayName"`
	Slug           string  `json:"slug"`
	Email          string  `json:"email"`
	Phone          *string `json:"phone,omitempty"`
	Website        *string `json:"website,omitempty"`
	Timezone       string  `json:"timezone"`
	Locale         string  `json:"locale"`
	SetupCompleted bool    `json:"setupCompleted"`
}
type Branding struct {
	FirmID                  string    `json:"firmId"`
	SystemTitle             string    `json:"systemTitle"`
	FirmDisplayName         string    `json:"firmDisplayName"`
	LogoLightURL            *string   `json:"logoLightUrl,omitempty"`
	LogoDarkURL             *string   `json:"logoDarkUrl,omitempty"`
	FaviconURL              *string   `json:"faviconUrl,omitempty"`
	LoginBackgroundURL      *string   `json:"loginBackgroundUrl,omitempty"`
	PrimaryColor            string    `json:"primaryColor"`
	SecondaryColor          string    `json:"secondaryColor"`
	AccentColor             string    `json:"accentColor"`
	SidebarStyle            string    `json:"sidebarStyle"`
	BorderRadiusStyle       string    `json:"borderRadiusStyle"`
	SupportEmail            *string   `json:"supportEmail,omitempty"`
	SupportPhone            *string   `json:"supportPhone,omitempty"`
	Website                 *string   `json:"website,omitempty"`
	ClientPortalTitle       string    `json:"clientPortalTitle"`
	ClientPortalWelcomeText string    `json:"clientPortalWelcomeText"`
	UpdatedAt               time.Time `json:"updatedAt"`
}
type User struct {
	ID          string     `json:"id"`
	FirmID      string     `json:"firmId"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Active      bool       `json:"active"`
	LastLoginAt *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	Roles       []string   `json:"roles"`
	Permissions []string   `json:"permissions"`
}
type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	System      bool     `json:"system"`
	Permissions []string `json:"permissions"`
}
type Client struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	LegalName *string   `json:"legalName,omitempty"`
	TradeName *string   `json:"tradeName,omitempty"`
	Document  *string   `json:"document,omitempty"`
	Email     *string   `json:"email,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Notes     *string   `json:"notes,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}
type Contact struct {
	ID       string  `json:"id"`
	ClientID *string `json:"clientId,omitempty"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	Email    *string `json:"email,omitempty"`
	Phone    *string `json:"phone,omitempty"`
	Document *string `json:"document,omitempty"`
}
type Matter struct {
	ID                string     `json:"id"`
	Type              string     `json:"type"`
	InternalNumber    string     `json:"internalNumber"`
	Title             string     `json:"title"`
	Description       *string    `json:"description,omitempty"`
	LegalAreaID       *string    `json:"legalAreaId,omitempty"`
	LegalAreaName     *string    `json:"legalAreaName,omitempty"`
	Status            string     `json:"status"`
	Priority          string     `json:"priority"`
	ResponsibleUserID *string    `json:"responsibleUserId,omitempty"`
	ResponsibleName   *string    `json:"responsibleName,omitempty"`
	Confidentiality   string     `json:"confidentiality"`
	OpenedAt          time.Time  `json:"openedAt"`
	ArchivedAt        *time.Time `json:"archivedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	CaseNumber        *string    `json:"caseNumber,omitempty"`
	Court             *string    `json:"court,omitempty"`
}
type MatterDetail struct {
	Matter    Matter           `json:"matter"`
	Timeline  []MatterEvent    `json:"timeline"`
	Documents []Document       `json:"documents"`
	Deadlines []Deadline       `json:"deadlines"`
	Tasks     []Task           `json:"tasks"`
	Parties   []Party          `json:"parties"`
	Notes     []Note           `json:"notes"`
	Financial FinancialSummary `json:"financial"`
}
type Party struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Role     string  `json:"role"`
	Side     string  `json:"side"`
	Document *string `json:"document,omitempty"`
}
type Note struct {
	ID         string    `json:"id"`
	Content    string    `json:"content"`
	Visibility string    `json:"visibility"`
	AuthorName string    `json:"authorName"`
	CreatedAt  time.Time `json:"createdAt"`
}
type Document struct {
	ID               string     `json:"id"`
	MatterID         *string    `json:"matterId,omitempty"`
	ClientID         *string    `json:"clientId,omitempty"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	Category         string     `json:"category"`
	VersionNumber    int        `json:"versionNumber"`
	OriginalFileName string     `json:"originalFileName"`
	MimeType         string     `json:"mimeType"`
	SizeBytes        int64      `json:"sizeBytes"`
	Checksum         string     `json:"checksum"`
	ClientVisible    bool       `json:"clientVisible"`
	CreatedAt        time.Time  `json:"createdAt"`
	DeletedAt        *time.Time `json:"deletedAt,omitempty"`
}
type DocumentVersion struct {
	ID               string    `json:"id"`
	VersionNumber    int       `json:"versionNumber"`
	OriginalFileName string    `json:"originalFileName"`
	MimeType         string    `json:"mimeType"`
	SizeBytes        int64     `json:"sizeBytes"`
	Checksum         string    `json:"checksum"`
	CreatedByName    string    `json:"createdByName"`
	CreatedAt        time.Time `json:"createdAt"`
	Notes            *string   `json:"notes,omitempty"`
}
type Deadline struct {
	ID           string     `json:"id"`
	MatterID     string     `json:"matterId"`
	MatterTitle  string     `json:"matterTitle"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	DueAt        time.Time  `json:"dueAt"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	AssignedTo   *string    `json:"assignedTo,omitempty"`
	AssigneeName *string    `json:"assigneeName,omitempty"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}
type Task struct {
	ID           string     `json:"id"`
	MatterID     *string    `json:"matterId,omitempty"`
	MatterTitle  *string    `json:"matterTitle,omitempty"`
	Title        string     `json:"title"`
	Description  *string    `json:"description,omitempty"`
	AssignedTo   *string    `json:"assignedTo,omitempty"`
	AssigneeName *string    `json:"assigneeName,omitempty"`
	Status       string     `json:"status"`
	Priority     string     `json:"priority"`
	DueAt        *time.Time `json:"dueAt,omitempty"`
}
type Hearing struct {
	ID          string    `json:"id"`
	MatterID    string    `json:"matterId"`
	MatterTitle string    `json:"matterTitle"`
	Title       string    `json:"title"`
	Type        string    `json:"type"`
	ScheduledAt time.Time `json:"scheduledAt"`
	Location    *string   `json:"location,omitempty"`
	Status      string    `json:"status"`
}
type MatterEvent struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`
	Summary       string    `json:"summary"`
	ResourceType  *string   `json:"resourceType,omitempty"`
	ResourceID    *string   `json:"resourceId,omitempty"`
	ActorName     string    `json:"actorName"`
	ClientVisible bool      `json:"clientVisible"`
	CreatedAt     time.Time `json:"createdAt"`
}
type Workflow struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Active      bool            `json:"active"`
	Stages      []WorkflowStage `json:"stages"`
}
type WorkflowStage struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  *string         `json:"description,omitempty"`
	Color        string          `json:"color"`
	SortOrder    int             `json:"sortOrder"`
	Checklist    json.RawMessage `json:"checklist"`
	OnEnterTasks json.RawMessage `json:"onEnterTasks"`
}
type FinancialSummary struct {
	FeesCents           int64 `json:"feesCents"`
	PaymentsCents       int64 `json:"paymentsCents"`
	ExpensesCents       int64 `json:"expensesCents"`
	CourtCostsCents     int64 `json:"courtCostsCents"`
	ReimbursementsCents int64 `json:"reimbursementsCents"`
	PendingCents        int64 `json:"pendingCents"`
	NetCents            int64 `json:"netCents"`
}
type Notification struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Message      string     `json:"message"`
	ResourceType *string    `json:"resourceType,omitempty"`
	ResourceID   *string    `json:"resourceId,omitempty"`
	ReadAt       *time.Time `json:"readAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}
type AuditEvent struct {
	ID           string          `json:"id"`
	UserName     *string         `json:"userName,omitempty"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   *string         `json:"resourceId,omitempty"`
	Metadata     json.RawMessage `json:"metadata"`
	CreatedAt    time.Time       `json:"createdAt"`
}
type SearchGroup struct {
	Type      string  `json:"type"`
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Subtitle  string  `json:"subtitle"`
	MatchedBy string  `json:"matchedBy"`
	Score     float64 `json:"score"`
}
type ConflictResult struct {
	Possible   bool            `json:"possible"`
	Matches    []ConflictMatch `json:"matches"`
	Disclaimer string          `json:"disclaimer"`
}
type ConflictMatch struct {
	EntityType   string  `json:"entityType"`
	Name         string  `json:"name"`
	Relationship string  `json:"relationship"`
	MatterID     *string `json:"matterId,omitempty"`
	MatterTitle  *string `json:"matterTitle,omitempty"`
	Status       *string `json:"status,omitempty"`
	Responsible  *string `json:"responsible,omitempty"`
}
type CommandCenter struct {
	CriticalDeadlines []Deadline    `json:"criticalDeadlines"`
	TodayDeadlines    []Deadline    `json:"todayDeadlines"`
	Hearings          []Hearing     `json:"hearings"`
	MyTasks           []Task        `json:"myTasks"`
	RecentDocuments   []Document    `json:"recentDocuments"`
	PriorityMatters   []Matter      `json:"priorityMatters"`
	RecentActivity    []MatterEvent `json:"recentActivity"`
	ArchiveReady      int           `json:"archiveReady"`
}
