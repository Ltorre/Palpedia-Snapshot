//go:build !windows

package gui

import "fmt"

func Run(version string) {
	fmt.Printf("Palworld Save Scrap %s includes the graphical interface in the Windows build.\n", version)
}
