package data

import (
	"fmt"
	"net"

	"github.com/qdm12/ddns-updater/internal/models"
)

func (db *database) Insert(record models.Record) (id int) {
	db.Lock()
	defer db.Unlock()
	db.data = append(db.data, record)
	return len(db.data) - 1
}

func (db *database) Select(id int) (record models.Record, err error) {
	db.RLock()
	defer db.RUnlock()
	if id < 0 {
		return record, fmt.Errorf("id %d cannot be lower than 0", id)
	}
	if id > len(db.data)-1 {
		return record, fmt.Errorf("no record config found for id %d", id)
	}
	return db.data[id], nil
}

func (db *database) SelectAll() (records []models.Record) {
	db.RLock()
	defer db.RUnlock()
	return db.data
}

func (db *database) SetPublicIP(ip net.IP) {
	db.Lock()
	defer db.Unlock()
	db.publicIP = ip
}

func (db *database) GetPublicIP() net.IP {
	db.RLock()
	defer db.RUnlock()
	return db.publicIP
}
