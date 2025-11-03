

package dns

import (
    "net"
)

// buildResponse constructs a minimal DNS response for an A record
func BuildResponse(request []byte, ip string) []byte {
    response := make([]byte, len(request))
    copy(response, request)
    response[2] = 0x81 // Flags: response + recursion available
    response[3] = 0x80

    // QDCOUNT = 1, ANCOUNT = 1
    response[6] = 0
    response[7] = 1
    response[8] = 0
    response[9] = 1

    // Append answer section
    answer := make([]byte, 16)
    answer[0], answer[1] = 0xc0, 0x0c // pointer to domain name
    answer[2], answer[3] = 0x00, 0x01 // type A
    answer[4], answer[5] = 0x00, 0x01 // class IN
    answer[6], answer[7], answer[8], answer[9] = 0x00, 0x00, 0x00, 0x3c // TTL = 60
    answer[10], answer[11] = 0x00, 0x04 // data length = 4

    ipParts := net.ParseIP(ip).To4()
    copy(answer[12:], ipParts)

    return append(response, answer...)
}