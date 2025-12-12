'use client'

import { useState, useEffect } from 'react'
import { Upload, FolderOpen, Server } from 'lucide-react'
import { AddTorrentForm } from '@/components/add-torrent-form'
import { TorrentList } from '@/components/torrent-list'
import { StatsOverview } from '@/components/stats-overview'

export default function Home() {
  const [activeTab, setActiveTab] = useState<'torrents' | 'add'>('torrents')
  const [stats, setStats] = useState({
    activeContainers: 0,
    totalDownloadSpeed: 0,
    averageProgress: 0,
  })

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border bg-card">
        <div className="mx-auto max-w-7xl px-4 py-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-lg bg-primary">
                <Server className="size-6 text-primary-foreground" />
              </div>
              <div>
                <h1 className="font-mono text-xl font-semibold text-foreground">
                  BitTorrent Client
                </h1>
                <p className="text-sm text-muted-foreground">
                  Container Management Dashboard
                </p>
              </div>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
        {/* Stats Overview */}
        <StatsOverview 
          activeContainers={stats.activeContainers}
          totalDownloadSpeed={stats.totalDownloadSpeed}
          averageProgress={stats.averageProgress}
        />

        {/* Tab Navigation */}
        <div className="mb-6 flex gap-2 border-b border-border">
          <button
            onClick={() => setActiveTab('torrents')}
            className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'torrents'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <FolderOpen className="size-4" />
            Active Torrents
          </button>
          <button
            onClick={() => setActiveTab('add')}
            className={`flex items-center gap-2 border-b-2 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'add'
                ? 'border-primary text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            <Upload className="size-4" />
            Add New Torrent
          </button>
        </div>

        {/* Tab Content */}
        <div className="space-y-6">
          {activeTab === 'add' && <AddTorrentForm onSuccess={() => setActiveTab('torrents')} />}
          {activeTab === 'torrents' && <TorrentList onStatsUpdate={setStats} />}
        </div>
      </main>
    </div>
  )
}
