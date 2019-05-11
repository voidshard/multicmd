package main

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"
)

const invalid = "invalid host string, expected: user@host:port:string:taga,tagb"
const TagAll = "all" // special tag that matches everything

// Checks that a line in the hosts config file is ok
//
func validateHostline(hostline string) error {
	if strings.Count(hostline, ":") != 3 {
		return errors.New(invalid)
	}
	if strings.Count(hostline, "@") != 1 {
		return errors.New(invalid)
	}

	return nil
}

// Reads a hosts config line & returns HostConfig struct
//
func parseHostline(hostline string) (*HostConfig, error) {
	err := validateHostline(hostline)
	if err != nil {
		return nil, err
	}

	tmp := strings.SplitN(hostline, "@", 2)
	username := tmp[0]

	tmp = strings.SplitN(tmp[1], ":", 4)

	tags := strings.SplitN(tmp[3], ",", -1)
	tags = append(tags, TagAll)

	return &HostConfig{
		Username:    username,
		Host:        tmp[0] + ":" + tmp[1],
		Credentials: tmp[2],
		Tags:        tags,
	}, nil
}

// util to check if a string is in a list of strings
//
func listContains(toFind string, list []string) bool {
	for _, s := range list {
		if toFind == s {
			return true
		}
	}
	return false
}

// read hostfile looking for desired hosts
//
func obtainHostlist(hostfile string, tags []string) (ls []*HostConfig) {
	f, err := os.Open(hostfile)
	if err != nil {
		log.Fatalln(err)
	}

	hosts := []*HostConfig{}

	scanner := bufio.NewScanner(f)
	for {
		if !scanner.Scan() {
			break
		}

		line := scanner.Text()

		hostData, err := parseHostline(line)
		if err != nil {
			log.Printf("err parsing line [ignoring]: %v", err)
		}

		for _, tag := range hostData.Tags {
			if listContains(tag, tags) {
				hosts = append(hosts, hostData)
				break
			}
		}
	}

	return hosts
}
