package handler

import (
	"bufio"
	"encoding/json"
	"net"
	"testing"
	"time"
)

// newTestManager crea un manager con timeouts cortos, sin goroutines de limpieza.
func newTestManager(ackTimeout time.Duration) *TCPConnectionManager {
	return &TCPConnectionManager{
		activeConnections: make(map[uint]*TCPConnection),
		commandQueues:     make(map[uint]*CommandQueue),
		inactivityTimeout: time.Minute,
		ackTimeout:        ackTimeout,
		maxRetries:        3,
	}
}

// fakeScooter conecta un net.Pipe al manager y lee lo que el backend le envía.
func fakeScooter(t *testing.T, m *TCPConnectionManager, id uint) (server net.Conn, lines chan map[string]any) {
	t.Helper()
	server, client := net.Pipe()
	lines = make(chan map[string]any, 10)
	go func() {
		sc := bufio.NewScanner(client)
		for sc.Scan() {
			var msg map[string]any
			if json.Unmarshal(sc.Bytes(), &msg) == nil {
				lines <- msg
			}
		}
	}()
	t.Cleanup(func() { client.Close(); server.Close() })
	m.RegisterConnection(id, server)
	return server, lines
}

func unlockMsg() []byte { return []byte(`{"command":"unlock"}`) }

// Reproduce el incidente 2026-11-06: un reintento pendiente cuando el scooter
// se desconecta hacía "panic: send on closed channel" y tumbaba el backend.
func TestDisconnectDuringRetryDoesNotPanic(t *testing.T) {
	m := newTestManager(30 * time.Millisecond)
	conn, lines := fakeScooter(t, m, 4)

	// Nadie espera el ACK más que el worker: reintenta hasta 3 veces.
	if _, err := m.SendToScooter(4, unlockMsg(), "", "req_test"); err != nil {
		t.Fatal(err)
	}
	<-lines // primer envío; el scooter nunca responde

	// Se desconecta mientras el worker está en el ciclo de reintentos.
	m.UnregisterConnection(4, conn)

	// Dar tiempo a que el worker venza el ackTimeout e intente reintentar.
	time.Sleep(150 * time.Millisecond)
	// Si hubiera panic, el proceso de test ya habría caído.
}

// Un comando cuyo waiter se rindió no debe reenviarse (unlock huérfano tras
// cancelar el viaje).
func TestAbandonedCommandIsNotRetried(t *testing.T) {
	m := newTestManager(60 * time.Millisecond)
	_, lines := fakeScooter(t, m, 4)

	ack, _, err := m.SendToScooterAndWaitACK(4, unlockMsg(), "", "req_test", 20*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if ack {
		t.Fatal("ack = true sin respuesta del scooter")
	}
	<-lines // primer envío

	select {
	case msg := <-lines:
		t.Fatalf("comando abandonado fue reenviado: %v", msg)
	case <-time.After(200 * time.Millisecond):
	}
}

// Un ACK real se reporta como éxito.
func TestACKReported(t *testing.T) {
	m := newTestManager(time.Second)
	_, lines := fakeScooter(t, m, 4)

	go func() {
		msg := <-lines
		m.ProcessACK(4, map[string]interface{}{"message_id": msg["message_id"]})
	}()

	ack, _, err := m.SendToScooterAndWaitACK(4, unlockMsg(), "", "req_test", time.Second)
	if err != nil || !ack {
		t.Fatalf("ack = %v, err = %v; want true, nil", ack, err)
	}
}

// Desconexión con un waiter esperando: debe reportar fallo, no éxito.
func TestDisconnectReportsFailureToWaiter(t *testing.T) {
	m := newTestManager(time.Second)
	conn, lines := fakeScooter(t, m, 4)

	go func() {
		<-lines
		m.UnregisterConnection(4, conn)
	}()

	ack, _, _ := m.SendToScooterAndWaitACK(4, unlockMsg(), "", "req_test", time.Second)
	if ack {
		t.Fatal("ack = true tras desconexión")
	}
}

// La goroutine del socket viejo no debe tumbar la conexión nueva.
func TestStaleUnregisterKeepsNewConnection(t *testing.T) {
	m := newTestManager(time.Second)
	oldConn, _ := fakeScooter(t, m, 4)
	fakeScooter(t, m, 4) // reconexión: reemplaza oldConn

	m.UnregisterConnection(4, oldConn)

	if !m.IsScooterOnline(4) {
		t.Fatal("la conexión nueva fue desregistrada por el socket viejo")
	}
}
