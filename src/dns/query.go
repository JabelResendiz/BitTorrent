
package dns

// parseQuery reads the domain name from the DNS query
func ParseQuery(data []byte) (string, error) {
    var domain string
    for i:=12; data[i] != 0; {
        length := int(data[i])
        domain += string(data[i+1:i+1+length]) + "."
        i += length + 1
    }
    return domain, nil
}
