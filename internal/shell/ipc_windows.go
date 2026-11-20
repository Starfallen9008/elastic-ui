//go:build windows

package shell

import (
	"io"
	"time"

	"github.com/Microsoft/go-winio"
	"golang.org/x/sys/windows"
)

const (
	mutexName = `Local\ElasticDefendConsoleMutex`
	pipeName  = `\\.\pipe\ElasticDefendConsole`
)

func AcquireSingleInstance() (release func(), alreadyRunning bool, err error) {
	name, err := windows.UTF16PtrFromString(mutexName)
	if err != nil {
		return nil, false, err
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			_ = windows.CloseHandle(handle)
			return func() {}, true, nil
		}
		return nil, false, err
	}
	return func() { _ = windows.CloseHandle(handle) }, false, nil
}

func StartPipeServer(handler func(string)) (stop func(), err error) {
	listener, err := winio.ListenPipe(pipeName, nil)
	if err != nil {
		return nil, err
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			data, _ := io.ReadAll(io.LimitReader(conn, 32*1024))
			_ = conn.Close()
			if len(data) > 0 {
				handler(string(data))
			}
		}
	}()
	return func() { _ = listener.Close() }, nil
}

func ForwardScan(path string) error {
	var lastErr error
	for i := 0; i < 10; i++ {
		conn, err := winio.DialPipe(pipeName, nil)
		if err == nil {
			_, writeErr := conn.Write([]byte(path))
			_ = conn.Close()
			return writeErr
		}
		lastErr = err
		time.Sleep(120 * time.Millisecond)
	}
	return lastErr
}
