package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	signalFile string
	pidFile    string
)

func init() {
	if runtime.GOOS == "windows" {
		signalFile = "C:\\Temp\\stop_signal.txt"
		pidFile = "C:\\Temp\\pids.txt"
	} else {
		signalFile = "/tmp/stop_signal"
		pidFile = "/tmp/pids"
	}
}

var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "My Application",
}

var startMonitorCmd = &cobra.Command{
	Use:   "start-monitor [partner pid]",
	Short: "Starts the monitor process",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var partnerPID int
		if len(args) > 0 {
			var err error
			partnerPID, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("Invalid partner pid")
				os.Exit(1)
			}
		}

		monitor(partnerPID)
	},
}

var startWorkerCmd = &cobra.Command{
	Use:   "start-worker [partner pid]",
	Short: "Starts the worker process",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var partnerPID int
		if len(args) > 0 {
			var err error
			partnerPID, err = strconv.Atoi(args[0])
			if err != nil {
				fmt.Println("Invalid partner pid")
				os.Exit(1)
			}
		}

		worker(partnerPID)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the monitor and worker processes",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Creating stop signal file...")
		createSignalFile()
	},
}

func main() {
	rootCmd.AddCommand(startMonitorCmd)
	rootCmd.AddCommand(startWorkerCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.Execute()
}

func monitor(partnerPID int) {
	go writePIDPeriodically()
	if partnerPID == 0 {
		cmd := startWorker()
		partnerPID = cmd.Process.Pid
	}

	for {
		if isSignalFileExist() {
			fmt.Println("Signal file exist, exiting...")
			os.Exit(0)
		}

		if !isProcessAlive(partnerPID) {
			fmt.Println("Worker is not alive, restarting it...")
			cmd := startWorker()
			partnerPID = cmd.Process.Pid
		}

		fmt.Println("Monitoring...", time.Now().Format(time.DateTime))

		time.Sleep(time.Second)
	}
}

func worker(partnerPID int) {
	go writePIDPeriodically()
	if partnerPID == 0 {
		cmd := startMonitor()
		partnerPID = cmd.Process.Pid
	}

	for {
		if isSignalFileExist() {
			fmt.Println("Signal file exist, exiting...")
			os.Exit(0)
		}

		if !isProcessAlive(partnerPID) {
			fmt.Println("Monitor is not alive, restarting it...")
			cmd := startMonitor()
			partnerPID = cmd.Process.Pid
		}

		fmt.Println("Doing some work...", time.Now().Format(time.DateTime))
		// Do some work here...
		time.Sleep(time.Second)
	}
}

func startMonitor() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "start-monitor", strconv.Itoa(os.Getpid()))
	cmd.Start()
	addPIDToFile(cmd.Process.Pid)
	return cmd
}

func startWorker() *exec.Cmd {
	cmd := exec.Command(os.Args[0], "start-worker", strconv.Itoa(os.Getpid()))
	cmd.Start()
	addPIDToFile(cmd.Process.Pid)
	return cmd
}

func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, os.FindProcess always succeeds.
		// We need to send a signal to the process to check if it's alive.
		err := process.Signal(syscall.Signal(0))
		return err == nil
	} else {
		// On Unix, if os.FindProcess succeeds, the process is alive.
		return true
	}
}

func isSignalFileExist() bool {
	_, err := os.Stat(signalFile)
	return !os.IsNotExist(err)
}

func createSignalFile() {
	file, err := os.Create(signalFile)
	if err != nil {
		fmt.Println("Failed to create signal file:", err)
		os.Exit(1)
	}
	file.Close()
}

func stopAllProcesses() {
	pids, err := readPIDsFromFile()
	if err != nil {
		fmt.Println("Error reading PID file:", err)
		return
	}

	for _, pid := range pids {
		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Println("Error finding process:", err)
			continue
		}

		// Send SIGTERM (graceful stop) to the process
		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Println("Error sending signal:", err)
		}
	}
}

func writePIDPeriodically() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		addPIDToFile(os.Getpid())
	}
}

func readPIDsFromFile() ([]int, error) {
	data, err := ioutil.ReadFile(pidFile)
	if err != nil {
		if os.IsNotExist(err) {
			// File does not exist, return an empty list
			return []int{}, nil
		}
		// Some other error occurred, return it
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	pids := make([]int, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}

		pid, err := strconv.Atoi(line)
		if err != nil {
			return nil, err
		}

		pids = append(pids, pid)
	}

	return pids, nil
}

func addPIDToFile(pid int) {
	pids, err := readPIDsFromFile()
	if err != nil {
		fmt.Println("Error reading PID file:", err)
		return
	}

	// Check if the PID already exists
	for _, existingPID := range pids {
		if existingPID == pid {
			// PID already exists, do not write it again
			return
		}
	}

	// PID does not exist, write it to the file
	f, err := os.OpenFile(pidFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening PID file:", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(strconv.Itoa(pid) + "\n"); err != nil {
		fmt.Println("Error writing to PID file:", err)
	}
}
