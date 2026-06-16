package cli

import (
	"bufio"
	"fmt"
	"os/exec"
	"time"

	"github.com/gdamore/tcell/v2"

	"mcsd/core"
)

type logLineEvent struct {
	when time.Time
	line string
}

func (e *logLineEvent) When() time.Time { return e.when }

func runConsole(instance *core.InstanceConfig) error {
	ports, err := instance.ReadPorts()
	if err != nil {
		return fmt.Errorf("read server.properties: %w", err)
	}

	rconClient, _ := core.DialRCON(fmt.Sprintf("127.0.0.1:%d", ports.RCON), ports.RCONPassword)
	if rconClient != nil {
		defer rconClient.Close()
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	defer screen.Fini()

	var (
		logs     []string
		inputBuf []rune
	)

	redStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)

	redraw := func() {
		width, height := screen.Size()
		screen.Clear()

		logRows := height - 2
		start := max(0, len(logs)-logRows)
		for row, line := range logs[start:] {
			for col, r := range []rune(line) {
				if col >= width {
					break
				}
				screen.SetContent(col, row, r, nil, tcell.StyleDefault)
			}
		}

		for x := range width {
			screen.SetContent(x, height-2, '─', nil, tcell.StyleDefault)
		}

		if rconClient == nil {
			msg := []rune("RCON unavailable — input disabled")
			for col, r := range msg {
				if col >= width {
					break
				}
				screen.SetContent(col, height-1, r, nil, redStyle)
			}
			screen.HideCursor()
		} else {
			prompt := []rune("> " + string(inputBuf))
			for col, r := range prompt {
				if col >= width {
					break
				}
				screen.SetContent(col, height-1, r, nil, tcell.StyleDefault)
			}
			screen.ShowCursor(2+len(inputBuf), height-1)
		}
		screen.Show()
	}

	unit := core.UnitName(instance.ID)

	jctlArgs := []string{"-u", unit, "-f", "--output=cat", "--no-pager"}
	if sdClient, err := core.NewSDClient(); err == nil {
		if since, err := sdClient.ActiveSince(unit); err == nil && !since.IsZero() {
			jctlArgs = append(jctlArgs, "--since", since.Format("2006-01-02 15:04:05"))
		} else {
			jctlArgs = append(jctlArgs, "-n", "50")
		}
		sdClient.Close()
	} else {
		jctlArgs = append(jctlArgs, "-n", "50")
	}

	jctl := exec.Command("journalctl", jctlArgs...)
	pipe, err := jctl.StdoutPipe()
	if err != nil {
		return err
	}
	if err := jctl.Start(); err != nil {
		return fmt.Errorf("journalctl: %w", err)
	}
	defer jctl.Process.Kill()

	go func() {
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			screen.PostEvent(&logLineEvent{when: time.Now(), line: scanner.Text()})
		}
		if err := scanner.Err(); err != nil {
			screen.PostEvent(&logLineEvent{when: time.Now(), line: "[journal error] " + err.Error()})
		}
	}()

	redraw()

	for {
		switch event := screen.PollEvent().(type) {
		case *logLineEvent:
			logs = append(logs, event.line)
			if len(logs) > 1000 {
				logs = logs[len(logs)-1000:]
			}
			redraw()

		case *tcell.EventResize:
			screen.Sync()
			redraw()

		case *tcell.EventKey:
			if event.Key() == tcell.KeyCtrlC || event.Key() == tcell.KeyCtrlD {
				return nil
			}
			if rconClient == nil {
				continue
			}
			switch event.Key() {
			case tcell.KeyEnter:
				cmd := string(inputBuf)
				inputBuf = inputBuf[:0]
				if cmd != "" {
					response, err := rconClient.Send(cmd)
					if err != nil {
						logs = append(logs, "[error] "+err.Error())
					} else if response != "" {
						logs = append(logs, "[rcon] "+response)
					}
				}
				redraw()

			case tcell.KeyBackspace, tcell.KeyBackspace2:
				if len(inputBuf) > 0 {
					inputBuf = inputBuf[:len(inputBuf)-1]
				}
				redraw()

			case tcell.KeyRune:
				inputBuf = append(inputBuf, event.Rune())
				redraw()
			}

		case *tcell.EventError:
			return event
		}
	}
}
