package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/amirhdev/crypto-trader/internal/domain"

	"github.com/tidwall/buntdb"
)

type Storage struct {
	db   *buntdb.DB
	lock sync.Mutex
}

func New(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dir, "orders.db")
	db, err := buntdb.Open(dbPath)
	if err != nil {
		return nil, err
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) SaveOrder(order domain.Order) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	data, err := json.Marshal(order)
	if err != nil {
		return err
	}

	return s.db.Update(func(tx *buntdb.Tx) error {
		_, _, err := tx.Set(order.ID(), string(data), nil)
		return err
	})
}

func (s *Storage) GetAllOrders() ([]domain.Order, error) {
	var orders []domain.Order

	err := s.db.View(func(tx *buntdb.Tx) error {
		return tx.AscendKeys("", func(key, value string) bool {
			var ord domain.Order
			if err := json.Unmarshal([]byte(value), &ord); err == nil {
				orders = append(orders, ord)
			}
			return true
		})
	})

	return orders, err
}

func (s *Storage) DeleteOrder(id string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.db.Update(func(tx *buntdb.Tx) error {
		_, err := tx.Delete(id)
		return err
	})
}
