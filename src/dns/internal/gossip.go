

package internal

import (
	"encoding/json"
	"log"
	"net"
	"time"
)



func StartGossip(peers []string, s * Store){
	go func() {
		ln, err := net.Listen("tcp",":5300")

		if err != nil {
			log.Fatal(err)
		}

		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}

			go handleConn(conn, s)
		}
	}()


	go func(){
		for {
			payload := GossipMessage{Type:"update", Records: s.List()}

			b, _ := json.Marshal(payload)

			for _, peer := range peers {
				conn, err := net.DialTimeout("tcp",peer,time.Second)

				if err == nil{
					conn.Write(b)
					conn.Close()
				}
			}
			time.Sleep(10*time.Second)
		}
	}()
}


func handleConn(conn net.Conn, s* Store){
	defer conn.Close()

	var msg GossipMessage

	dec := json.NewDecoder(conn)

	if err := dec.Decode(&msg) ; err != nil {
		return
	}

	for _, r := range msg.Records {
		s.Add(r)
	}
}