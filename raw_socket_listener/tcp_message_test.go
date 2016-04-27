package rawSocket

import (
	"bytes"
	_ "log"
	"strconv"
	"testing"
)

func buildPacket(isIncoming bool, Ack, Seq uint32, Data []byte) (packet *TCPPacket) {
	packet = &TCPPacket{
		Addr: "",
		Ack:  Ack,
		Seq:  Seq,
		Data: Data,
	}

	// For tests `listening` port is 0
	if isIncoming {
		packet.SrcPort = 1
	} else {
		packet.DestPort = 1
	}

	return packet
}

func buildMessage(p *TCPPacket) *TCPMessage {
	id := p.Addr + strconv.Itoa(int(p.DestPort)) + strconv.Itoa(int(p.Ack))

	isIncoming := false
	if p.SrcPort == 1 {
		isIncoming = true
	}

	m := NewTCPMessage(id, p.Seq, p.Ack, isIncoming)
	m.AddPacket(p)

	return m
}

func TestTCPMessagePacketsOrder(t *testing.T) {
	msg := buildMessage(buildPacket(true, 1, 1, []byte("a")))
	msg.AddPacket(buildPacket(true, 1, 2, []byte("b")))

	if !bytes.Equal(msg.Bytes(), []byte("ab")) {
		t.Error("Should contatenate packets in right order")
	}

	// When first packet have wrong order (Seq)
	msg = buildMessage(buildPacket(true, 1, 2, []byte("b")))
	msg.AddPacket(buildPacket(true, 1, 1, []byte("a")))

	if !bytes.Equal(msg.Bytes(), []byte("ab")) {
		t.Error("Should contatenate packets in right order")
	}

	// Should ignore packets with same sequence
	msg = buildMessage(buildPacket(true, 1, 1, []byte("a")))
	msg.AddPacket(buildPacket(true, 1, 1, []byte("a")))

	if !bytes.Equal(msg.Bytes(), []byte("a")) {
		t.Error("Should ignore packet with same Seq")
	}
}

func TestTCPMessageSize(t *testing.T) {
	msg := buildMessage(buildPacket(true, 1, 1, []byte("POST / HTTP/1.1\r\nContent-Length: 2\r\n\r\na")))
	msg.AddPacket(buildPacket(true, 1, 2, []byte("b")))

	if msg.BodySize() != 2 {
		t.Error("Should count only body", msg.BodySize())
	}

	if msg.Size() != 40 {
		t.Error("Should count all sizes", msg.Size())
	}
}

func TestTCPMessageIsFinished(t *testing.T) {
	methodsWithoutBodies := []string{"GET", "OPTIONS", "HEAD"}

	for _, m := range methodsWithoutBodies {
		msg := buildMessage(buildPacket(true, 1, 1, []byte(m+" / HTTP/1.1")))

		if !msg.IsFinished() {
			t.Error(m, " request should be finished")
		}
	}

	methodsWithBodies := []string{"POST", "PUT", "PATCH"}

	for _, m := range methodsWithBodies {
		msg := buildMessage(buildPacket(true, 1, 1, []byte(m+" / HTTP/1.1\r\nContent-Length: 1\r\n\r\na")))

		if !msg.IsFinished() {
			t.Error(m, " should be finished as body length == content length")
		}

		msg = buildMessage(buildPacket(true, 1, 1, []byte(m+" / HTTP/1.1\r\nContent-Length: 2\r\n\r\na")))

		if msg.IsFinished() {
			t.Error(m, " should not be finished as body length != content length")
		}
	}

	msg := buildMessage(buildPacket(true, 1, 1, []byte("UNKNOWN / HTTP/1.1\r\n\r\n")))
	if msg.IsFinished() {
		t.Error("non http or wrong methods considered as not finished")
	}

	// Responses
	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\n\r\n")))
	msg.RequestAck = 1
	if !msg.IsFinished() {
		t.Error("Should mark simple response as finished")
	}

	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\n\r\n")))
	msg.RequestAck = 0
	if msg.IsFinished() {
		t.Error("Should not mark responses without associated requests")
	}

	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\nTransfer-Encoding: chunked\r\n\r\n")))
	msg.RequestAck = 1

	if msg.IsFinished() {
		t.Error("Should mark chunked response as non finished")
	}

	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\nContent-Length: 0\r\n\r\n")))
	msg.RequestAck = 1

	if !msg.IsFinished() {
		t.Error("Should mark Content-Length: 0 respones as finished")
	}

	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\nContent-Length: 1\r\n\r\na")))
	msg.RequestAck = 1

	if !msg.IsFinished() {
		t.Error("Should mark valid Content-Length respones as finished")
	}

	msg = buildMessage(buildPacket(false, 1, 1, []byte("HTTP/1.1 200 OK\r\nContent-Length: 10\r\n\r\na")))
	msg.RequestAck = 1

	if msg.IsFinished() {
		t.Error("Should not mark not valid Content-Length respones as finished")
	}
}
