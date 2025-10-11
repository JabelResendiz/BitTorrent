

package peerwire

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)


type PeerConn struct {
	Conn net.Conn
	InfoHash [20]byte
	PeerId [20]byte
	AmChoking bool
	AmInterested bool
	PeerChoking bool
	PeerInterested bool
}

const (
	pstr = "BitTorrent protocol"
	ptrslen = 19
	HandshakeLen = 49 + ptrslen
)

const (
	MsgChoke = 0
	MsgUnchoke = 1
	MsgInterested = 2
	MsgNotInterested = 3
	MsgHave = 4
	MsgBitfiled = 5
	MsgRequest = 6
	MsgPiece = 7
	MsgCancel = 8
)