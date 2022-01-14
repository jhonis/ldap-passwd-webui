//go:build windows
// +build windows

package main

import (
	"fmt"
	"ldap-passwd-webui/service"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

func main() {
	const svcName = "ldap-password-webui"

	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Failed to determine if we are running in service: %v", err)
	}
	if inService {
		service.RunService(svcName, false)
		return
	}

	if len(os.Args) < 2 {
		usage("No command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		service.RunService(svcName, true)
		return
	case "install":
		err = service.InstallService(svcName, "Web interface for Active Directory users to change their passwords")
	case "remove":
		err = service.RemoveService(svcName)
	case "start":
		err = service.StartService(svcName)
	case "stop":
		err = service.ControlService(svcName, svc.Stop, svc.Stopped)
	case "pause":
		err = service.ControlService(svcName, svc.Pause, svc.Paused)
	case "continue":
		err = service.ControlService(svcName, svc.Continue, svc.Running)
	default:
		usage(fmt.Sprintf("Invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("Failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
