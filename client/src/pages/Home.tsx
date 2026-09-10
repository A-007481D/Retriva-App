import { GlowingInput } from "../components/GlowingInput"
import { MediaCard } from "../components/MediaCard"

const mockData = [
  {
    id: "1",
    title: "Avatar: The Way of Water",
    subtitle: "Avatar: The Way of Water is a premium movie available for download in high quality.",
    imageUrl: "https://images.unsplash.com/photo-1626814026160-2237a95fc5a0?q=80&w=2070&auto=format&fit=crop",
    progress: 100,
    size: "12 GB",
    status: "Completed" as const
  },
  {
    id: "2",
    title: "The Last of Us S01 E03",
    subtitle: "The Last of Us S01 E03 is a premium episode for the hit series.",
    imageUrl: "https://images.unsplash.com/photo-1605379399642-870262d3d051?q=80&w=2106&auto=format&fit=crop",
    progress: 100,
    size: "8.1 GB",
    status: "Completed" as const
  },
  {
    id: "3",
    title: "Spider-Man: Across the Spider-Verse",
    subtitle: "Spider-Man: across the two spider-verse.",
    imageUrl: "https://images.unsplash.com/photo-1635805737707-575885ab0820?q=80&w=1974&auto=format&fit=crop",
    progress: 45,
    size: "11 GB",
    status: "Transcoding" as const
  },
  {
    id: "4",
    title: "Black Panther: Wakanda Forever",
    subtitle: "Black Panther: Wakanda Forever for Black Panther Forever.",
    imageUrl: "https://images.unsplash.com/photo-1608889175123-8ee362201f81?q=80&w=2080&auto=format&fit=crop",
    progress: 0,
    size: "10 GB",
    status: "Queued" as const
  },
]

export function Home() {
  return (
    <div className="pt-8 pb-20">
      {/* Top Section */}
      <div className="flex flex-col items-center justify-center py-16 mb-12 relative">
        <h1 className="text-5xl font-bold tracking-tight text-white mb-6 text-center">
          Download <span className="text-gradient">Any Media</span>
        </h1>
        <p className="text-lg text-slate-400 mb-10 text-center max-w-xl">
          Paste a link from anywhere. We handle the extraction, transcoding, and formatting natively in the background.
        </p>
        <GlowingInput />
      </div>

      {/* Grid Section */}
      <div>
        <div className="flex items-center justify-between mb-8">
          <h2 className="text-2xl font-bold text-white tracking-tight">Recently Downloaded</h2>
          <div className="text-sm font-medium text-slate-400 cursor-pointer hover:text-white transition-colors">
            Sort by Date ▾
          </div>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
          {mockData.map((item) => (
            <MediaCard key={item.id} {...item} />
          ))}
        </div>
      </div>
    </div>
  )
}
