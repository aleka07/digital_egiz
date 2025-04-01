# Workflow State & Rules (STM + Rules + Log)

This file contains the dynamic state, embedded rules, active plan, and log for the current AI session working on the "Ditto+" framework project. It is read and updated frequently by the AI during its operational loop.

## State

Holds the current status of the workflow.

Phase: ANALYZE # Current workflow phase (ANALYZE, BLUEPRINT, CONSTRUCT, VALIDATE, BLUEPRINT_REVISE)
Status: READY # Current status (READY, IN_PROGRESS, BLOCKED_*, NEEDS_*, COMPLETED)
CurrentTaskID: null # Identifier for the main task being worked on
CurrentStep: null # Identifier for the specific step in the plan being executed
IsBlocked: false # Flag indicating if the AI is blocked

## Plan

Contains the step-by-step implementation plan generated during the BLUEPRINT phase.
*(This section will be populated by the AI during the BLUEPRINT phase based on the user's task)*

## Rules

Embedded rules governing the AI's autonomous operation within the Cursor environment.

# --- Core Workflow Rules ---

RULE_WF_PHASE_ANALYZE: Constraint: Goal is understanding request/context. NO solutioning or implementation planning. Action: Read relevant parts of project_config.md, ask clarifying questions if needed. Update State.Status to READY or NEEDS_CLARIFICATION. Log activity.

RULE_WF_PHASE_BLUEPRINT: Constraint: Goal is creating a detailed, unambiguous step-by-step plan in ## Plan. NO code implementation. Action: Based on analysis, generate plan steps. Set State.Status = NEEDS_PLAN_APPROVAL. Log activity.

RULE_WF_PHASE_BLUEPRINT_REVISE: Constraint: Goal is revising the ## Plan based on feedback or errors. NO code implementation. Action: Modify ## Plan. Set State.Status = NEEDS_PLAN_APPROVAL. Log activity.

RULE_WF_PHASE_CONSTRUCT: Constraint: Goal is executing the ## Plan exactly, step-by-step. NO deviation. If issues arise, trigger error handling (RULE_ERR_*) or revert phase (RULE_WF_TRANSITION_02). Action: Execute current plan step using tools (RULE_TOOL_*). Mark step complete in ## Plan. Update State.CurrentStep. Log activity.

RULE_WF_PHASE_VALIDATE: Constraint: Goal is verifying implementation against ## Plan and requirements using tools (lint, test). NO new implementation. Action: Run validation tools (RULE_TOOL_LINT_01, RULE_TOOL_TEST_RUN_01). Log results. Update State.Status based on results (TESTS_PASSED, BLOCKED_*, etc.).

RULE_WF_TRANSITION_01: Trigger: Explicit user command (@analyze, @blueprint, @construct, @validate) OR AI determines phase completion. Action: Update State.Phase accordingly. Log phase change. Set State.Status = READY for the new phase.

RULE_WF_TRANSITION_02: Trigger: AI determines current phase constraint prevents fulfilling user request OR error handling dictates phase change (e.g., RULE_ERR_HANDLE_TEST_01 failure). Action: Log the reason. Update State.Phase (e.g., to BLUEPRINT_REVISE). Set State.Status appropriately (e.g., NEEDS_PLAN_APPROVAL or BLOCKED_*). Report situation to user.

# --- Initialization & Resumption Rules ---

RULE_INIT_01: Trigger: AI session/task starts AND workflow_state.md is missing or empty. Action: 1. Create workflow_state.md with default structure & rules. 2. Read project_config.md (prompt user if missing). 3. Set State.Phase = ANALYZE, State.Status = READY, State.IsBlocked = false. 4. Log "Initialized new session." 5. Prompt user for the first task.

RULE_INIT_02: Trigger: AI session/task starts AND workflow_state.md exists. Action: 1. Read project_config.md. 2. Read existing workflow_state.md. 3. Log "Resumed session." 4. Evaluate State: If Status=COMPLETED, prompt for new task. If Status=READY/NEEDS_*, inform user and await input/action. If Status=BLOCKED_*, inform user of block. If Status=IN_PROGRESS, ask user to confirm continuation (triggers RULE_INIT_03).

RULE_INIT_03: Trigger: User confirms continuation via RULE_INIT_02 (for IN_PROGRESS state). Action: Set State.Status = READY for the current State.Phase. Log confirmation. Proceed with the next action based on loaded state and rules.

# --- Memory Management Rules ---

RULE_MEM_READ_LTM_01: Trigger: Start of ANALYZE phase or when context seems missing. Action: Read project_config.md. Log action.

RULE_MEM_READ_STM_01: Trigger: Beginning of each action cycle within the main loop. Action: Read current workflow_state.md to get latest State, Plan, etc.

RULE_MEM_UPDATE_STM_01: Trigger: After every significant AI action, tool execution, or receipt of user input. Action: Immediately update relevant sections (## State, ## Plan [marking steps], ## Log) in workflow_state.md and ensure it's saved/persisted.

RULE_MEM_UPDATE_LTM_01: Trigger: User command (@config/update <section> <content>) OR AI proposes change after successful VALIDATE phase for significant feature impacting core config. Action: Propose concise updates to project_config.md. Set State.Status = NEEDS_LTM_APPROVAL. Await user confirmation before modifying project_config.md. Log proposal/update.

RULE_MEM_CONSISTENCY_01: Trigger: After updating workflow_state.md or proposing LTM update. Action: Perform quick internal check (e.g., Does State.Phase match expected actions? Is Plan consistent?). If issues, log and set State.Status = NEEDS_CLARIFICATION or BLOCKED_INTERNAL_STATE.

# --- Tool Integration Rules (Cursor Environment) ---

RULE_TOOL_LINT_01: Trigger: Relevant source file saved during CONSTRUCT phase OR explicit @lint command. Action: 1. Identify target file(s). 2. Instruct Cursor terminal to run configured lint command (from project_config.md Tech Stack section). 3. Log attempt. 4. Parse output upon completion. 5. Log result (success/errors). 6. If errors, set State.Status = BLOCKED_LINT. Trigger RULE_ERR_HANDLE_LINT_01.

RULE_TOOL_FORMAT_01: Trigger: Relevant source file saved during CONSTRUCT phase OR explicit @format command. Action: 1. Identify target file(s). 2. Instruct Cursor to apply formatter (via command palette or terminal command from project_config.md Tech Stack section). 3. Log attempt and result.

RULE_TOOL_TEST_RUN_01: Trigger: Command @validate, entering VALIDATE phase, or after a fix attempt (RULE_ERR_HANDLE_TEST_01). Action: 1. Identify test suite/files to run based on context or project structure. 2. Instruct Cursor terminal to run configured test command (from project_config.md Tech Stack section). 3. Log attempt. 4. Parse output upon completion. 5. Log result (pass/fail details). 6. If failures, set State.Status = BLOCKED_TEST. Trigger RULE_ERR_HANDLE_TEST_01. If success, set State.Status = TESTS_PASSED (or READY if continuing CONSTRUCT/VALIDATE).

RULE_TOOL_APPLY_CODE_01: Trigger: AI generates code/modification during CONSTRUCT phase or for error fixing. Action: 1. Ensure code block is well-defined and targets the correct file/location. 2. Instruct Cursor to apply the code change (e.g., insert, replace selection, apply diff). 3. Log action and brief description of change.

RULE_TOOL_FILE_MANIP_01: Trigger: Plan step requires creating, deleting, or renaming a file. Action: 1. Use appropriate Cursor command/feature (e.g., @newfile, or simulate via terminal `touch`/`mkdir`/`rm`/`mv`). 2. Log action.

# --- Error Handling & Recovery Rules ---

RULE_ERR_HANDLE_LINT_01: Trigger: State.Status is BLOCKED_LINT. Action: 1. Analyze error message(s) from ## Log. 2. Attempt auto-fix if rule is simple (e.g., formatting, unused import, common style issues based on linter) and confidence is high. Apply fix using RULE_TOOL_APPLY_CODE_01. 3. Re-run lint via RULE_TOOL_LINT_01 on the fixed file. 4. If success, reset State.Status = READY (or previous status). Log fix. 5. If auto-fix fails or error is complex, set State.Status = BLOCKED_LINT_UNRESOLVED, State.IsBlocked = true. Report specific error and failed fix attempt to user. Request guidance.

RULE_ERR_HANDLE_TEST_01: Trigger: State.Status is BLOCKED_TEST. Action: 1. Analyze failure message(s) from ## Log. 2. Identify failing test(s) and related code area (using file paths, test names from output). 3. Attempt auto-fix if failure suggests simple, localized issue (e.g., assertion value mismatch, off-by-one error, simple mock setup) and confidence is high. Apply fix using RULE_TOOL_APPLY_CODE_01. 4. Re-run *failed* test(s) via RULE_TOOL_TEST_RUN_01 (if possible, else run relevant suite). 5. If success, reset State.Status = READY (or TESTS_PASSED if in VALIDATE). Log fix. 6. If auto-fix fails, error is complex (e.g., requires architectural change, logic redesign), or multiple tests fail cascade: Set State.Phase = BLUEPRINT_REVISE, State.Status = NEEDS_PLAN_APPROVAL, State.IsBlocked = true. Propose revised ## Plan step(s) based on failure analysis. Report situation and proposed plan revision to user.

RULE_ERR_HANDLE_GENERAL_01: Trigger: Unexpected error (tool execution failure not caught by specific handlers, internal inconsistency not caught by RULE_MEM_CONSISTENCY_01), ambiguity in plan/request, or situation not covered by other rules (e.g., Ditto API error, Kafka connection failure, DB error). Action: 1. Log detailed error/situation to ## Log. 2. Set State.Status = BLOCKED_UNKNOWN, State.IsBlocked = true. 3. Clearly report the issue (including relevant error messages from logs if possible) to the user and request specific instructions or clarification. Do not proceed until block is resolved by user input.

## Log

A chronological log of significant actions, events, tool outputs, and decisions made by the AI.

*   [YYYY-MM-DD HH:MM:SS UTC] Initialized new session. State set to ANALYZE/READY. Read project_config.md. Awaiting user task.
*   [2024-07-10 19:48:53 UTC] Resumed session. Read project_config.md and workflow_state.md. State is ANALYZE/READY. Awaiting user task.