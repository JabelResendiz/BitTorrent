docker run  --name tracker   --network net   --publish 8081:8080 tracker

# Cliente en modo tracker
docker run -it --rm \
  --name client1 \
  --network net \
  -v ~/Desktop/peers/1:/app/src/archives \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client1" \
  --discovery-mode=tracker
# Cliente en modo overlay (necesita overlay-port y bootstrap)
docker run -it --rm \
  --name client2 \
  --network net \
  -v ~/Desktop/peers/2:/app/src/archives \
  -p 6001:6001 \
  client_img \
  --torrent="/app/src/archives/ST.torrent" \
  --archives="/app/src/archives" \
  --hostname="client2" \
  --discovery-mode=overlay \
  --overlay-port=6001 \
  --bootstrap=client1:6000