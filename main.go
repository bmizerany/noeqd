package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrInvalidAuth    = errors.New("invalid auth")
)

var (
	token = os.Getenv("NOEQ_TOKEN")
)

const (
	workerIdBits       = uint64(5)
	datacenterIdBits   = uint64(5)
	maxWorkerId        = int64(-1) ^ (int64(-1) << workerIdBits)
	maxDatacenterId    = int64(-1) ^ (int64(-1) << datacenterIdBits)
	sequenceBits       = uint64(12)
	workerIdShift      = sequenceBits
	datacenterIdShift  = sequenceBits + workerIdBits
	timestampLeftShift = sequenceBits + workerIdBits + datacenterIdBits
	sequenceMask       = int64(-1) ^ (int64(-1) << sequenceBits)

	// Tue, 21 Mar 2006 20:50:14.000 GMT
	twepoch = int64(1288834974657)
)

// Flags
var (
	wid   = flag.Int64("w", 0, "worker id")
	did   = flag.Int64("d", 0, "datacenter id")
	laddr = flag.String("l", "0.0.0.0:4444", "the address to listen on")
	lts   = flag.Int64("t", -1, "the last timestamp in milliseconds")
)

var (
	mu  sync.Mutex
	seq int64
)

func main() {
	parseFlags()
	acceptAndServe(mustListen())
}

func parseFlags() {
	flag.Parse()
	if *wid < 0 || *wid > maxWorkerId {
		log.Fatalf("worker id must be between 0 and %d", maxWorkerId)
	}

	if *did < 0 || *did > maxDatacenterId {
		log.Fatalf("datacenter id must be between 0 and %d", maxDatacenterId)
	}
}

func mustListen() net.Listener {
	l, err := net.Listen("tcp", *laddr)
	if err != nil {
		log.Fatal(err)
	}
	return l
}

func acceptAndServe(l net.Listener) {
	for {
		cn, err := l.Accept()
		if err != nil {
			log.Println(err)
		}

		go func() {
			err := serve(cn, cn)
			if err != io.EOF {
				log.Println(err)
			}
			cn.Close()
		}()
	}
}

func serve(r io.Reader, w io.Writer) error {
	if token != "" {
		err := auth(r)
		if err != nil {
			return err
		}
	}

	c := make([]byte, 1)
	for {
		// Wait for 1 byte request
		_, err := io.ReadFull(r, c)
		if err != nil {
			return err
		}

		n := uint(c[0])
		if n == 0 {
			// No authing at this point
			return ErrInvalidRequest
		}

		b := make([]byte, n*8)
		for i := uint(0); i < n; i++ {
			id, err := nextId()
			if err != nil {
				return err
			}

			off := i * 8
			b[off+0] = byte(id >> 56)
			b[off+1] = byte(id >> 48)
			b[off+2] = byte(id >> 40)
			b[off+3] = byte(id >> 32)
			b[off+4] = byte(id >> 24)
			b[off+5] = byte(id >> 16)
			b[off+6] = byte(id >> 8)
			b[off+7] = byte(id)
		}

		_, err = w.Write(b)
		if err != nil {
			return err
		}
	}

	panic("not reached")
}

func milliseconds() int64 {
	return time.Now().UnixNano() / 1e6
}

func nextId() (int64, error) {
	mu.Lock()
	defer mu.Unlock()

	ts := milliseconds()

	if ts < *lts {
		return 0, fmt.Errorf("time is moving backwards, waiting until %d\n", *lts)
	}

	if *lts == ts {
		seq = (seq + 1) & sequenceMask
		if seq == 0 {
			for ts <= *lts {
				ts = milliseconds()
			}
		}
	} else {
		seq = 0
	}

	*lts = ts

	id := ((ts - twepoch) << timestampLeftShift) |
		(*did << datacenterIdShift) |
		(*wid << workerIdShift) |
		seq

	return id, nil
}

func auth(r io.Reader) error {
	b := make([]byte, 2)
	_, err := io.ReadFull(r, b)
	if err != nil {
		return err
	}

	if b[0] != 0 {
		return ErrInvalidRequest
	}

	b = make([]byte, b[1])
	_, err = io.ReadFull(r, b)
	if err != nil {
		return err
	}

	if string(b) != token {
		return ErrInvalidAuth
	}

	return nil
}
