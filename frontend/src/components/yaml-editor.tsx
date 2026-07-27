import Editor, { OnMount } from "@monaco-editor/react"
import { setupMonacoYaml } from "@/components/monaco-config"

interface YamlEditorProps {
  value: string
  onChange?: (value: string) => void
  height?: string
  readOnly?: boolean
}

export function YamlEditor({ value, onChange, height = "400px", readOnly = false }: YamlEditorProps) {
  const handleMount: OnMount = (_, monaco) => {
    setupMonacoYaml(monaco)
  }

  return (
    <Editor
      height={height}
      defaultLanguage="yaml"
      value={value}
      theme="vs-dark"
      onMount={handleMount}
      options={{
        minimap: { enabled: false },
        fontSize: 13,
        wordWrap: "on",
        automaticLayout: true,
        scrollBeyondLastLine: false,
        tabSize: 2,
        readOnly,
        lineNumbers: "on",
        folding: true,
        renderWhitespace: "selection",
      }}
      onChange={(val) => onChange?.(val ?? "")}
    />
  )
}
