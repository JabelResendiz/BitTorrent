"use client"

import type React from "react"

import { useState } from "react"
import { Upload, FileText, FolderOpen, Container, Network, Hash, Users, RefreshCw } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"

type DiscoveryMode = "tracker" | "overlay"

const API_BASE_URL = 'http://localhost:7000/api'

interface AddTorrentFormProps {
  onSuccess?: () => void
}

export function AddTorrentForm({ onSuccess }: AddTorrentFormProps) {
  const [discoveryMode, setDiscoveryMode] = useState<DiscoveryMode>("tracker")
  const [torrentFile, setTorrentFile] = useState<File | null>(null)
  const [containerName, setContainerName] = useState("")
  const [networkName, setNetworkName] = useState("net")
  const [folderPath, setFolderPath] = useState("")
  const [imageName, setImageName] = useState("client_img")
  const [port, setPort] = useState("6001")
  const [bootstrap, setBootstrap] = useState("")
  const [loading, setLoading] = useState(false)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0]
      setTorrentFile(file)
      
      // Extraer la carpeta del archivo seleccionado
      // @ts-ignore - webkitRelativePath existe en algunos navegadores
      const fullPath = file.webkitRelativePath || file.name
      // Obtener directorio padre del archivo
      const pathParts = fullPath.split('/')
      if (pathParts.length > 1) {
        // Si tiene ruta completa, usar todo menos el nombre del archivo
        const folderPath = pathParts.slice(0, -1).join('/')
        setFolderPath(folderPath)
      } else {
        // Si solo es el nombre del archivo, intentar obtener la ruta del path del archivo
        // En el navegador esto es limitado por seguridad, así que dejamos vacío
        setFolderPath('')
      }
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (!torrentFile) {
      alert('Por favor selecciona un archivo .torrent')
      return
    }

    if (!containerName) {
      alert('Por favor ingresa un nombre para el contenedor')
      return
    }

    setLoading(true)

    try {
      // 1. Subir archivo .torrent
      const formData = new FormData()
      formData.append('file', torrentFile)

      const uploadRes = await fetch(`${API_BASE_URL}/torrents/upload`, {
        method: 'POST',
        body: formData,
      })

      if (!uploadRes.ok) {
        throw new Error('Failed to upload torrent file')
      }

      const uploadData = await uploadRes.json()
      console.log('Torrent uploaded:', uploadData)

      // 2. Crear contenedor con parámetros según el modo
      const createPayload: any = {
        containerName: containerName,
        networkName: networkName,
        torrentFile: torrentFile.name,
        imageName: imageName,
        folderPath: folderPath,
        discoveryMode: discoveryMode,
      }

      // Agregar parámetros específicos del modo overlay
      if (discoveryMode === 'overlay') {
        createPayload.port = port
        if (bootstrap) {
          createPayload.bootstrap = bootstrap
        }
      }

      const createRes = await fetch(`${API_BASE_URL}/containers`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(createPayload),
      })

      if (!createRes.ok) {
        throw new Error('Failed to create container')
      }

      const createData = await createRes.json()
      console.log('Container created:', createData)

      alert('✓ Contenedor creado exitosamente!')
      
      // Limpiar formulario
      setTorrentFile(null)
      setContainerName('')
      setFolderPath('')
      
      // Reset file input
      const fileInput = document.getElementById('torrent-file') as HTMLInputElement
      if (fileInput) fileInput.value = ''

      // Llamar callback de éxito para cambiar a la pestaña de torrents
      if (onSuccess) {
        onSuccess()
      }
    } catch (error) {
      console.error('Error:', error)
      alert('Error al crear el contenedor: ' + (error as Error).message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Upload className="size-5" />
          Add New Torrent Container
        </CardTitle>
        <CardDescription>Configure client container with tracker or gossip discovery mode</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Discovery Mode Selection */}
          <div className="space-y-3">
            <Label className="text-base font-semibold">Discovery Mode</Label>
            <RadioGroup
              value={discoveryMode}
              onValueChange={(value) => setDiscoveryMode(value as DiscoveryMode)}
              className="flex gap-4"
              disabled={loading}
            >
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="tracker" id="tracker" />
                <Label htmlFor="tracker" className="font-normal cursor-pointer">
                  Tracker Mode
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <RadioGroupItem value="overlay" id="overlay" />
                <Label htmlFor="overlay" className="font-normal cursor-pointer">
                  Gossip/Overlay Mode
                </Label>
              </div>
            </RadioGroup>
          </div>

          {/* Torrent File Upload */}
          <div className="space-y-2">
            <Label htmlFor="torrent-file" className="flex items-center gap-2">
              <FileText className="size-4" />
              Torrent File
            </Label>
            <Input
              id="torrent-file"
              type="file"
              accept=".torrent"
              onChange={handleFileChange}
              disabled={loading}
              className="cursor-pointer"
              required
            />
            {torrentFile && <p className="text-sm text-muted-foreground">Selected: {torrentFile.name}</p>}
          </div>

          {/* Container Name / Hostname */}
          <div className="space-y-2">
            <Label htmlFor="container-name" className="flex items-center gap-2">
              <Container className="size-4" />
              Container Name (Hostname)
            </Label>
            <Input
              id="container-name"
              type="text"
              placeholder="e.g., client1"
              value={containerName}
              onChange={(e) => setContainerName(e.target.value)}
              disabled={loading}
              required
            />
            <p className="text-sm text-muted-foreground">Used as both container name and hostname</p>
          </div>

          {/* Network Name */}
          <div className="space-y-2">
            <Label htmlFor="network-name" className="flex items-center gap-2">
              <Network className="size-4" />
              Network Name
            </Label>
            <Input
              id="network-name"
              type="text"
              placeholder="overlay_network"
              value={networkName}
              onChange={(e) => setNetworkName(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          {/* Folder Path */}
          <div className="space-y-2">
            <Label htmlFor="folder-path" className="flex items-center gap-2">
              <FolderOpen className="size-4" />
              Local Folder Path
            </Label>
            <Input
              id="folder-path"
              type="text"
              placeholder="~/Desktop/peers/1"
              value={folderPath}
              onChange={(e) => setFolderPath(e.target.value)}
              disabled={loading}
              required
            />
            <p className="text-sm text-muted-foreground">Local path to mount as /app/src/archives in container</p>
          </div>

          {/* Docker Image Name */}
          <div className="space-y-2">
            <Label htmlFor="image-name" className="flex items-center gap-2">
              <Hash className="size-4" />
              Docker Image Name
            </Label>
            <Input
              id="image-name"
              type="text"
              value={imageName}
              onChange={(e) => setImageName(e.target.value)}
              disabled={loading}
              required
            />
          </div>

          {/* Overlay Mode Specific Fields */}
          {discoveryMode === 'overlay' && (
            <>
              <div className="space-y-2">
                <Label htmlFor="port" className="flex items-center gap-2">
                  <Network className="size-4" />
                  Port (Overlay Port)
                </Label>
                <Input
                  id="port"
                  type="text"
                  placeholder="6001"
                  value={port}
                  onChange={(e) => setPort(e.target.value)}
                  disabled={loading}
                  required
                />
                <p className="text-sm text-muted-foreground">Used for both port mapping and overlay-port parameter</p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="bootstrap" className="flex items-center gap-2">
                  <Users className="size-4" />
                  Bootstrap Node
                </Label>
                <Input
                  id="bootstrap"
                  type="text"
                  placeholder="client1:6000"
                  value={bootstrap}
                  onChange={(e) => setBootstrap(e.target.value)}
                  disabled={loading}
                />
                <p className="text-sm text-muted-foreground">Bootstrap client for gossip discovery (optional)</p>
              </div>
            </>
          )}

          {/* Submit Button */}
          <Button type="submit" className="w-full" size="lg" disabled={loading}>
            {loading ? (
              <>
                <RefreshCw className="mr-2 size-4 animate-spin" />
                Creating...
              </>
            ) : (
              <>
                <Upload className="mr-2 size-4" />
                Create Container
              </>
            )}
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
