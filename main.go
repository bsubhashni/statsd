package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	interval int64
}

func getCPUSample() (idle, total uint64) {
	contents, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return
	}
	lines := strings.Split(string(contents), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if fields[0] == "cpu" {
			numFields := len(fields)
			for i := 1; i < numFields; i++ {
				val, err := strconv.ParseUint(fields[i], 10, 64)
				if err != nil {
					fmt.Println("Error: ", i, fields[i], err)
				}
				total += val
				if i == 4 {
					idle = val
				}
			}
			return
		}
	}
	return
}

func getMemoryUsage() (inuse uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc
}

func SendStats(namespace string, conn net.Conn) {
	cpuns := namespace + ".stats.cpu"
	memns := namespace + ".stats.memory"

	idle0, total0 := getCPUSample()
	time.Sleep(3 * time.Second)
	idle1, total1 := getCPUSample()
	idleTicks := float64(idle1 - idle0)
	totalTicks := float64(total1 - total0)
	cpuUsage := (100 * (totalTicks - idleTicks) / totalTicks)

	memUsage := getMemoryUsage()

	cpuinfo := cpuns + " " + fmt.Sprintf("%.6f %d", cpuUsage, time.Now().Unix())
	meminfo := memns + " " + fmt.Sprintf("%d %d\n", memUsage, time.Now().Unix())
	fmt.Printf("%s \n", cpuinfo)
	fmt.Printf("%s", meminfo)
	if _, err := conn.Write([]byte(meminfo)); err != nil {
		log.Fatalf("Error writing to server %v", err)
	}
}

func main() {
	server := flag.String("server", "", "Graphite Server and carbon port to connect to")
	namespace := flag.String("Namespace", "", "This is the graphite namespace")

	conn, err := net.Dial("tcp", *server)
	if err != nil {
		log.Fatalf("Cannot connect to the server %v", err)
	}
	defer conn.Close()
	for {
		SendStats(*namespace, conn)
		time.Sleep(1 * time.Second)
	}
}
