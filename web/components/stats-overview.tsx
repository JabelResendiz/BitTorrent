import { Download, Activity, Users, TrendingUp } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

interface StatsOverviewProps {
  activeContainers: number
  totalDownloadSpeed: number
  averageProgress?: number
}

export function StatsOverview({ activeContainers, totalDownloadSpeed, averageProgress = 0 }: StatsOverviewProps) {
  // Formatear velocidad de descarga
  const formatSpeed = (bytesPerSecond: number): string => {
    if (bytesPerSecond === 0) return '0 B/s'
    const k = 1024
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s']
    const i = Math.floor(Math.log(bytesPerSecond) / Math.log(k))
    return `${(bytesPerSecond / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  return (
    <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-primary/10">
            <Activity className="size-6 text-primary" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Active Containers</p>
            <p className="text-2xl font-bold">{activeContainers}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-success/10">
            <Download className="size-6 text-success" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Total Download Speed</p>
            <p className="font-mono text-2xl font-bold">{formatSpeed(totalDownloadSpeed)}</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-blue-500/10">
            <TrendingUp className="size-6 text-blue-500" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Average Progress</p>
            <p className="text-2xl font-bold">{Math.round(averageProgress)}%</p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
