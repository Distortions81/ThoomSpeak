//go:build windows
// +build windows

package main

import "syscall"

func windowsSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
