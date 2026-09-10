import { useState } from "react"
import { motion } from "framer-motion"
import { Search, DownloadCloud } from "lucide-react"

export function GlowingInput() {
  const [isFocused, setIsFocused] = useState(false)
  const [value, setValue] = useState("")

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!value) return
    // Dispatch action here
    setValue("")
  }

  return (
    <div className="relative group w-full max-w-2xl">
      {/* Glow Effect */}
      <motion.div
        className="absolute -inset-0.5 bg-gradient-to-r from-blue-500 to-indigo-500 rounded-3xl opacity-20 blur-md group-hover:opacity-40 transition duration-500"
        animate={{
          opacity: isFocused ? 0.6 : 0.2,
          scale: isFocused ? 1.02 : 1,
        }}
      />
      
      <form onSubmit={handleSubmit} className="relative flex items-center w-full h-16 bg-slate-800/80 backdrop-blur-xl border border-white/10 rounded-3xl overflow-hidden shadow-2xl">
        <div className="pl-6 flex items-center justify-center text-slate-400">
          <Search className="w-5 h-5" />
        </div>
        <input
          type="url"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onFocus={() => setIsFocused(true)}
          onBlur={() => setIsFocused(false)}
          placeholder="Paste URL to download media..."
          className="w-full h-full bg-transparent px-4 text-slate-50 placeholder-slate-500 focus:outline-none font-medium"
        />
        <div className="pr-2">
          <button
            type="submit"
            disabled={!value}
            className="h-12 px-6 rounded-2xl bg-gradient-to-r from-blue-500 to-indigo-600 text-white font-medium flex items-center gap-2 hover:from-blue-400 hover:to-indigo-500 transition-all disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-indigo-500/25"
          >
            <DownloadCloud className="w-4 h-4" />
            <span>Download</span>
          </button>
        </div>
      </form>
    </div>
  )
}
