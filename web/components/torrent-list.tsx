'use client'

import { useState, useEffect } from 'react'
import { Clock, Download, Users, HardDrive, Play, Pause, Trash2, RefreshCw } from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'

const API_BASE_URL = 'http://localhost:7000/api'

interface TorrentItem {
  id: string
  name: string
  containerName: string
  progress: number
  downloadSpeed: string
  downloadSpeedBytes: number
  seeders: number
  leechers: number
  connectedPeers: number
  totalPeers: number
  size: string
  downloaded: string
  eta: string
  status: 'downloading' | 'seeding' | 'paused' | 'starting' | 'unknown'
  paused: boolean
}

interface TorrentListProps {
  onStatsUpdate?: (stats: { activeContainers: number; totalDownloadSpeed: number; averageProgress: number }) => void
}

export function TorrentList({ onStatsUpdate }: TorrentListProps) {
  const [torrents, setTorrents] = useState<TorrentItem[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  // Obtener lista de contenedores y sus stats
  const fetchTorrents = async () => {
    try {
      // 1. Obtener lista de contenedores
      const containersRes = await fetch(`${API_BASE_URL}/containers`)
      const containers = await containersRes.json()

      // Verificar que containers sea un array válido
      if (!containers || !Array.isArray(containers)) {
        setTorrents([])
        setLoading(false)
        if (onStatsUpdate) {
          onStatsUpdate({ activeContainers: 0, totalDownloadSpeed: 0, averageProgress: 0 })
        }
        return
      }

      // 2. Obtener status de cada contenedor
      const torrentsWithStats = await Promise.all(
        containers.map(async (container: any) => {
          try {
            const statusRes = await fetch(`${API_BASE_URL}/containers/${container.id}/status`)
            const status = await statusRes.json()

            return {
              id: container.id,
              name: status.torrent_name || container.name || 'BitTorrent Client',
              containerName: container.name,
              progress: Math.round(status.progress || 0),
              downloadSpeed: formatBytes(status.download_speed || 0) + '/s',
              downloadSpeedBytes: status.download_speed || 0,
              seeders: status.total_peers || 0,
              leechers: status.connected_peers || 0,
              connectedPeers: status.connected_peers || 0,
              totalPeers: status.total_peers || 0,
              size: formatBytes(status.total_size || 0),
              downloaded: formatBytes(status.downloaded || 0),
              eta: status.eta || '∞',
              status: mapState(status.state, status.paused),
              paused: status.paused || false,
            }
          } catch (error) {
            // Si no hay respuesta del HTTP server, mostrar estado básico
            return {
              id: container.id,
              name: 'BitTorrent Client',
              containerName: container.name,
              progress: 0,
              downloadSpeed: '0 B/s',
              downloadSpeedBytes: 0,
              seeders: 0,
              leechers: 0,
              connectedPeers: 0,
              totalPeers: 0,
              size: '0 B',
              downloaded: '0 B',
              eta: '∞',
              status: container.state === 'running' ? 'starting' : 'unknown',
              paused: false,
            }
          }
        })
      )

      setTorrents(torrentsWithStats)

      // Actualizar estadísticas generales
      if (onStatsUpdate) {
        const totalDownloadSpeed = torrentsWithStats.reduce(
          (sum, torrent) => sum + torrent.downloadSpeedBytes,
          0
        )
        const averageProgress = torrentsWithStats.length > 0
          ? torrentsWithStats.reduce((sum, torrent) => sum + torrent.progress, 0) / torrentsWithStats.length
          : 0
        
        onStatsUpdate({
          activeContainers: torrentsWithStats.length,
          totalDownloadSpeed,
          averageProgress,
        })
      }
    } catch (error) {
      console.error('Error fetching torrents:', error)
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  // Formatear bytes a tamaño legible
  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
  }

  // Mapear estado del contenedor
  const mapState = (state: string, paused: boolean): TorrentItem['status'] => {
    if (paused) return 'paused'
    if (state === 'downloading') return 'downloading'
    if (state === 'seeding' || state === 'completed') return 'seeding'
    if (state === 'starting') return 'starting'
    return 'unknown'
  }

  // Pausar contenedor
  const handlePause = async (containerId: string) => {
    try {
      await fetch(`${API_BASE_URL}/containers/${containerId}/pause`, {
        method: 'POST',
      })
      fetchTorrents() // Refrescar datos
    } catch (error) {
      console.error('Error pausing container:', error)
    }
  }

  // Reanudar contenedor
  const handleResume = async (containerId: string) => {
    try {
      await fetch(`${API_BASE_URL}/containers/${containerId}/resume`, {
        method: 'POST',
      })
      fetchTorrents() // Refrescar datos
    } catch (error) {
      console.error('Error resuming container:', error)
    }
  }

  // Detener contenedor (envía SIGINT, equivalente a Ctrl+C)
  const handleStop = async (containerId: string) => {
    if (!confirm('¿Estás seguro de que quieres detener este contenedor?')) return

    try {
      await fetch(`${API_BASE_URL}/containers/${containerId}/stop`, {
        method: 'POST',
      })
      fetchTorrents() // Refrescar datos
    } catch (error) {
      console.error('Error stopping container:', error)
    }
  }

  // Refrescar manualmente
  const handleRefresh = () => {
    setRefreshing(true)
    fetchTorrents()
  }

  // Cargar datos inicial y auto-refresh cada 3 segundos
  useEffect(() => {
    fetchTorrents()
    const interval = setInterval(fetchTorrents, 3000)
    return () => clearInterval(interval)
  }, [])

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <RefreshCw className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

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
                {/* Solo mostrar botones de pausa/reanudación si no está en seeding o completed */}
                {torrent.status !== 'seeding' && torrent.status !== 'completed' && (
                  <Button 
                    variant="ghost" 
                    size="icon"
                    onClick={() => 
                      torrent.status === 'paused' 
                        ? handleResume(torrent.id) 
                        : handlePause(torrent.id)
                    }
                  >
                    {torrent.status === 'paused' ? (
                      <Play className="size-4" />
                    ) : (
                      <Pause className="size-4" />
                    )}
                  </Button>
                )}
                <Button 
                  variant="ghost" 
                  size="icon"
                  onClick={() => handleStop(torrent.id)}
                >
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
            <div className="grid grid-cols-2 gap-4 rounded-lg bg-muted/50 p-4 md:grid-cols-3">
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Download className="size-3" />
                  Download Speed
                </div>
                <p className="font-mono text-sm font-medium">
                  {torrent.downloadSpeed}
                </p>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Users className="size-3" />
                  Peers
                </div>
                <p className="font-mono text-sm font-medium">
                  {torrent.connectedPeers}/{torrent.totalPeers}
                </p>
              </div>
              <div className="space-y-1">
                <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Clock className="size-3" />
                  ETA
                </div>
                <p className="font-mono text-sm font-medium">
                  {torrent.eta}
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
