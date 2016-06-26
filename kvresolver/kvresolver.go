package grpcexperiments

import (
	"errors"
	"time"

	"google.golang.org/grpc/naming"
)

type pollResolver struct {
	target       string
	pollFunc     func(target string) ([]string, error)
	pollInterval time.Duration
}

type pollWatcher struct {
	updChan       chan []*naming.Update
	closeChan     chan struct{}
	currAddresses []string
}

func New(target string, pollInterval time.Duration, pollFunc func(target string) ([]string, error)) naming.Resolver {
	return &pollResolver{
		target:       target,
		pollFunc:     pollFunc,
		pollInterval: pollInterval,
	}
}

func (p *pollResolver) Resolve(target string) (naming.Watcher, error) {
	uc := make(chan []*naming.Update)
	cc := make(chan struct{})

	pw := &pollWatcher{
		updChan:       uc,
		closeChan:     cc,
		currAddresses: []string{},
	}

	go func() {
		ticker := time.NewTicker(p.pollInterval)
		for {
			select {
			case _, ok := <-cc:
				if !ok {
					ticker.Stop()
					return
				}
			case _ = <-ticker.C:
				updates := []*naming.Update{}
				addresses, err := p.pollFunc(p.target)
				if err != nil {
					// Retry later
					// TODO - log?
					break
				}
				// for each address that we found that isn't in the current state, send an update
				for _, a := range addresses {
					found := false
					for _, curr := range pw.currAddresses {
						if a == curr {
							found = true
							break
						}
					}
					if !found {
						updates = append(updates, &naming.Update{Op: naming.Add, Addr: a})
					}
				}

				// for each address that is in the current state but isn't in the found, send a delete
				for _, curr := range pw.currAddresses {
					found := false
					for _, a := range addresses {
						if curr == a {
							found = true
							break
						}
					}
					if !found {
						updates = append(updates, &naming.Update{Op: naming.Delete, Addr: curr})
					}
				}

				pw.currAddresses = addresses
				uc <- updates
			}
		}
	}()

	return pw, nil
}

func (p *pollWatcher) Close() {
	close(p.updChan)
}

func (p *pollWatcher) Next() ([]*naming.Update, error) {
	select {
	case ret, ok := <-p.updChan:
		if !ok {
			return nil, errors.New("Closed update channel")
		}
		return ret, nil
	case _, ok := <-p.closeChan:
		if !ok {
			return nil, errors.New("Warcher closed")
		}
	}
	return nil, errors.New("How did we get here")
}
