// Copyright (C) 2014 Nippon Telegraph and Telephone Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"./server"
	"fmt"
	"github.com/jessevdk/go-flags"
	"github.com/osrg/gobgp/config"
	"github.com/osrg/gobgp/packet"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)

	var opts struct {
		ConfigFile string `short:"f" long:"config-file" description:"specifying a config file"`
	}

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.ConfigFile == "" {
		opts.ConfigFile = "gobgpd.conf"
	}

	configCh := make(chan config.BgpType)
	reloadCh := make(chan bool)
	go config.ReadConfigfileServe(opts.ConfigFile, configCh, reloadCh)
	reloadCh <- true

	bgpServer := server.NewBgpServer(bgp.BGP_PORT)
	go bgpServer.Serve()

	var bgpConfig *config.BgpType = nil
	for {
		select {
		case newConfig := <-configCh:
			var added []config.NeighborType
			var deleted []config.NeighborType

			if bgpConfig == nil {
				bgpServer.SetGlobalType(newConfig.Global)
				bgpConfig = &newConfig
				added = newConfig.NeighborList
				deleted = []config.NeighborType{}
			} else {
				_, added, deleted = config.UpdateConfig(bgpConfig, &newConfig)
			}

			for _, p := range added {
				bgpServer.PeerAdd(p)
			}
			for _, p := range deleted {
				bgpServer.PeerDelete(p)
			}
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				fmt.Println("relaod the config file")
				reloadCh <- true
			}
		}
	}
}
