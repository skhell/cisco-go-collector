package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/skhell/cisco-go-collector/internal/device"
	"github.com/skhell/cisco-go-collector/internal/output"
	"github.com/skhell/cisco-go-collector/internal/progress"
	"github.com/skhell/cisco-go-collector/internal/sshrunner"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

func main() {
	outDir := flag.String("out", "output", "base output directory")
	workers := flag.Int("workers", 10, "number of devices processed in parallel")
	port := flag.Int("port", 22, "default SSH port")
	timeout := flag.Duration("timeout", 30*time.Second, "per-command and connection timeout")
	showVersion := flag.Bool("version", false, "print version information and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ciscocollector [options] <devices.csv>\n\n")
		fmt.Fprintf(os.Stderr, "Portable Cisco CLI collector written in Go for fast, CSV-driven configuration and command output collection..\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nCSV columns (required): datacenter, room, rack, hostname, ip, platform, category, command\n")
		fmt.Fprintf(os.Stderr, "CSV columns (optional): target_ip, vrf\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  ciscocollector devices.csv\n")
		fmt.Fprintf(os.Stderr, "  ciscocollector --out /tmp/audit --workers 20 devices.csv\n")
		fmt.Fprintf(os.Stderr, "  ciscocollector --port 2222 --timeout 60s devices.csv\n")
	}

	flag.Parse()

	if *showVersion {
		fmt.Printf("ciscocollector %s Tia Zanella https://skhell.com\n", Version)
		fmt.Println("This is free software; see the source for LICENSE. There is no")
		fmt.Println("warranty, not even for merchantability or fitness for any particular purpose.")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}
	csvPath := args[0]

	username, password, err := promptCredentials()
	if err != nil {
		log.Fatalf("reading credentials: %v", err)
	}

	devices, err := device.Load(csvPath)
	if err != nil {
		log.Fatalf("loading devices: %v", err)
	}
	if len(devices) == 0 {
		log.Fatalf("no devices with commands found in %s", csvPath)
	}

	runStamp := time.Now().UTC().Format("20060102-150405")

	tr := progress.New(len(devices))
	log.SetOutput(tr.LogWriter())

	jobs := make(chan *device.Device)
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Go(func() {
			for d := range jobs {
				processDevice(d, username, password, *port, *timeout, *outDir, runStamp, tr)
			}
		})
	}

	for _, d := range devices {
		jobs <- d
	}
	close(jobs)
	wg.Wait()

	tr.Stop()
	fmt.Printf("DONE. Output is saved under %s/%s\n", *outDir, runStamp)
}

func processDevice(d *device.Device, username, password string, port int, timeout time.Duration, outDir, runStamp string, tr *progress.Tracker) {
	tr.DeviceStarted(d.Hostname)
	defer tr.DeviceDone(d.Hostname)

	sess, err := sshrunner.Open(d.IP, port, username, password, timeout)
	if err != nil {
		log.Printf("[%s] connect failed: %v", d.Hostname, err)
		return
	}
	defer sess.Close()

	for _, cmd := range d.Commands {
		tr.CommandStarted(d.Hostname, cmd.Command)
		result, err := sess.Run(cmd.Command)
		if err != nil {
			log.Printf("[%s] command %q failed: %v", d.Hostname, cmd.Command, err)
			if errors.Is(err, sshrunner.ErrSessionDead) {
				break
			}
			continue
		}

		path := output.Path(outDir, runStamp, d.Datacenter, d.Room, d.Rack, d.Hostname, cmd.Category, cmd.Filename)
		if err := output.Write(path, result); err != nil {
			log.Printf("[%s] write failed for %q: %v", d.Hostname, cmd.Command, err)
		}
	}
}

func promptCredentials() (string, string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return "", "", err
	}
	username = trimNewline(username)

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", "", err
	}

	return username, string(passwordBytes), nil
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
