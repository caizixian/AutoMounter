package main

import (
	"log"
	"flag"
	"strings"
	"time"
	"os/exec"
	"fmt"
	"github.com/antonholmquist/jason"
)

var (
	interval int
	device_serial map[string]struct{}
	filesystem_UUID map[string]struct{}
	list_devices bool
)

func mount_filesystem(filesystem *jason.Object) {
	name, err := filesystem.GetString("name")
	if err != nil {
		log.Fatal(err)
	}
	mounted, _ := filesystem.GetString("mountpoint")
	// Skip if already mounted
	if mounted == "" {
		log.Println("Mounting", "/dev/" + name)
		exec.Command("mount", "/dev/" + name).CombinedOutput()
	}
}

func init() {
	// Init logging facility
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	device_serial = make(map[string]struct{})
	filesystem_UUID = make(map[string]struct{})

	// Defining command line arguments
	scanning_interval := flag.Int("i", 5, "Scanning Interval")
	devices := flag.String("d", "", "Serial of Devices")
	filesystems := flag.String("f", "", "UUID of Filesystems")
	list_device := flag.Bool("l", false, "List Current Devices")
	flag.Parse()

	// Parsing command line arguments
	interval = *scanning_interval
	list_devices = *list_device
	for _, device := range strings.Split(*devices, ",") {
		if len(strings.TrimSpace(device)) > 0 {
			device_serial[device] = struct{}{}
		}
	}

	for _, filesystem := range strings.Split(*filesystems, ",") {
		if len(strings.TrimSpace(filesystem)) > 0 {
			filesystem_UUID[filesystem] = struct{}{}
		}
	}
}

func main() {
	if list_devices {
		out, err := exec.Command("lsblk", "-o", "NAME,SERIAL,UUID,SIZE,TYPE,MOUNTPOINT").Output()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(out))
		return
	}
	for {
		// Read current block devices by using lsblk
		out, err := exec.Command("lsblk", "-J", "-o", "NAME,UUID,SERIAL,MOUNTPOINT").Output()
		if err != nil {
			log.Fatal(err)
		}
		lsblk, err := jason.NewObjectFromBytes(out)
		if err != nil {
			log.Fatal(err)
		}
		// Parse output
		blockdevices, err := lsblk.GetObjectArray("blockdevices")
		if err != nil {
			log.Fatal(err)
		}
		for _, blockdevice := range blockdevices {
			serial, _ := blockdevice.GetString("serial")

			// All children of this device needs to be mounted
			if _, ok := device_serial[serial]; ok {
				filesystems, err := blockdevice.GetObjectArray("children")
				if err != nil {
					log.Fatal(err)
				}
				for _, filesystem := range filesystems {
					mount_filesystem(filesystem)
				}
			} else {
				filesystems, _ := blockdevice.GetObjectArray("children")
				for _, filesystem := range filesystems {
					uuid, _ := filesystem.GetString("uuid")
					// Does this device contains filesystems that needs to be mounted
					if _, ok := filesystem_UUID[uuid]; ok {
						mount_filesystem(filesystem)
					}
				}
			}
		}
		time.Sleep(time.Second * time.Duration(interval))
	}
}
