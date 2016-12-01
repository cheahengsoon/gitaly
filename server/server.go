package server

import (
	"bufio"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type Service struct {
	ch        chan bool
	waitGroup *sync.WaitGroup
}

type Callback func([]byte) []byte

func NewService() *Service {
	service := &Service{
		ch:        make(chan bool),
		waitGroup: &sync.WaitGroup{},
	}
	service.waitGroup.Add(1)
	return service
}

func (s *Service) Serve(address string, cb Callback) {
	listener, err := newListener(address)
	if err != nil {
		log.Fatalln(err)
	}
	defer s.waitGroup.Done()
	log.Println("Listening on address", address)
	for {
		select {
		case <-s.ch:
			log.Println("Received shutdown message, stopping server on", listener.Addr())
			listener.Close()
			return
		default:
		}
		listener.SetDeadline(time.Now().Add(1e9))
		conn, err := listener.AcceptTCP()
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			log.Println(err)
		}
		log.Println("Client connected from ", conn.RemoteAddr())
		s.waitGroup.Add(1)
		go s.serve(conn, cb)
	}
}

func newListener(address string) (*net.TCPListener, error) {
	tcpAddress, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return &net.TCPListener{}, err
	}
	return net.ListenTCP("tcp", tcpAddress)
}

func (s *Service) Stop() {
	close(s.ch)
	s.waitGroup.Wait()
}

func (s *Service) serve(conn *net.TCPConn, cb Callback) {
	defer conn.Close()
	defer s.waitGroup.Done()

	for {
		select {
		case <-s.ch:
			log.Println("Received shutdown message, disconnecting client from", conn.RemoteAddr())
			return
		default:
		}

		conn.SetDeadline(time.Now().Add(1e9))

		reader := bufio.NewReader(conn)
		buffer, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Client", conn.RemoteAddr(), "closed the connection")
				return
			}
			if opError, ok := err.(*net.OpError); ok && opError.Timeout() {
				continue
			}
			log.Println(err)
		}

		ret := cb(buffer)
		if _, err := conn.Write(ret); nil != err {
			log.Println(err)
			return
		}
	}
}
