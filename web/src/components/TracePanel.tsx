import { useState } from 'react'
import { PanelRightOpen, PanelRightClose, Loader, Check, X } from 'lucide-react'
import type { TraceStep } from '../types'

export default function TracePanel({ steps }: { steps: TraceStep[] }) {
  const [collapsed, setCollapsed] = useState(false)

  if (collapsed) {
    return (
      <button
        onClick={() => setCollapsed(false)}
        className="w-10 h-10 flex items-center justify-center bg-gray-900 rounded-l-lg text-gray-400 hover:text-white border-l border-y border-gray-800"
        title="Show trace panel"
      >
        <PanelRightOpen size={18} />
      </button>
    )
  }

  return (
    <div className="w-80 bg-gray-900 border-l border-gray-800 flex flex-col flex-shrink-0">
      <div className="flex items-center justify-between p-3 border-b border-gray-800">
        <h3 className="text-sm font-semibold text-gray-300">Trace</h3>
        <button
          onClick={() => setCollapsed(true)}
          className="text-gray-500 hover:text-white transition-colors"
        >
          <PanelRightClose size={16} />
        </button>
      </div>
      <div className="flex-1 overflow-auto p-3 space-y-2">
        {steps.length === 0 && (
          <p className="text-xs text-gray-500 text-center py-8">
            No trace data yet. Send a message to see agent activity.
          </p>
        )}
        {steps.map((step: TraceStep) => (
          <div key={step.id} className="bg-gray-800 rounded-lg p-2.5 text-xs">
            <div className="flex items-center gap-2">
              {step.status === 'running' ? (
                <Loader size={12} className="animate-spin text-indigo-400 flex-shrink-0" />
              ) : step.status === 'error' ? (
                <X size={12} className="text-red-400 flex-shrink-0" />
              ) : (
                <Check size={12} className="text-green-400 flex-shrink-0" />
              )}
              <span className="text-gray-300 font-medium">{step.name}</span>
              <span className="text-gray-600 ml-auto flex-shrink-0">
                {step.type === 'plan' ? 'plan' : step.type === 'tool_start' ? 'start' : 'done'}
              </span>
            </div>
            {step.details && (
              <p className="text-gray-500 mt-1.5 break-all line-clamp-4">{step.details}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
