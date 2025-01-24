package main

import (
	"fmt"

	"github.com:it-laborato/MDM_Lab/orbit/pkg/lvm"
)

func main() {
	disk, err := lvm.FindRootDisk()
	if err != nil {
		panic(err)
	}
	fmt.Println("Root Partition:", disk)
}
