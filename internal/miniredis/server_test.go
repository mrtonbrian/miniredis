//go:build test
// +build test

package miniredis

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

func startTestServer(t *testing.T) (string, func()) {
	// Let the system pick an available port by using ":0".
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen on random port: %v", err)
	}
	addr := listener.Addr().String()

	srvStopped := make(chan struct{})
	go func() {
		defer close(srvStopped)
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-srvStopped:
					return
				default:
					return
				}
			}
			go func(c net.Conn) {
				if err := HandleConnection(c); err != nil {
					log.Println(err)
				}
			}(conn)
		}
	}()

	cleanup := func() {
		listener.Close()
		<-srvStopped
	}

	return addr, cleanup
}

// dialAndSend is a helper to connect to the mini-Redis server and send a list
// of commands. It returns the server's raw reply (as lines).
func dialAndSend(t *testing.T, addr string, commands []string) []string {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("failed to dial test server: %v", err)
	}
	defer conn.Close()

	for _, cmd := range commands {
		_, err := conn.Write([]byte(cmd))
		if err != nil {
			t.Fatalf("failed to write command: %v", err)
		}
	}

	reply := []string{}
	reader := bufio.NewReader(conn)

	// Wait for 100ms - relevant for pipeline tests
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			netErr, ok := err.(net.Error)
			if ok && netErr.Timeout() {
				// break out if we got a read timeout (not necessarily an error)
				break
			}
			// Otherwise it’s some unexpected error.
			break
		}
		reply = append(reply, strings.TrimRight(line, "\r\n"))
	}

	return reply
}

func TestBasicCommands(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	setFooBar := "*3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n"
	getFoo := "*2\r\n$3\r\nGET\r\n$3\r\nfoo\r\n"

	// Send both commands in sequence, read the replies
	replies := dialAndSend(t, addr, []string{setFooBar, getFoo})

	if len(replies) < 4 {
		t.Fatalf("expected at least 4 lines, got %d: %v", len(replies), replies)
	}
	// Checking quickly:
	if replies[0] != "$3" || replies[1] != "bar" {
		t.Errorf("SET response mismatch, got lines: %v", replies[:2])
	}
	if replies[2] != "$3" || replies[3] != "bar" {
		t.Errorf("GET response mismatch, got lines: %v", replies[2:4])
	}

	// 2) ECHO
	echoHello := "*2\r\n$4\r\nECHO\r\n$5\r\nHello\r\n"
	echoReplies := dialAndSend(t, addr, []string{echoHello})
	// Expect a bulk string "Hello"
	if len(echoReplies) < 2 || echoReplies[0] != "$5" || echoReplies[1] != "Hello" {
		t.Errorf("ECHO response mismatch, got: %v", echoReplies)
	}
}

func TestPipelining(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	// Send multiple commands in one shot
	commands := []string{
		// SET key1 val1
		"*3\r\n$3\r\nSET\r\n$4\r\nkey1\r\n$4\r\nval1\r\n",
		// GET key1
		"*2\r\n$3\r\nGET\r\n$4\r\nkey1\r\n",
		// SET key2 val2
		"*3\r\n$3\r\nSET\r\n$4\r\nkey2\r\n$4\r\nval2\r\n",
		// GET key2
		"*2\r\n$3\r\nGET\r\n$4\r\nkey2\r\n",
		// ECHO test
		"*2\r\n$4\r\nECHO\r\n$4\r\ntest\r\n",
	}

	replies := dialAndSend(t, addr, commands)

	if replies[0] != "$4" || replies[1] != "val1" {
		t.Errorf("first SET response mismatch:\n"+
			"  expected: $4, val1\n"+
			"  got: %v, %v", replies[0], replies[1])
	}

	// 2. GET key1
	if replies[2] != "$4" || replies[3] != "val1" {
		t.Errorf("first GET response mismatch:\n"+
			"  expected: $4, val1\n"+
			"  got: %v, %v", replies[2], replies[3])
	}

	// 3. SET key2 val2
	if replies[4] != "$4" || replies[5] != "val2" {
		t.Errorf("second SET response mismatch:\n"+
			"  expected: $4, val2\n"+
			"  got: %v, %v", replies[4], replies[5])
	}

	// 4. GET key2
	if replies[6] != "$4" || replies[7] != "val2" {
		t.Errorf("second GET response mismatch:\n"+
			"  expected: $4, val2\n"+
			"  got: %v, %v", replies[6], replies[7])
	}

	// 5. ECHO test
	if replies[8] != "$4" || replies[9] != "test" {
		t.Errorf("ECHO response mismatch:\n"+
			"  expected: $4, test\n"+
			"  got: %v, %v", replies[8], replies[9])
	}

	// Verify that values persist after pipeline execution by doing a separate GET
	getKey1 := "*2\r\n$3\r\nGET\r\n$4\r\nkey1\r\n"
	getKey2 := "*2\r\n$3\r\nGET\r\n$4\r\nkey2\r\n"

	// Check key1
	verifyReplies := dialAndSend(t, addr, []string{getKey1})
	if len(verifyReplies) != 2 || verifyReplies[0] != "$4" || verifyReplies[1] != "val1" {
		t.Errorf("key1 not persisted after pipeline:\n"+
			"  expected: $4, val1\n"+
			"  got: %v", verifyReplies)
	}

	// Check key2
	verifyReplies = dialAndSend(t, addr, []string{getKey2})
	if len(verifyReplies) != 2 || verifyReplies[0] != "$4" || verifyReplies[1] != "val2" {
		t.Errorf("key2 not persisted after pipeline:\n"+
			"  expected: $4, val2\n"+
			"  got: %v", verifyReplies)
	}
}

