package uiprogress

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/gosuri/uilive"
	"github.com/pkg/errors"
)

// Out is the default writer to render progress bars to
var Out = os.Stdout

// RefreshInterval in the default time duration to wait for refreshing the output
var RefreshInterval = time.Millisecond * 10

// Progress represents the container that renders progress bars
type Progress struct {
	// Out is the writer to render progress bars to
	Out io.Writer

	// Width is the width of the progress bars
	Width int

	// Bars is the collection of progress bars
	Bars []*Bar

	// RefreshInterval in the time duration to wait for refreshing the output
	RefreshInterval time.Duration

	lw     *uilive.Writer
	ticker *time.Ticker
	tdone  chan bool
	mtx    *sync.RWMutex
}

// CanOpenTerm returns true if this can be run under a tty
func CanOpenTerm() bool {
	if runtime.GOOS == "openbsd" {
		_, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		return err == nil
	}

	_, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	return err == nil
}

// New returns a new progress bar with defaults
func New() (*Progress, error) {
	if !CanOpenTerm() {
		return nil, errors.New("Unable to open tty")
	}

	lw := uilive.New()
	lw.Out = Out

	return &Progress{
		Width:           Width,
		Out:             Out,
		Bars:            make([]*Bar, 0),
		RefreshInterval: RefreshInterval,

		tdone: make(chan bool),
		lw:    uilive.New(),
		mtx:   &sync.RWMutex{},
	}, nil
}

func (p *Progress) SetOut(o io.Writer) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.Out = o
	p.lw.Out = o
}

func (p *Progress) SetRefreshInterval(interval time.Duration) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	p.RefreshInterval = interval
}

// AddBar creates a new progress bar and adds to the container
func (p *Progress) AddBar(total int) *Bar {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	bar := NewBar(total)
	bar.Width = p.Width
	p.Bars = append(p.Bars, bar)
	return bar
}

// Listen listens for updates and renders the progress bars
func (p *Progress) Listen() {
	for {

		p.mtx.Lock()
		interval := p.RefreshInterval
		p.mtx.Unlock()

		select {
		case <-time.After(interval):
			p.print()
		case <-p.tdone:
			p.print()
			close(p.tdone)
			return
		}
	}
}

func (p *Progress) print() {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	for _, bar := range p.Bars {
		fmt.Fprintln(p.lw, bar.String())
	}
	p.lw.Flush()
}

// Start starts the rendering the progress of progress bars. It listens for updates using `bar.Set(n)` and new bars when added using `AddBar`
func (p *Progress) Start() {
	go p.Listen()
}

// Stop stops listening
func (p *Progress) Stop() {
	p.tdone <- true
	<-p.tdone
}

// Bypass returns a writer which allows non-buffered data to be written to the underlying output
func (p *Progress) Bypass() io.Writer {
	return p.lw.Bypass()
}
