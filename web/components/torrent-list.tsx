'use client'

import { useState } from 'react'
import { Download, Upload, Users, HardDrive, Play, Pause, Trash2 } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'

interface TorrentItem {
  id: string
  name: string
  containerName: string
  progress: number
  downloadSpeed: string
  uploadSpeed: string
  seeders: number
  leechers: number
  size: string
  downloaded: string
  status: 'downloading' | 'seeding' | 'paused'
}

// Mock data para demostración
const mockTorrents: TorrentItem[] = [
  {
    id: '1',
    name: 'Ubuntu-22.04-desktop-amd64.iso',
    containerName: 'torrent-client-01',
    progress: 67,
    downloadSpeed: '5.2 MB/s',
    uploadSpeed: '1.8 MB/s',
    seeders: 234,
    leechers: 89,
    size: '3.6 GB',
    downloaded: '2.4 GB',
    status: 'downloading',
  },
  {
    id: '2',
    name: 'Debian-12.0-netinst.iso',
    containerName: 'torrent-client-02',
    progress: 100,
    downloadSpeed: '0 KB/s',
    uploadSpeed: '3.1 MB/s',
    seeders: 156,
    leechers: 45,
    size: '650 MB',
    downloaded: '650 MB',
    status: 'seeding',
  },
  {
    id: '3',
    name: 'Fedora-Workstation-38-x86_64.iso',
    containerName: 'torrent-client-03',
    progress: 23,
    downloadSpeed: '0 KB/s',
    uploadSpeed: '0 KB/s',
    seeders: 87,
    leechers: 12,
    size: '2.1 GB',
    downloaded: '483 MB',
    status: 'paused',
  },
]

export function TorrentList() {
  const [torrents] = useState<TorrentItem[]>(mockTorrents)

  const getStatusColor = (status: TorrentItem['status']) => {
    switch (status) {
      case 'downloading':
        return 'text-primary'
      case 'seeding':
        return 'text-success'
      case 'paused':
        return 'text-muted-foreground'
      default:
        return 'text-foreground'
    }
  }

  const getStatusText = (status: TorrentItem['status']) => {
    switch (status) {
      case 'downloading':
        return 'Downloading'
      case 'seeding':
        return 'Seeding'
      case 'paused':
        return 'Paused'
      default:
        return 'Unknown'
    }
  }

  return (
    <div className="space-y-4">
      {torrents.map((torrent) => (
        <Card key={torrent.id}>
          <CardHeader className="pb-3">
            <div className="flex items-start justify-between">
              <div className="space-y-1">
                <CardTitle className="text-lg font-medium">
                  {torrent.name}
                </CardTitle>
                <CardDescription className="flex items-center gap-4 text-xs">
                  <span className="flex items-center gap-1">
                    <HardDrive className="size-3" />
                    {torrent.containerName}
                  </span>
                  <span className={getStatusColor(torrent.status)}>
                    {getStatusText(torrent.status)}
                  </span>
                </CardDescription>
              </div>
              <div className="flex gap-1">
                <Button variant="ghost" size="icon">
                  {torrent.status === 'paused' ? (
                    <Play className="size-4" />
                  ) : (
                    <Pause className="size-4" />
                  )}
                </Button>
                <Button variant="ghost" size="icon">
                  <Trash2 className="size-4" />
                </Button>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {/* Progress Bar */}
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">
                  {torrent.downloaded} / {torrent.size}
                </span>
                <span className="font-mono font-medium">
                  {torrent.progress}%
                </span>
              </div>
              <Progress value={torrent.progress} className="h-2" />
            </div>

            {/* Stats Grid */}
            <div className="grid grid-cols-2 gap-4 rounded-lg bg-muted/50 p-4 md:grid-cols-4">
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Download className="size-3" />
                  Download
                </div>
                <p className="font-mono text-sm font-medium">
                  {torrent.downloadSpeed}
                </p>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Upload className="size-3" />
                  Upload
                </div>
                <p className="font-mono text-sm font-medium">
                  {torrent.uploadSpeed}
                </p>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Users className="size-3" />
                  Seeders
                </div>
                <p className="font-mono text-sm font-medium text-success">
                  {torrent.seeders}
                </p>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Users className="size-3" />
                  Leechers
                </div>
                <p className="font-mono text-sm font-medium text-warning">
                  {torrent.leechers}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      ))}

      {torrents.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <HardDrive className="mb-4 size-12 text-muted-foreground" />
            <p className="text-center text-muted-foreground">
              No active torrents. Add a new torrent to get started.
            </p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
