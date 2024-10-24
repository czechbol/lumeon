package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/czechbol/lumeon/core/resources"
)

func main() {
	cpu := resources.NewCPU()
	hdd := resources.NewHDD()
	network := resources.NewNetwork()
	memory := resources.NewMemory()

	cpuStats, err := cpu.GetStats()
	if err != nil {
		panic(err)
	}

	hddStats, err := hdd.GetStats()
	if err != nil {
		panic(err)
	}

	networkStats, err := network.GetAllInterfaceStats()
	if err != nil {
		panic(err)
	}

	memStats, err := memory.GetStats()
	if err != nil {
		panic(err)
	}

	// print stats in json format

	// CPU stats
	cpuStatsJSON, err := json.MarshalIndent(cpuStats, "", "  ")
	if err != nil {
		panic(err)
	}

	// HDD stats
	hddStatsJSON, err := json.MarshalIndent(hddStats, "", "  ")
	if err != nil {
		panic(err)
	}

	// Network stats
	networkStatsJSON, err := json.MarshalIndent(networkStats, "", "  ")
	if err != nil {
		panic(err)
	}

	// Memory stats
	memStatsJSON, err := json.MarshalIndent(memStats, "", "  ")
	if err != nil {
		panic(err)
	}

	// save to file
	err = os.WriteFile("cpu_stats.json", cpuStatsJSON, 0644)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("hdd_stats.json", hddStatsJSON, 0644)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("network_stats.json", networkStatsJSON, 0644)
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("mem_stats.json", memStatsJSON, 0644)
	if err != nil {
		panic(err)
	}

	fmt.Println("Stats saved to files")

}
