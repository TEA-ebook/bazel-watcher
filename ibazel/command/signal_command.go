// Copyright 2017 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"bytes"
	"os"

	"github.com/bazelbuild/bazel-watcher/ibazel/log"
	"github.com/bazelbuild/bazel-watcher/ibazel/process_group"
)

type signalCommand struct {
	target      string
	startupArgs []string
	bazelArgs   []string
	args        []string
	useKill     bool

	pg    process_group.ProcessGroup
}

// SignalCommand is an alternate mode for starting a command. In this mode the
// command will be notified by SIGHUP that the source files have changed.
func SignalCommand(startupArgs []string, bazelArgs []string, target string, args []string, useKill bool) Command {
	return &signalCommand{
		startupArgs: startupArgs,
		target:      target,
		bazelArgs:   bazelArgs,
		args:        args,
		useKill:     useKill,
	}
}

func (c *signalCommand) Terminate() {
	if c.pg != nil && !subprocessRunning(c.pg.RootProcess()) {
		return
	}

	if c.useKill {
		c.pg.Kill()
	} else {
		c.pg.Terminate()
	}
	c.pg.Wait()
	c.pg.Close()
	c.pg = nil
}

func (c *signalCommand) Start() (*bytes.Buffer, error) {
	b := bazelNew()
	b.SetStartupArgs(c.startupArgs)
	b.SetArguments(c.bazelArgs)

	b.WriteToStderr(true)
	b.WriteToStdout(true)

	var outputBuffer *bytes.Buffer
	outputBuffer, c.pg = start(b, c.target, c.args)

	c.pg.RootProcess().Env = append(os.Environ(), "IBAZEL_SIGNAL_CHANGES=y")

	if err := c.pg.Start(); err != nil {
		log.Errorf("Error starting process: %v", err)
		return outputBuffer, err
	}
	log.Log("Starting...")
	return outputBuffer, nil
}

func (c *signalCommand) NotifyOfChanges() *bytes.Buffer {
	if !c.IsSubprocessRunning() {
		outputBuffer, _ := c.Start()
		return outputBuffer
	}

	b := bazelNew()
	b.SetStartupArgs(c.startupArgs)
	b.SetArguments(c.bazelArgs)

	b.WriteToStderr(true)
	b.WriteToStdout(true)

	outputBuffer, _ := b.Build(c.target)

	c.pg.RefreshSignal()
	return outputBuffer
}

func (c *signalCommand) IsSubprocessRunning() bool {
	return c.pg != nil && subprocessRunning(c.pg.RootProcess())
}