func TestConcurrentClients(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	numClients := 10
	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			defer wg.Done()

			// Each client sets a unique key
			key := fmt.Sprintf("key_%d", clientID)
			val := fmt.Sprintf("val_%d", clientID)

			setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
				len(key), key, len(val), val)
			getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n", len(key), key)

			replies := dialAndSend(t, addr, []string{setCmd, getCmd})
			// Check the replies
			// Expect 4 lines total: for the SET, and for the GET
			if len(replies) < 4 {
				t.Errorf("client %d: expected 4 lines, got %v", clientID, replies)
				return
			}
			// The GET reply lines should contain val
			// e.g. "$4" / "val_3"
			if !strings.Contains(replies[3], val) {
				t.Errorf("client %d: GET mismatch, expected %s, got %s", clientID, val, replies[3])
			}
		}(i)
	}

	wg.Wait()
}

func TestStressPipelined(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	numWorkers := 512
	iterations := 1024

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()

			// Each worker has one TCP connection
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				t.Errorf("worker %d: dial error: %v", workerID, err)
				return
			}
			defer conn.Close()

			var allCommands []byte
			expectedValues := make([]string, iterations)

			for i := 0; i < iterations; i++ {
				key := fmt.Sprintf("w_%d_%d", workerID, i)
				val := fmt.Sprintf("val_w_%d_%d", workerID, i)
				expectedValues[i] = val

				// SET key val
				setCmd := fmt.Sprintf(
					"*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
					len(key), key, len(val), val,
				)

				// GET key
				getCmd := fmt.Sprintf(
					"*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n",
					len(key), key,
				)

				allCommands = append(allCommands, setCmd...)
				allCommands = append(allCommands, getCmd...)
			}

			// Write all (SET, GET) commands at once
			_, err = conn.Write(allCommands)
			if err != nil {
				t.Errorf("worker %d: write error: %v", workerID, err)
				return
			}

			reader := bufio.NewReader(conn)

			for i := 0; i < iterations; i++ {
				_, err1 := reader.ReadString('\n')
				if err1 != nil {
					t.Errorf("worker %d: read set line1 err: %v", workerID, err1)
					return
				}
				_, err2 := reader.ReadString('\n')
				if err2 != nil {
					t.Errorf("worker %d: read set line2 err: %v", workerID, err2)
					return
				}

				line3, err3 := reader.ReadString('\n')
				if err3 != nil {
					t.Errorf("worker %d: read get line3 err: %v", workerID, err3)
					return
				}
				line4, err4 := reader.ReadString('\n')
				if err4 != nil {
					t.Errorf("worker %d: read get line4 err: %v", workerID, err4)
					return
				}

				gotVal := strings.TrimSpace(line4)
				expected := expectedValues[i]
				if gotVal != expected {
					// bad = true
					t.Errorf(
						"worker %d iteration %d: GET mismatch\n  want=%q\n  got=%q\n(lines:\n  %q\n  %q)",
						workerID, i, expected, gotVal, line3, line4,
					)
				}
			}
		}(w)
	}

	wg.Wait()
}

func TestLongBulkStrings(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	// Generate a large bulk string (>1MB). Let's use 2MB for this test.
	size := 2 * 1024 * 1024 // 2MB
	longValue := strings.Repeat("A", size)
	key := "large_key"

	setCmd := fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
		len(key), key, len(longValue), longValue)

	getCmd := fmt.Sprintf("*2\r\n$3\r\nGET\r\n$%d\r\n%s\r\n",
		len(key), key)

	replies := dialAndSend(t, addr, []string{setCmd, getCmd})

	if len(replies) < 4 {
		t.Fatalf("expected at least 4 lines, got %d: %v", len(replies), replies)
	}

	// Validate SET response
	expectedSetPrefix := fmt.Sprintf("$%d", len(longValue))
	if replies[0] != expectedSetPrefix {
		t.Errorf("SET response length mismatch, got: %s, want: %s", replies[0], expectedSetPrefix)
	}
	if replies[1] != longValue {
		t.Errorf("SET response value mismatch, expected length %d, got different value", len(longValue))
	}

	// Validate GET response
	expectedGetPrefix := fmt.Sprintf("$%d", len(longValue))
	if replies[2] != expectedGetPrefix {
		t.Errorf("GET response length mismatch, got: %s, want: %s", replies[2], expectedGetPrefix)
	}
	if replies[3] != longValue {
		t.Errorf("GET response value mismatch, expected length %d, got different value", len(longValue))
	}
}
