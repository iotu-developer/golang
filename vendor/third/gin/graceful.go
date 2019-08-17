// reference: https://grisha.org/blog/2014/06/03/graceful-restart-in-golang/
package gin

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	ENV_CD_GIN_CONTINUE     = "CD_GIN_CONTINUE"
	ENV_CD_GIN_SOCKET_ORDER = "CD_GIN_SOCKET_ORDER"
)

var (
	isChild            bool
	socketPtrOffsetMap = map[string]uint{}
	runningServers     = []*ServerListener{}

	// graceful exit
	exitOnce sync.Once
)

type ServerListener struct {
	srv *http.Server
	ln  net.Listener
}

func init() {
	isChild = os.Getenv(ENV_CD_GIN_CONTINUE) != ""
	socketOrder := os.Getenv(ENV_CD_GIN_SOCKET_ORDER)
	for i, addr := range strings.Split(socketOrder, ",") {
		socketPtrOffsetMap[addr] = uint(i)
	}
}

// getListener either opens a new socket to listen on, or takes the acceptor socket
// it got passed when restarted.
func getListener(laddr string) (l net.Listener, err error) {
	if isChild {
		var ptrOffset uint = 0
		var found bool
		if len(socketPtrOffsetMap) > 0 {
			if ptrOffset, found = socketPtrOffsetMap[laddr]; !found {
				log.Printf("[addr:%s] ptroffset not found", laddr)
				os.Exit(1)
			}
			log.Println("laddr", laddr, "ptr offset", socketPtrOffsetMap[laddr])
		}

		f := os.NewFile(uintptr(3+ptrOffset), "")
		l, err = net.FileListener(f)
		if err != nil {
			err = fmt.Errorf("net.FileListener error: %v", err)
			return
		}
	} else {
		l, err = net.Listen("tcp", laddr)
		if err != nil {
			err = fmt.Errorf("net.Listen error: %v", err)
			return
		}
	}
	return
}

func getListenerFile(ln net.Listener) *os.File {
	tl := ln.(*net.TCPListener)
	f, _ := tl.File()
	return f
}

func fork() (err error) {
	files := make([]*os.File, len(runningServers))
	addrs := make([]string, len(runningServers))
	for i, sl := range runningServers {
		files[i] = getListenerFile(sl.ln)
		addrs[i] = sl.srv.Addr
	}

	env := append(
		os.Environ(),
		fmt.Sprintf("%s=1", ENV_CD_GIN_CONTINUE),
		fmt.Sprintf("%s=%s", ENV_CD_GIN_SOCKET_ORDER, strings.Join(addrs, ",")),
	)

	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		args = os.Args[1:]
	}

	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = files
	cmd.Env = env
	return cmd.Start()
}

func shutdownServers() {
	for _, sl := range runningServers {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		sl.srv.Shutdown(ctx)
		cancel()
	}
}

func gracefulExit(sig os.Signal) {
	onceFunc := func() {
		switch sig {
		case syscall.SIGUSR2:
			log.Printf("gin: graceful reloading ...")
			if err := fork(); err != nil {
				log.Printf("gin: hot reload failed:%v", err)
				return
			}
			shutdownServers()
			log.Printf("gin: graceful reload done!")
		default:
			log.Printf("gin: graceful exiting ...")
			shutdownServers()
			log.Printf("gin: graceful exit done!")
		}
	}
	exitOnce.Do(onceFunc)
}

func HandleSignal(signals ...os.Signal) {
	sig := make(chan os.Signal, 1)
	if len(signals) == 0 {
		signals = append(signals, os.Interrupt, syscall.SIGTERM, syscall.SIGUSR2)
	}
	signal.Notify(sig, signals...)

	s := <-sig
	log.Printf("gin: graceful exit action from signal [%s]", s.String())
	gracefulExit(s)
	log.Println("gin: ByeBye!")
}

func ShutdownServers() {
	shutdownServers()
}
