package dht

import (
	"code.google.com/p/vitess/go/cache"
	"math/rand"
)

const (
	// Values "inspired" by jch's dht.c.
	maxInfoHashes    = 16384
	maxInfoHashPeers = 2048
)

// For the inner map, the key address in binary form. value=ignored.
type peerContactsSet map[string]bool

func (p peerContactsSet) Size() int {
	return len(p)
}

func newPeerStore() *peerStore {
	return &peerStore{
		infoHashPeers:        cache.NewLRUCache(maxInfoHashes),
		localActiveDownloads: make(peerContactsSet),
	}
}

type peerStore struct {
	// cache of peers for infohashes. Each key is an infohash and the values are peerContactsSet.
	infoHashPeers *cache.LRUCache
	// infoHashes for which we are peers.
	localActiveDownloads map[string]bool
}

func (h *peerStore) size() int {
	length, _, _, _ := h.infoHashPeers.Stats()
	return int(length)
}

func (h *peerStore) get(ih string) peerContactsSet {
	c, ok := h.infoHashPeers.Get(ih)
	if !ok {
		return nil
	}
	contacts := c.(peerContactsSet)
	return contacts
}

// count shows the number of know peers for the given infohash.
func (h *peerStore) count(ih string) int {
	return len(h.get(ih))
}

// peerContacts returns a random set of 8 peers for the ih InfoHash.
func (h *peerStore) peerContacts(ih string) []string {
	c := make([]string, 0, kNodes)
	peers := h.get(ih)

	// peers is a map, but I need a randomized set of 8 nodes to return.
	// Choose eight continguous ones starting from a random position.
	// Pseudo-random and un-seeded is fine.
	first := rand.Intn(len(peers) - kNodes)
	i := -1
	for p, _ := range peers {
		i++
		if i < first {
			// ranging and skipping isn't very smart because I'll
			// be doing some 1000 skips. But it was a consicous
			// choice against using extra memory to store an extra
			// slice of nodes. I'll see if it becomes a problem
			// before improving it.
			continue
		}
		c = append(c, p)
		if len(c) >= kNodes {
			break
		}
	}
	return c
}

// updateContact adds peerContact as a peer for the provided ih. Returns true if the contact was added, false otherwise (e.g: already present) .
func (h *peerStore) addContact(ih string, peerContact string) bool {
	var peers peerContactsSet
	p, ok := h.infoHashPeers.Get(ih)
	if ok {
		peers = p.(peerContactsSet)
	} else {
		if h.size() > maxInfoHashes {
			return false
		}
		peers = peerContactsSet{}
		h.infoHashPeers.Set(ih, peers)
	}
	if len(peers) > maxInfoHashPeers {
		return false
	}
	if p := peers[peerContact]; !p {
		peers[peerContact] = true
		return true
	}
	return false
}

func (h *peerStore) addLocalDownload(ih string) {
	h.localActiveDownloads[ih] = true
}

func (h *peerStore) hasLocalDownload(ih string) bool {
	_, ok := h.localActiveDownloads[ih]
	return ok
}
