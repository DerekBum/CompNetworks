package main

import (
	"log"

	"github.com/jlaffaye/ftp"
)

func getAllFiles(c *ftp.ServerConn) string {
	entries, err := c.List("/")
	if err != nil {
		log.Fatal(err)
	}

	ans := ""

	padding := ""
	for _, entry := range entries {
		ans += printDir(c, entry, padding, "/")
	}
	return ans
}

func printDir(c *ftp.ServerConn, entry *ftp.Entry, padding string, path string) string {
	ans := ""
	ans += padding + " " + entry.Name + "\n"
	if entry.Type == ftp.EntryTypeFolder {
		entries, err := c.List(path + entry.Name)
		if err != nil {
			return ""
		}
		for _, subEntry := range entries {
			ans += printDir(c, subEntry, "--"+padding, path+entry.Name+"/")
		}
	}
	return ans
}
