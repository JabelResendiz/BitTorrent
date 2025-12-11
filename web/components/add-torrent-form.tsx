"use client"

import type React from "react"

import { useState } from "react"
import { Upload, FileText, FolderOpen, Container, Network, Hash, Users } from "lucide-react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"

type DiscoveryMode = "tracker" | "overlay"

export function AddTorrentForm() {
  const [discoveryMode, setDiscoveryMode] = useState<DiscoveryMode>("tracker")
  const [torrentFile, setTorrentFile] = useState<File | null>(null)
  const [containerName, setContainerName] = useState("")
  const [networkName, setNetworkName] = useState("net")
  const [folderPath, setFolderPath] = useState("")
  const [imageName, setImageName] = useState("client_img")

  const [port, setPort] = useState("6001")
  const [bootstrap, setBootstrap] = useState("")

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setTorrentFile(e.target.files[0])
    }
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    const torrentPath = `/app/src/archives/${torrentFile?.name || "ST.torrent"}`
    const archives = "/app/src/archives"

    let dockerCommand = `docker run -it --rm \\\n`
    dockerCommand += `  --name ${containerName} \\\n`
    dockerCommand += `  --network ${networkName} \\\n`
    dockerCommand += `  -v ${folderPath}:${archives} \\\n`

    if (discoveryMode === "overlay") {
      dockerCommand += `  -p ${port}:${port} \\\n`
    }

    dockerCommand += `  ${imageName} \\\n`
    dockerCommand += `  --torrent="${torrentPath}" \\\n`
    dockerCommand += `  --archives="${archives}" \\\n`
    dockerCommand += `  --hostname="${containerName}" \\\n`
    dockerCommand += `  --discovery-mode=${discoveryMode}`

    if (discoveryMode === "overlay") {
      dockerCommand += ` \\\n  --overlay-port=${port}`
      if (bootstrap) {
        dockerCommand += ` \\\n  --bootstrap=${bootstrap}`
      }
    }

    console.log("[v0] Generated Docker command:\n", dockerCommand)
    alert(`Docker command generated:\n\n${dockerCommand}`)
    // Aquí iría la lógica para ejecutar el comando
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
          <div className="space-y-3">
            <Label className="text-base font-semibold">Discovery Mode</Label>
            <RadioGroup
              value={discoveryMode}
              onValueChange={(value) => setDiscoveryMode(value as DiscoveryMode)}
              className="flex gap-4"
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
              required
            />
            <p className="text-sm text-muted-foreground">Used as both container name and hostname</p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="network-name" className="flex items-center gap-2">
              <Network className="size-4" />
              Network Name
            </Label>
            <Input
              id="network-name"
              type="text"
              placeholder="net"
              value={networkName}
              onChange={(e) => setNetworkName(e.target.value)}
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
              required
            />
            <p className="text-sm text-muted-foreground">Local path to mount as /app/src/archives in container</p>
          </div>

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
              required
            />
          </div>

          {discoveryMode === "overlay" && (
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
                />
                <p className="text-sm text-muted-foreground">Bootstrap client for gossip discovery (optional)</p>
              </div>
            </>
          )}

          {/* Submit Button */}
          <Button type="submit" className="w-full" size="lg">
            <Upload className="mr-2 size-4" />
            Generate Docker Command
          </Button>
        </form>
      </CardContent>
    </Card>
  )
}
