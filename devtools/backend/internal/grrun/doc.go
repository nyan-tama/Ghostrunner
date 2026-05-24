// Package grrun provides the core logic for the gr-run CLI, a one-shot
// executor for bulk-coding orchestration.
//
// # Overview
//
// grrun implements a single-task execution pipeline:
// acquire an exclusive lock, claim a task file from the kanban board,
// invoke Claude CLI with the /coding skill, classify the result, and
// send a notification. Each gr-run process handles exactly one task
// and then exits, making it safe to launch multiple instances in parallel.
//
// # Key Components
//
//   - [Config]: holds runtime parameters (project path, task file name,
//     locks directory).
//   - [Runner]: orchestrates the full pipeline via [Runner.Run].
//   - [AcquireLock]: obtains a per-project exclusive lock using flock(2)
//     with LOCK_NB so that concurrent invocations on the same project
//     fail fast instead of blocking.
//   - [ClaimTask]: moves a task file from the waiting directory to the
//     running directory using os.Rename for atomic claim.
//   - [ClassifyResult]: inspects the working tree after Claude finishes
//     and returns an [Outcome] value (completed, waiting_answer,
//     abnormal, needs_check, or lock_busy).
//   - [CommandExecutor]: function type that abstracts Claude CLI
//     invocation, allowing test doubles to be injected.
//
// # Design Decisions
//
//   - flock(2) is used for exclusive locking because it is automatically
//     released when the process exits (including crashes), avoiding stale
//     lock issues that PID-file schemes suffer from.
//   - os.Rename is used for task claiming because it is atomic on POSIX
//     filesystems, preventing two runners from claiming the same task.
//   - [CommandExecutor] is a function type rather than an interface to
//     keep the abstraction lightweight; tests supply a closure that
//     records calls and returns a predetermined exit code.
//   - [Notifier] mirrors service.NtfyService signatures without importing
//     the service package, keeping the dependency graph shallow.
package grrun
