# Learnings

## ReAct Agent Loop Implementation

### Architecture
- `LoopState` is immutable: `WithObservation()` returns a new `LoopState` (copies observation slice)
- Channel sends must use `select` with `ctx.Done()` to avoid goroutine leaks
- `cancel()` from `context.WithTimeout` is called immediately after tool execution loop (NOT deferred), matching the spec requirement

### TDD Cycle
- Tests-first: wrote `loop_test.go` before any production code
- RED confirmed: compilation errors for missing `TurnPlan`, `ToolCall`, `Planner`, etc.
- GREEN: minimal production code to make 11 tests pass
- REFACTOR: removed redundant `sort` + `keys` slice in `callKey` (json.Marshal sorts map keys natively)

### Key Design Decisions
- `AgentLoop.Run()` returns `<-chan event.Event` for SSE-compatible streaming
- Event types: `plan`, `tool_start`, `tool_done`, `delta`, `done`, `error`
- `fixedExecutor` test mock delegates to the actual tool's `Execute()` method
- 20s per-step timeout via `context.WithTimeout`
- Dedup key format: `toolName:json_marshaled_args`

### Gotchas
- Go's `json.Marshal` on `map[string]any` produces sorted keys, making `sort.Strings` redundant
- Fixed executor mock initially didn't call the tool, making error propagation test fail
- Testing actual 20s timeout is impractical; test tool errors instead via an `errorTool`
