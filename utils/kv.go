package utils

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

type KVStore struct {
	mu    *sync.RWMutex
	chans map[string]chan string
}

func NewKVStore() *KVStore {
	return &KVStore{
		mu:    new(sync.RWMutex),
		chans: make(map[string]chan string),
	}
}

func (kv *KVStore) Put(ctx context.Context, key, val string) error {
	if key == "" {
		return errors.New("key is empty")
	}
	if val == "" {
		return errors.New("val is empty")
	}
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if _, exists := kv.chans[key]; !exists {
		kv.chans[key] = make(chan string, 1024)
	}
	if len(kv.chans[key]) >= 1024 {
		return errors.New("chan is full")
	}
	kv.chans[key] <- val
	return nil
}

func (kv *KVStore) Get(ctx context.Context, key string) (string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if _, exists := kv.chans[key]; !exists {
		kv.chans[key] = make(chan string, 1024)
	}
	ch := kv.chans[key]
	lenth := len(ch)
	if lenth == 0 {
		return "", nil
	}
	var i int = 0
	for {
		i++
		select {
		case val, exists := <-kv.chans[key]:
			if !exists {
				return "", errors.New("chan is closed")
			}
			if i >= lenth {
				return val, nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func (kv *KVStore) Watch(ctx context.Context, key string) (<-chan string, error) {
	kv.mu.RLock()
	defer kv.mu.RUnlock()
	if _, exists := kv.chans[key]; !exists {
		kv.chans[key] = make(chan string, 1024)
	}
	return kv.chans[key], nil
}

func (kv *KVStore) Delete(ctx context.Context, key string) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if ch, exists := kv.chans[key]; exists {
		close(ch)
		delete(kv.chans, key)
	}
	return nil
}

func (kv *KVStore) Close() {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	for key, ch := range kv.chans {
		close(ch)
		delete(kv.chans, key)
	}
}
