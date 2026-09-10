import { motion } from "framer-motion"

interface MediaCardProps {
  title: string
  subtitle: string
  imageUrl: string
  progress?: number
  size: string
  status: "Completed" | "Queued" | "Transcoding"
}

export function MediaCard({ title, subtitle, imageUrl, progress, size, status }: MediaCardProps) {
  return (
    <motion.div
      whileHover={{ y: -4, scale: 1.01 }}
      transition={{ type: "spring", stiffness: 300, damping: 20 }}
      className="glass rounded-3xl p-4 cursor-pointer group flex flex-col h-[320px] relative overflow-hidden"
    >
      {/* Background glow on hover */}
      <div className="absolute inset-0 bg-gradient-to-br from-indigo-500/0 to-indigo-500/0 group-hover:from-indigo-500/10 group-hover:to-blue-500/10 transition-colors duration-500 rounded-3xl z-0" />
      
      {/* Thumbnail */}
      <div className="w-full h-40 rounded-2xl overflow-hidden relative z-10 bg-slate-800 border border-white/5">
        <img
          src={imageUrl}
          alt={title}
          className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-500"
        />
        <div className="absolute top-2 right-2 bg-black/60 backdrop-blur-md px-2 py-1 rounded-lg text-xs font-medium border border-white/10">
          4K Movie
        </div>
      </div>

      {/* Info */}
      <div className="mt-4 flex-1 flex flex-col relative z-10">
        <h3 className="font-semibold text-lg text-slate-50 line-clamp-1 group-hover:text-white transition-colors">{title}</h3>
        <p className="text-sm text-slate-400 mt-1 line-clamp-2 leading-relaxed">{subtitle}</p>
        
        <div className="mt-auto">
          {/* Progress / Status */}
          <div className="flex items-center justify-between text-xs font-medium text-slate-400 mb-2">
            <span>
              {status === "Transcoding" ? "Transcoding..." : status === "Queued" ? "Queued" : "Progress Bar | 222 kb/s"}
            </span>
            <span className="flex items-center gap-2">
              <span className="text-slate-50">{size}</span>
              <span className={status === "Completed" ? "text-emerald-400" : "text-blue-400"}>
                {status}
              </span>
            </span>
          </div>
          
          {/* Progress Bar Track */}
          <div className="w-full h-1.5 bg-slate-800 rounded-full overflow-hidden border border-white/5">
            <motion.div 
              className={`h-full ${status === 'Completed' ? 'bg-emerald-500' : 'bg-gradient-to-r from-blue-500 to-indigo-500'} rounded-full`}
              initial={{ width: 0 }}
              animate={{ width: `${progress || 100}%` }}
              transition={{ duration: 1, ease: "easeOut" }}
            />
          </div>
        </div>
      </div>
    </motion.div>
  )
}
