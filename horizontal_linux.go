package horizontal

import (
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

func onResize() {
	ch := make(chan os.Signal, 1)
	sig := unix.SIGWINCH
	signal.Notify(ch, sig)
	go func() {
		for {
			select {
			case <-ch:
				resizeSeparator()
			}
		}
	}()
}
