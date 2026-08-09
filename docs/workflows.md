# Workflows

A `WorkflowDefinition` belongs to a firm and may target a Matter type. Ordered `WorkflowStage` rows contain name, description, color, checklist, default role and declarative `onEnterTasks`. `WorkflowTransition` records allowed edges; `WorkflowInstance` records a Matter's current stage.

The builder is intentionally simpler than BPMN: create, rename, color and reorder stages; define checklists and entry tasks; select a default responsibility. On transition the service can create tasks with a relative internal deadline and append `workflow.transitioned` to the Matter timeline.

V0.1 has no arbitrary scripts, branching expression language, distributed worker or external trigger system. Approval flows, richer rules and idempotent background automation belong to later milestones.
