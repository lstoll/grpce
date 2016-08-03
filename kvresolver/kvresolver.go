package kvresolver

import (
	"errors"
	"time"

	"github.com/lstoll/grpce/reporters"

	"google.golang.org/grpc/naming"
)

type pollResolver struct {
	target       string
	pollFunc     func(target string) ([]string, error)
	pollInterval time.Duration
	opts         *kvrOptions
}

type pollWatcher struct {
	updChan       chan []*naming.Update
	closeChan     chan struct{}
	currAddresses []string
}

type kvrOptions struct {
	errorReporter   reporters.ErrorReporter
	metricsReporter reporters.MetricsReporter
}

type KVROption func(*kvrOptions)

func WithErrorReporter(er reporters.ErrorReporter) KVROption {
	return func(o *kvrOptions) {
		o.errorReporter = er
	}
}

func WithMetricsReporter(mr reporters.MetricsReporter) KVROption {
	return func(o *kvrOptions) {
		o.metricsReporter = mr
	}
}

func New(target string, pollInterval time.Duration, pollFunc func(target string) ([]string, error), opts ...KVROption) naming.Resolver {
	kvo := &kvrOptions{}
	for _, opt := range opts {
		opt(kvo)
	}

	return &pollResolver{
		target:       target,
		pollFunc:     pollFunc,
		pollInterval: pollInterval,
		opts:         kvo,
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

	updateAddrs := func() error {
		updates := []*naming.Update{}
		addresses, err := p.pollFunc(p.target)
		if err != nil {
			reporters.ReportError(p.opts.errorReporter, err)
			reporters.ReportCount(p.opts.metricsReporter, "kvresolver.pollfunc.errors", 1)
			return err
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

		return nil
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
				if err := updateAddrs(); err != nil {
					break
				}
			}
		}
	}()

	// Initial seed. Do async to not block
	go updateAddrs()

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
