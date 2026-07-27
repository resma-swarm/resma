import { Line, LineChart, ResponsiveContainer, YAxis } from "recharts"

interface SparklineProps {
  data: number[]
  color?: string
  width?: number
  height?: number
}

export function Sparkline({ data, color = "var(--chart-1)", width = 80, height = 28 }: SparklineProps) {
  if (!data || data.length < 2) {
    return <div style={{ width, height }} className="flex items-center justify-center text-[10px] text-muted-foreground">—</div>
  }

  const chartData = data.map((v, i) => ({ i, v }))

  return (
    <div style={{ width, height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 1, right: 1, bottom: 1, left: 1 }}>
          <YAxis domain={["dataMin", "dataMax"]} hide />
          <Line
            type="monotone"
            dataKey="v"
            stroke={color}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
