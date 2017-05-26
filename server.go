package main

import (
	"fmt"
	"os"
)

type LockOperation string

const (
	LockOp   LockOperation = "Lock"
	UnlockOp               = "Unlock"
	ListOp                 = "ListLocks"
)

type LockStatus int

const (
	Locked LockStatus = iota
	Unlocked
	Error
	None
)

type LockRequest struct {
	Command  LockOperation
	Pool     string
	Lock     LockInput
	Response chan LockResponse
}
type LockResponse struct {
	Status  LockStatus
	Message interface{}
	Error   error
}

type LockInput struct {
	Key       string `json:"key"`
	Requestor string `json:"requestor"`
}

func lockServer(lockRequests chan LockRequest, lockConfig string) {
	locker := Locker{LockConfig: lockConfig}
	_, err := locker.GetLocks()
	if err != nil {
		panic(fmt.Sprintf("Unable to load lock config from '%s': %s", lockConfig, err))
	}
	for req := range lockRequests {
		if req.Command == ListOp {
			locks, err := locker.GetLocks()

			res := LockResponse{
				Status:  None,
				Message: locks,
				Error:   err,
			}
			req.Response <- res
		} else if req.Command == LockOp {
			res := LockResponse{}

			current, err := locker.GetLock(req.Pool)
			if err != nil {
				res.Status = Error
				res.Error = err
				req.Response <- res
				continue
			}

			err = locker.Lock(req.Pool, req.Lock.Key, req.Lock.Requestor)
			if err != nil {
				res.Status = Error
				res.Error = err
				req.Response <- res
				continue
			}
			current, err = locker.GetLock(req.Pool)
			if err != nil {
				res.Status = Error
				res.Error = err
				req.Response <- res
				continue
			}
			if current.Key != req.Lock.Key {
				res.Status = Error
				res.Error = fmt.Errorf("Locking failed. Should be locked by '%s', but found '%s'", req.Lock.Key, current.Key)
				req.Response <- res
				continue
			}

			res.Status = Locked
			res.Message = map[string]string{
				"response": fmt.Sprintf("Lock for '%s' acquired by '%s' using key '%s'", req.Pool, req.Lock.Requestor, req.Lock.Key),
			}
			req.Response <- res
		} else if req.Command == UnlockOp {
			res := LockResponse{}

			err := locker.Unlock(req.Pool, req.Lock.Key, req.Lock.Requestor)
			if err != nil {
				res.Status = Error
				res.Error = err
				req.Response <- res
				continue
			}

			res.Status = Unlocked
			res.Message = map[string]string{"response": fmt.Sprintf("'%s' released a lock on on '%s'", req.Lock.Requestor, req.Pool)}
			req.Response <- res
		} else {
			fmt.Fprintf(os.Stderr, "Invalid lock request '%s'", req.Command)
			req.Response <- LockResponse{Error: fmt.Errorf("Invalid lock request '%s'", req.Command)}
		}
	}
}
