import { X } from 'lucide-react'

interface PromptModalProps {
  title: string
  content: string
  onClose: () => void
}

export default function PromptModal({ title, content, onClose }: PromptModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70" onClick={onClose}>
      <div
        className="bg-gray-800 rounded-xl w-[90vw] max-w-4xl max-h-[85vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-center justify-between p-4 border-b border-gray-700 flex-shrink-0">
          <h3 className="text-sm font-medium text-gray-200">{title}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-white">
            <X size={18} />
          </button>
        </div>
        <pre className="flex-1 overflow-auto p-4 text-xs text-gray-300 font-mono whitespace-pre-wrap break-all">
          {content}
        </pre>
      </div>
    </div>
  )
}
