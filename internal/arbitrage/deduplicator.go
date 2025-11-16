package arbitrage

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"
)

// Deduplicator запобігає створенню дублікатів арбітражних можливостей
type Deduplicator struct {
	seen   map[string]time.Time // externalID -> timestamp
	mu     sync.RWMutex
	ttl    time.Duration
}

// NewDeduplicator створює новий Deduplicator
func NewDeduplicator(ttl time.Duration) *Deduplicator {
	d := &Deduplicator{
		seen: make(map[string]time.Time),
		ttl:  ttl,
	}

	// Періодично очищати старі записи
	go d.cleanup()

	return d
}

// IsDuplicate перевіряє чи це дублікат
func (d *Deduplicator) IsDuplicate(externalID string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if timestamp, exists := d.seen[externalID]; exists {
		// Перевірити чи не застарів запис
		if time.Since(timestamp) < d.ttl {
			return true
		}
	}

	return false
}

// Add додає ID до списку побачених
func (d *Deduplicator) Add(externalID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seen[externalID] = time.Now()
}

// cleanup періодично видаляє старі записи
func (d *Deduplicator) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.mu.Lock()
		now := time.Now()

		for id, timestamp := range d.seen {
			if now.Sub(timestamp) > d.ttl {
				delete(d.seen, id)
			}
		}

		d.mu.Unlock()
	}
}

// Size повертає кількість записів
func (d *Deduplicator) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.seen)
}

// Clear очищає всі записи
func (d *Deduplicator) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.seen = make(map[string]time.Time)
}

// GenerateArbitrageID генерує унікальний ID для арбітражної можливості
func GenerateArbitrageID(pair, buyExchange, sellExchange string, timestamp time.Time) string {
	// Округлити timestamp до 3 хвилин (щоб схожі можливості мали один ID)
	roundedTime := timestamp.Truncate(3 * time.Minute)

	data := fmt.Sprintf("%s:%s:%s:%d", pair, buyExchange, sellExchange, roundedTime.Unix())
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
