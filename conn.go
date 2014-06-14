package pooly

import (
	"net"
	"time"
)

// Conn abstracts user connections that are part of a Pool.
type Conn struct {
	iface     interface{}
	timer     *time.Timer
	timerStop chan bool
	closed    bool
	host      *host
}

// NewConn creates a new connection container, wrapping up a user defined connection object.
func NewConn(i interface{}) *Conn {
	return &Conn{
		iface:     i,
		timerStop: make(chan bool),
	}
}

// Interface returns an interface referring to the underlying user object.
func (c *Conn) Interface() interface{} {
	return c.iface
}

// NetConn is a helper for underlying user objects that satisfy
// the standard library net.Conn interface
func (c *Conn) NetConn() net.Conn {
	if c.iface == nil {
		return nil
	}
	return c.iface.(net.Conn)
}

func (c *Conn) isClosed() bool {
	return c.closed
}

func (c *Conn) setClosed() {
	if c.timer != nil {
		select {
		case c.timerStop <- true:
			c.timer.Stop()
		default:
		}
	}
	c.closed = true
}

func (c *Conn) setIdle(p *Pool) {
	if p.IdleTimeout > 0 {
		c.timer = time.NewTimer(p.IdleTimeout)
		go func() {
			select {
			case <-c.timerStop:
				return
			case <-c.timer.C:
				// The connection has been idle for too long,
				// send it to the garbage collector
				p.gc <- c
			}
		}()
	}
}

func (c *Conn) setActive() bool {
	if c.timer != nil {
		select {
		case c.timerStop <- true:
			c.timer.Stop()
		default:
			return false
		}
	}
	return true
}

func (c *Conn) setHost(h *host) {
	c.host = h
}

func (c *Conn) Release(e error, score float64) error {
	if c.host == nil {
		return ErrNoHostAvailable
	}
	if score < 0 || score > 1 {
		return ErrInvalidArg
	}

	h := c.host
	c.host = nil
	return h.releaseConn(c, e, score)
}

func (c *Conn) Address() (string, error) {
	if c.host == nil {
		return "", ErrNoHostAvailable
	}
	return c.host.pool.Address(), nil
}
