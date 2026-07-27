"use client"

import * as React from "react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { XIcon } from "lucide-react"
import { cn } from "@/lib/utils"

type InputTagsProps = Omit<React.ComponentProps<"input">, "value" | "onChange"> & {
  value: string[]
  onChange: React.Dispatch<React.SetStateAction<string[]>>
}

function InputTags({ className, value, onChange, ...props }: InputTagsProps) {
  const [pendingDataPoint, setPendingDataPoint] = React.useState("")

  React.useEffect(() => {
    if (pendingDataPoint.includes(",")) {
      const newDataPoints = new Set([
        ...value,
        ...pendingDataPoint.split(",").map((chunk) => chunk.trim()),
      ])
      onChange(Array.from(newDataPoints))
      setPendingDataPoint("")
    }
  }, [pendingDataPoint, onChange, value])

  const addPendingDataPoint = () => {
    if (pendingDataPoint) {
      const newDataPoints = new Set([...value, pendingDataPoint])
      onChange(Array.from(newDataPoints))
      setPendingDataPoint("")
    }
  }

  return (
    <div
      data-slot="input-tags"
      className={cn(
        "flex min-h-9 w-full flex-wrap items-center gap-2 rounded-md border border-input bg-transparent px-3 py-1.5 text-sm shadow-xs transition-[color,box-shadow] outline-none",
        "focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50",
        "disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30",
        className
      )}
    >
      {value.map((item) => (
        <Badge key={item} variant="secondary" className="gap-1">
          {item}
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="hover:bg-transparent hover:text-destructive"
            onClick={() => {
              onChange(value.filter((i) => i !== item))
            }}
          >
            <XIcon />
            <span className="sr-only">Remover {item}</span>
          </Button>
        </Badge>
      ))}
      <input
        className="flex-1 outline-none placeholder:text-muted-foreground"
        value={pendingDataPoint}
        onChange={(e) => setPendingDataPoint(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter" || e.key === ",") {
            e.preventDefault()
            addPendingDataPoint()
          } else if (
            e.key === "Backspace" &&
            pendingDataPoint.length === 0 &&
            value.length > 0
          ) {
            e.preventDefault()
            onChange(value.slice(0, -1))
          }
        }}
        {...props}
      />
    </div>
  )
}

export { InputTags, type InputTagsProps }
