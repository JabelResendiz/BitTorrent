import { Download, Upload, HardDrive, Activity } from 'lucide-react'
import { Card, CardContent } from '@/components/ui/card'

export function StatsOverview() {
  return (
    <div className="mb-6 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-primary/10">
            <Activity className="size-6 text-primary" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Active Containers</p>
            <p className="text-2xl font-bold">3</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-success/10">
            <Download className="size-6 text-success" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Total Download</p>
            <p className="font-mono text-2xl font-bold">5.2 MB/s</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-warning/10">
            <Upload className="size-6 text-warning" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Total Upload</p>
            <p className="font-mono text-2xl font-bold">4.9 MB/s</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center gap-4 p-6">
          <div className="flex size-12 items-center justify-center rounded-lg bg-accent/10">
            <HardDrive className="size-6 text-accent" />
          </div>
          <div>
            <p className="text-sm text-muted-foreground">Storage Used</p>
            <p className="text-2xl font-bold">6.7 GB</p>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
