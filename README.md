# Twin-Process

Twin-Process 是一个用 Go 语言编写的程序，其主要功能是创建和管理两个相互监视的进程。如果一个进程停止，另一个进程就会启动它。程序支持在 Windows 和 Linux 上运行。

## 命令

程序包含以下命令：

- `start-monitor [partner pid]`：启动监视进程。如果提供了伙伴进程的 PID，监视进程将监视这个进程。否则，监视进程将启动一个新的工作进程，并监视它。
- `start-worker [partner pid]`：启动工作进程。如果提供了伙伴进程的 PID，工作进程将监视这个进程。否则，工作进程将启动一个新的监视进程，并监视它。
- `stop`：停止所有进程。这个命令将发送一个停止信号给所有的监视进程和工作进程。

## 文件

程序使用两个文件来管理进程和通信：

- `/tmp/stop_signal`：这个文件用于传递停止信号。如果这个文件存在，所有的进程都将停止运行。
- `/tmp/myapp_pids`：这个文件包含了所有正在运行的进程的 PID。每个进程在启动时都会把自己的 PID 写入这个文件。`stop`命令将读取这个文件，并向每个 PID 发送一个停止信号。

## 代码示例

以下是程序的主要代码：

```go
// ...

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

        time.Sleep(time.Second)
    }
}

// ...

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

        // Do some work here...
        time.Sleep(time.Second)
    }
}

// ...

func writePIDPeriodically() {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        addPIDToFile(os.Getpid())
    }
}

// ...
```

## 注意事项

在使用 Twin-Process 时，需要注意以下问题：

- PID 文件可能会被删除。程序会每 30 秒检查一次 PID 文件，并在必要时创建一个新的文件。
- 磁盘可能会满。如果磁盘满，程序可能无法写入 PID 文件。你需要确保有足够的磁盘空间，或者定期清理 PID 文件。

