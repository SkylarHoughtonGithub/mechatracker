export function Stat({ label, value }: { label: string; value: string | number }) {
  return (
    <div className="bg-zinc-900 rounded-xl border border-zinc-800 p-3">
      <div className="text-xs text-zinc-500 mb-1">{label}</div>
      <div className="text-xl font-semibold text-zinc-100">{value}</div>
    </div>
  )
}
