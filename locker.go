package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type LockerState map[string]Lock

type Locker struct {
	LockConfig string
}

type Lock struct {
	LockedBy map[string]int `json:"locked_by"`
	Key      string         `json:"key"`
}

func (l Locker) GetLocks() (LockerState, error) {
	locks := LockerState{}

	data, err := ioutil.ReadFile(l.LockConfig)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return locks, nil
	}
	err = json.Unmarshal(data, &locks)
	if err != nil {
		return nil, err
	}
	return locks, nil
}
func (l Locker) SaveLocks(locks LockerState) error {
	data, err := json.MarshalIndent(locks, "", "    ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(l.LockConfig, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
func (l Locker) GetLock(pool string) (Lock, error) {
	locks, err := l.GetLocks()
	if err != nil {
		return Lock{}, err
	}
	lock := locks[pool]
	if lock.LockedBy == nil {
		lock.LockedBy = map[string]int{}
	}
	return lock, nil
}
func (l Locker) SetLock(pool string, lock Lock) error {
	locks, err := l.GetLocks()
	if err != nil {
		return err
	}
	locks[pool] = lock
	return l.SaveLocks(locks)
}
func (l Locker) Lock(pool, key, requestor string) error {
	lock, err := l.GetLock(pool)
	if err != nil {
		return err
	}
	if requestor == "" {
		requestor = key
	}
	if lock.Key != key && lock.Key != "" {
		return fmt.Errorf("Attempt to steal lock for '%s' with '%s' by '%s' thwarted. Currently held by someone else", pool, key, requestor)
	}
	lock.LockedBy[requestor] += 1
	lock.Key = key
	return l.SetLock(pool, lock)
}
func (l Locker) Unlock(pool, key, requestor string) error {
	lock, err := l.GetLock(pool)
	if err != nil {
		return err
	}
	if requestor == "" {
		requestor = key
	}
	if lock.Key != key && lock.Key != "" {
		return fmt.Errorf("Attempt to unlock '%s' with '%s' by '%s' thwarted. Currently held by someone else", pool, key, requestor)
	}

	lock.LockedBy[requestor] = 0
	totalLocks := 0
	for _, locks := range lock.LockedBy {
		totalLocks += locks
	}
	// reset the key if we are totally unlocked
	if totalLocks == 0 {
		lock.Key = ""
	}
	return l.SetLock(pool, lock)
}
