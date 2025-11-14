

package internal

import "time"

//
// === Distributed DNS Core Data Structures ===
//
// This file defines the core data models used across the distributed DNS system:
// - Record: represents a single DNS entry stored locally.
// - GossipMessage: defines the structure of synchronization messages exchanged between nodes.
//
// Both are serialized as JSON when sent over the gossip protocol or HTTP API.
//


// Record represents a single DNS entry (name → IP) stored in the local database.
// It includes a TTL (time-to-live) and a timestamp to manage cache expiration
// and consistency across distributed nodes.
type Record struct {
    Name      string    `json:"name"`
    IP        string    `json:"ip"`
    TTL       int       `json:"ttl"`
    Timestamp time.Time `json:"timestamp"`
}


// GossipMessage defines the structure of a message exchanged between
// DNS nodes during synchronization (via gossip protocol).
// Messages can be of type "update" (new or modified records)
// or "delete" (to remove records that have expired or been withdrawn).
type GossipMessage struct {
    Type    string   `json:"type"` // "update" | "delete"
    Records []Record `json:"records"`
}