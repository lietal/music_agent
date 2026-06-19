package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/music-agent/music-agent/internal/event"
	"github.com/music-agent/music-agent/internal/tool"
)

type AgentLoop struct {
	planner  Planner
	executor Executor
	tools    map[string]tool.Tool
}

func NewAgentLoop(planner Planner, executor Executor, tools map[string]tool.Tool) *AgentLoop {
	return &AgentLoop{
		planner:  planner,
		executor: executor,
		tools:    tools,
	}
}

func (l *AgentLoop) Run(ctx context.Context, state LoopState) <-chan event.Event {
	ch := make(chan event.Event, 64)

	go func() {
		defer close(ch)
		defer func() {
			if r := recover(); r != nil {
				errData, _ := json.Marshal(map[string]string{
					"message": fmt.Sprintf("panic: %v", r),
				})
				select {
				case ch <- event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData}:
				case <-ctx.Done():
				}
			}
		}()

		emit := func(evt event.Event) bool {
			select {
			case ch <- evt:
				return true
			case <-ctx.Done():
				return false
			}
		}

		planData, _ := json.Marshal(state.Goal)
		if !emit(event.Event{Type: event.TypePlan, RunID: state.RunID, Data: planData}) {
			return
		}

		for step := 0; step < state.MaxSteps; step++ {
			if ctx.Err() != nil {
				return
			}

			stepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

			plan, err := l.planner.Plan(stepCtx, state)
			if err != nil {
				errData, _ := json.Marshal(map[string]string{
					"message": fmt.Sprintf("plan error: %v", err),
				})
				emit(event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData})
				cancel()
				return
			}

			var lastObs tool.Observation
			for _, call := range plan.ToolCalls {
				key := callKey(call)
				if state.ExecutedCalls[key] {
					continue
				}

				startData, _ := json.Marshal(call)
				if !emit(event.Event{Type: event.TypeToolStart, RunID: state.RunID, Data: startData}) {
					cancel()
					return
				}

				t, ok := l.tools[call.ToolName]
				if !ok {
					errData, _ := json.Marshal(map[string]string{
						"message": fmt.Sprintf("tool not found: %s", call.ToolName),
					})
					emit(event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData})
					cancel()
					return
				}

				obs, err := l.executor.Execute(stepCtx, call, t)
				if err != nil {
					errData, _ := json.Marshal(map[string]string{
						"message": fmt.Sprintf("execution error: %v", err),
					})
					emit(event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData})
					cancel()
					return
				}

				state = state.WithObservation(obs)
				lastObs = obs

				if obs.Result.Error != "" {
					errData, _ := json.Marshal(map[string]string{
						"message": fmt.Sprintf("tool error: %s", obs.Result.Error),
					})
					emit(event.Event{Type: event.TypeError, RunID: state.RunID, Data: errData})
				}

				doneData, _ := json.Marshal(obs)
				if !emit(event.Event{Type: event.TypeToolDone, RunID: state.RunID, Data: doneData}) {
					cancel()
					return
				}
			}

			cancel()

			if l.planner.Next(lastObs) == FinalAnswer {
				break
			}
		}

	if gen, ok := l.planner.(AnswerGenerator); ok {
		textCh, err := gen.GenerateAnswer(ctx, state)
		if err == nil {
			for text := range textCh {
				deltaData, _ := json.Marshal(map[string]string{"message": text})
				if !emit(event.Event{Type: event.TypeDelta, RunID: state.RunID, Data: deltaData}) {
					return
				}
			}
		}
	}

	emit(event.Event{Type: event.TypeDone, RunID: state.RunID})

	}()

	return ch
}

func callKey(call ToolCall) string {
	argJSON, _ := json.Marshal(call.Args)
	return call.ToolName + ":" + string(argJSON)
}
